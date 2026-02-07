package licensehistory

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
)

// ListLicenseHistoryRepositories groups all repository dependencies
type ListLicenseHistoryRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // Primary entity repository
}

// ListLicenseHistoryServices groups all business service dependencies
type ListLicenseHistoryServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ListLicenseHistoryUseCase handles the business logic for listing license history entries
type ListLicenseHistoryUseCase struct {
	repositories ListLicenseHistoryRepositories
	services     ListLicenseHistoryServices
}

// NewListLicenseHistoryUseCase creates a new ListLicenseHistoryUseCase
func NewListLicenseHistoryUseCase(
	repositories ListLicenseHistoryRepositories,
	services ListLicenseHistoryServices,
) *ListLicenseHistoryUseCase {
	return &ListLicenseHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list license history operation
func (uc *ListLicenseHistoryUseCase) Execute(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) (*licensehistorypb.ListLicenseHistoryResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.errors.authorization_failed", "Authorization failed for license history [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityLicense, ports.ActionRead)
		hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.errors.authorization_failed", "Authorization failed for license history [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		if !hasPerm {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.errors.authorization_failed", "Authorization failed for license history [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LicenseHistory.ListLicenseHistory(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLicenseHistoryUseCase) validateInput(ctx context.Context, req *licensehistorypb.ListLicenseHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.request_required", "request is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing license history
func (uc *ListLicenseHistoryUseCase) validateBusinessRules(req *licensehistorypb.ListLicenseHistoryRequest) error {
	// If license_id is provided, validate format
	if req.LicenseId != nil && len(*req.LicenseId) > 0 && len(*req.LicenseId) < 3 {
		return errors.New("license ID must be at least 3 characters long")
	}

	return nil
}
