package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	licensehistory "leapfor.xyz/espyna/internal/application/usecases/subscription/license_history"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// RevokeLicenseAssignmentRepositories groups all repository dependencies
type RevokeLicenseAssignmentRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For assigned_count updates
}

// RevokeLicenseAssignmentServices groups all business service dependencies
type RevokeLicenseAssignmentServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// RevokeLicenseAssignmentUseCase handles the business logic for revoking license assignments
type RevokeLicenseAssignmentUseCase struct {
	repositories         RevokeLicenseAssignmentRepositories
	services             RevokeLicenseAssignmentServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewRevokeLicenseAssignmentUseCase creates a new RevokeLicenseAssignmentUseCase
func NewRevokeLicenseAssignmentUseCase(
	repositories RevokeLicenseAssignmentRepositories,
	services RevokeLicenseAssignmentServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *RevokeLicenseAssignmentUseCase {
	return &RevokeLicenseAssignmentUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the revoke license assignment operation
func (uc *RevokeLicenseAssignmentUseCase) Execute(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the revoke assignment operation within a transaction
func (uc *RevokeLicenseAssignmentUseCase) executeWithTransaction(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	var result *licensepb.RevokeLicenseAssignmentResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.revocation_failed", "license revocation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core revoke license assignment operation
func (uc *RevokeLicenseAssignmentUseCase) executeCore(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) (*licensepb.RevokeLicenseAssignmentResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityLicense, ports.ActionUpdate)
		hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		if !hasPerm {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

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

	// Check if license is assigned
	if license.AssigneeId == nil || *license.AssigneeId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.not_assigned", "license is not assigned [DEFAULT]"))
	}

	// Store previous assignee info for history
	previousAssigneeId := license.AssigneeId
	previousAssigneeType := license.AssigneeType
	previousAssigneeName := license.AssigneeName
	previousStatus := license.Status

	// Clear assignment
	now := time.Now()
	emptyString := ""
	license.AssigneeId = &emptyString
	license.AssigneeType = nil
	license.AssigneeName = nil
	license.AssignedBy = nil
	license.DateAssigned = nil
	license.DateAssignedString = nil
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_PENDING
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
				LicenseId:            updatedLicense.Id,
				Action:               licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_REVOKED,
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

	// Update subscription assigned count
	uc.updateSubscriptionAssignedCount(ctx, updatedLicense.SubscriptionId, -1)

	return &licensepb.RevokeLicenseAssignmentResponse{
		License: updatedLicense,
		Success: true,
	}, nil
}

// updateSubscriptionAssignedCount updates the subscription's assigned_count
func (uc *RevokeLicenseAssignmentUseCase) updateSubscriptionAssignedCount(ctx context.Context, subscriptionID string, delta int32) {
	subResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: subscriptionID},
	})
	if err != nil || subResp == nil || len(subResp.Data) == 0 {
		return
	}

	sub := subResp.Data[0]
	var currentCount int32
	if sub.AssignedCount != nil {
		currentCount = *sub.AssignedCount
	}
	newCount := currentCount + delta
	if newCount < 0 {
		newCount = 0
	}
	sub.AssignedCount = &newCount

	if sub.Quantity != nil {
		availableCount := *sub.Quantity - newCount
		sub.AvailableCount = &availableCount
	}

	_, _ = uc.repositories.Subscription.UpdateSubscription(ctx, &subscriptionpb.UpdateSubscriptionRequest{
		Data: sub,
	})
}

// validateInput validates the input request
func (uc *RevokeLicenseAssignmentUseCase) validateInput(ctx context.Context, req *licensepb.RevokeLicenseAssignmentRequest) error {
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
