package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	licensehistory "leapfor.xyz/espyna/internal/application/usecases/subscription/license_history"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// AssignLicenseRepositories groups all repository dependencies
type AssignLicenseRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For assigned_count updates
	Client       clientpb.ClientDomainServiceServer             // For assignee validation
}

// AssignLicenseServices groups all business service dependencies
type AssignLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// AssignLicenseUseCase handles the business logic for assigning licenses
type AssignLicenseUseCase struct {
	repositories         AssignLicenseRepositories
	services             AssignLicenseServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewAssignLicenseUseCase creates a new AssignLicenseUseCase
func NewAssignLicenseUseCase(
	repositories AssignLicenseRepositories,
	services AssignLicenseServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *AssignLicenseUseCase {
	return &AssignLicenseUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the assign license operation
func (uc *AssignLicenseUseCase) Execute(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the assign license operation within a transaction
func (uc *AssignLicenseUseCase) executeWithTransaction(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
	var result *licensepb.AssignLicenseResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.assignment_failed", "license assignment failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core assign license operation
func (uc *AssignLicenseUseCase) executeCore(ctx context.Context, req *licensepb.AssignLicenseRequest) (*licensepb.AssignLicenseResponse, error) {
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

	// Check if license is already assigned
	if license.AssigneeId != nil && *license.AssigneeId != "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.already_assigned", "license is already assigned [DEFAULT]"))
	}

	// Check if license is in a valid state for assignment
	if license.Status == licensepb.LicenseStatus_LICENSE_STATUS_REVOKED ||
		license.Status == licensepb.LicenseStatus_LICENSE_STATUS_EXPIRED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.invalid_status_for_assignment", "license cannot be assigned in its current status [DEFAULT]"))
	}

	// Store previous status for history
	previousStatus := license.Status

	// Update license with assignment
	now := time.Now()
	license.AssigneeId = &req.AssigneeId
	license.AssigneeType = &req.AssigneeType
	license.AssigneeName = req.AssigneeName
	license.AssignedBy = &req.AssignedBy
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

	// Create history entry
	if uc.createHistoryUseCase != nil {
		historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
			Data: &licensehistorypb.LicenseHistory{
				LicenseId:           updatedLicense.Id,
				Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_ASSIGNED,
				AssigneeId:          updatedLicense.AssigneeId,
				AssigneeType:        updatedLicense.AssigneeType,
				AssigneeName:        updatedLicense.AssigneeName,
				PerformedBy:         req.AssignedBy,
				Reason:              req.Reason,
				LicenseStatusBefore: previousStatus,
				LicenseStatusAfter:  updatedLicense.Status,
			},
		}
		_, _ = uc.createHistoryUseCase.Execute(ctx, historyReq)
	}

	// Update subscription assigned count
	uc.updateSubscriptionAssignedCount(ctx, updatedLicense.SubscriptionId, 1)

	return &licensepb.AssignLicenseResponse{
		License: updatedLicense,
		Success: true,
	}, nil
}

// updateSubscriptionAssignedCount updates the subscription's assigned_count
func (uc *AssignLicenseUseCase) updateSubscriptionAssignedCount(ctx context.Context, subscriptionID string, delta int32) {
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
func (uc *AssignLicenseUseCase) validateInput(ctx context.Context, req *licensepb.AssignLicenseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.LicenseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.id_required", "license ID is required [DEFAULT]"))
	}
	if req.AssigneeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.assignee_id_required", "assignee ID is required [DEFAULT]"))
	}
	if req.AssigneeType == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.assignee_type_required", "assignee type is required [DEFAULT]"))
	}
	if req.AssignedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.assigned_by_required", "assigned_by is required [DEFAULT]"))
	}
	return nil
}
