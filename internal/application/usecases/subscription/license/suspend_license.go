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
)

// SuspendLicenseRepositories groups all repository dependencies
type SuspendLicenseRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// SuspendLicenseServices groups all business service dependencies
type SuspendLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// SuspendLicenseUseCase handles the business logic for suspending licenses
type SuspendLicenseUseCase struct {
	repositories         SuspendLicenseRepositories
	services             SuspendLicenseServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewSuspendLicenseUseCase creates a new SuspendLicenseUseCase
func NewSuspendLicenseUseCase(
	repositories SuspendLicenseRepositories,
	services SuspendLicenseServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *SuspendLicenseUseCase {
	return &SuspendLicenseUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the suspend license operation
func (uc *SuspendLicenseUseCase) Execute(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the suspend license operation within a transaction
func (uc *SuspendLicenseUseCase) executeWithTransaction(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
	var result *licensepb.SuspendLicenseResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.suspension_failed", "license suspension failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core suspend license operation
func (uc *SuspendLicenseUseCase) executeCore(ctx context.Context, req *licensepb.SuspendLicenseRequest) (*licensepb.SuspendLicenseResponse, error) {
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

	// Check if license is already suspended
	if license.Status == licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.already_suspended", "license is already suspended [DEFAULT]"))
	}

	// Check if license is in a valid state for suspension
	if license.Status == licensepb.LicenseStatus_LICENSE_STATUS_REVOKED ||
		license.Status == licensepb.LicenseStatus_LICENSE_STATUS_EXPIRED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.invalid_status_for_suspension", "license cannot be suspended in its current status [DEFAULT]"))
	}

	// Store previous status for history
	previousStatus := license.Status

	// Update license status
	now := time.Now()
	license.Status = licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED
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
				Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_SUSPENDED,
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

	return &licensepb.SuspendLicenseResponse{
		License: updatedLicense,
		Success: true,
	}, nil
}

// validateInput validates the input request
func (uc *SuspendLicenseUseCase) validateInput(ctx context.Context, req *licensepb.SuspendLicenseRequest) error {
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
