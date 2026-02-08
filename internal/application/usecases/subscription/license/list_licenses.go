package license

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

// ListLicensesRepositories groups all repository dependencies
type ListLicensesRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// ListLicensesServices groups all business service dependencies
type ListLicensesServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ListLicensesUseCase handles the business logic for listing licenses
type ListLicensesUseCase struct {
	repositories ListLicensesRepositories
	services     ListLicensesServices
}

// NewListLicensesUseCase creates a new ListLicensesUseCase
func NewListLicensesUseCase(
	repositories ListLicensesRepositories,
	services ListLicensesServices,
) *ListLicensesUseCase {
	return &ListLicensesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list licenses operation
func (uc *ListLicensesUseCase) Execute(ctx context.Context, req *licensepb.ListLicensesRequest) (*licensepb.ListLicensesResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityLicense, ports.ActionRead)
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

	// Business rule validation
	if err := uc.validateBusinessRules(req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.License.ListLicenses(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLicensesUseCase) validateInput(ctx context.Context, req *licensepb.ListLicensesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing licenses
func (uc *ListLicensesUseCase) validateBusinessRules(req *licensepb.ListLicensesRequest) error {
	// No specific business rules for listing licenses
	return nil
}
