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
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

// ReactivateLicenseRepositories groups all repository dependencies
type ReactivateLicenseRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// ReactivateLicenseServices groups all business service dependencies
type ReactivateLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ReactivateLicenseUseCase handles the business logic for reactivating suspended licenses
type ReactivateLicenseUseCase struct {
	repositories         ReactivateLicenseRepositories
	services             ReactivateLicenseServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewReactivateLicenseUseCase creates a new ReactivateLicenseUseCase
func NewReactivateLicenseUseCase(
	repositories ReactivateLicenseRepositories,
	services ReactivateLicenseServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *ReactivateLicenseUseCase {
	return &ReactivateLicenseUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the reactivate license operation
func (uc *ReactivateLicenseUseCase) Execute(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
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

// executeWithTransaction performs the reactivate license operation within a transaction
func (uc *ReactivateLicenseUseCase) executeWithTransaction(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {
	var result *licensepb.ReactivateLicenseResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.reactivation_failed", "license reactivation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core reactivate license operation
func (uc *ReactivateLicenseUseCase) executeCore(ctx context.Context, req *licensepb.ReactivateLicenseRequest) (*licensepb.ReactivateLicenseResponse, error) {

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

	// Check if license is suspended
	if license.Status != licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.not_suspended", "license is not suspended [DEFAULT]"))
	}

	// Store previous status for history
	previousStatus := license.Status

	// Determine new status - ACTIVE if assigned, PENDING if not
	var newStatus licensepb.LicenseStatus
	if license.AssigneeId != nil && *license.AssigneeId != "" {
		newStatus = licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE
	} else {
		newStatus = licensepb.LicenseStatus_LICENSE_STATUS_PENDING
	}

	// Update license status
	now := time.Now()
	license.Status = newStatus
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

	// Create history entry
	if uc.createHistoryUseCase != nil {
		historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
			Data: &licensehistorypb.LicenseHistory{
				LicenseId:           updatedLicense.Id,
				Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_REACTIVATED,
				AssigneeId:          updatedLicense.AssigneeId,
				AssigneeType:        updatedLicense.AssigneeType,
				AssigneeName:        updatedLicense.AssigneeName,
				PerformedBy:         req.PerformedBy,
				Reason:              req.Reason,
				LicenseStatusBefore: previousStatus,
				LicenseStatusAfter:  updatedLicense.Status,
			},
		}
		_, _ = uc.createHistoryUseCase.Execute(ctx, historyReq)
	}

	return &licensepb.ReactivateLicenseResponse{
		License: updatedLicense,
		Success: true,
	}, nil
}

// validateInput validates the input request
func (uc *ReactivateLicenseUseCase) validateInput(ctx context.Context, req *licensepb.ReactivateLicenseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.LicenseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.id_required", "license ID is required [DEFAULT]"))
	}
	if req.PerformedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.performed_by_required", "performed_by is required [DEFAULT]"))
	}
	return nil
}
