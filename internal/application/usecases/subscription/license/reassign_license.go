package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensehistory "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/license_history"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// ReassignLicenseRepositories groups all repository dependencies
type ReassignLicenseRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For assigned_count updates
	Client       clientpb.ClientDomainServiceServer             // For new assignee validation
}

// ReassignLicenseServices groups all business service dependencies
type ReassignLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ReassignLicenseUseCase handles the business logic for reassigning licenses
type ReassignLicenseUseCase struct {
	repositories         ReassignLicenseRepositories
	services             ReassignLicenseServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewReassignLicenseUseCase creates a new ReassignLicenseUseCase
func NewReassignLicenseUseCase(
	repositories ReassignLicenseRepositories,
	services ReassignLicenseServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *ReassignLicenseUseCase {
	return &ReassignLicenseUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the reassign license operation
func (uc *ReassignLicenseUseCase) Execute(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLicense, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the reassign license operation within a transaction
func (uc *ReassignLicenseUseCase) executeWithTransaction(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {
	var result *licensepb.ReassignLicenseResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.reassignment_failed", "license reassignment failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core reassign license operation
func (uc *ReassignLicenseUseCase) executeCore(ctx context.Context, req *licensepb.ReassignLicenseRequest) (*licensepb.ReassignLicenseResponse, error) {

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Read current license
	readResp, err := uc.repositories.License.ReadLicense(ctx, &licensepb.ReadLicenseRequest{
		Data: &licensepb.License{Id: req.LicenseId},
	})
	if err != nil || readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.not_found", "license not found [DEFAULT]"))
	}

	license := readResp.Data[0]

	// Check if license is in a valid state for reassignment
	if license.Status == licensepb.LicenseStatus_LICENSE_STATUS_REVOKED ||
		license.Status == licensepb.LicenseStatus_LICENSE_STATUS_EXPIRED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.invalid_status_for_reassignment", "license cannot be reassigned in its current status [DEFAULT]"))
	}

	// Check if trying to reassign to the same person
	if license.AssigneeId != nil && *license.AssigneeId == req.NewAssigneeId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.same_assignee", "cannot reassign to the same assignee [DEFAULT]"))
	}

	// Store previous assignee info for history
	previousAssigneeId := license.AssigneeId
	previousAssigneeType := license.AssigneeType
	previousAssigneeName := license.AssigneeName
	previousStatus := license.Status

	// Update license with new assignment
	now := time.Now()
	license.AssigneeId = &req.NewAssigneeId
	license.AssigneeType = &req.NewAssigneeType
	license.AssigneeName = req.NewAssigneeName
	license.AssignedBy = &req.PerformedBy
	license.DateAssigned = &[]int64{now.UnixMilli()}[0]
	dateAssignedStr := now.Format(time.RFC3339)
	license.DateAssignedString = &dateAssignedStr
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Update the license
	updateResp, err := uc.repositories.License.UpdateLicense(ctx, &licensepb.UpdateLicenseRequest{
		Data: license,
	})
	if err != nil {
		return nil, err
	}

	updatedLicense := license
	if updateResp != nil && len(updateResp.Data) > 0 {
		updatedLicense = updateResp.Data[0]
	}

	// Create history entry with both previous and new assignee info
	if uc.createHistoryUseCase != nil {
		historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
			Data: &licensehistorypb.LicenseHistory{
				LicenseId:            updatedLicense.Id,
				Action:               licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_REASSIGNED,
				AssigneeId:           updatedLicense.AssigneeId,
				AssigneeType:         updatedLicense.AssigneeType,
				AssigneeName:         updatedLicense.AssigneeName,
				PreviousAssigneeId:   previousAssigneeId,
				PreviousAssigneeType: previousAssigneeType,
				PreviousAssigneeName: previousAssigneeName,
				PerformedBy:          req.PerformedBy,
				Reason:               req.Reason,
				LicenseStatusBefore:  previousStatus,
				LicenseStatusAfter:   updatedLicense.Status,
			},
		}
		_, _ = uc.createHistoryUseCase.Execute(ctx, historyReq)
	}

	// Note: For reassignment, the assigned_count doesn't change since it's a transfer

	return &licensepb.ReassignLicenseResponse{
		License: updatedLicense,
		Success: true,
	}, nil
}

// validateInput validates the input request
func (uc *ReassignLicenseUseCase) validateInput(ctx context.Context, req *licensepb.ReassignLicenseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.LicenseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.id_required", "license ID is required [DEFAULT]"))
	}
	if req.NewAssigneeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.new_assignee_id_required", "new assignee ID is required [DEFAULT]"))
	}
	if req.NewAssigneeType == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.new_assignee_type_required", "new assignee type is required [DEFAULT]"))
	}
	if req.PerformedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.performed_by_required", "performed_by is required [DEFAULT]"))
	}
	return nil
}
