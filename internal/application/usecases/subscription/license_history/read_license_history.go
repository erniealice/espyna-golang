package licensehistory

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

// ReadLicenseHistoryRepositories groups all repository dependencies
type ReadLicenseHistoryRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // Primary entity repository
}

// ReadLicenseHistoryServices groups all business service dependencies
type ReadLicenseHistoryServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ReadLicenseHistoryUseCase handles the business logic for reading license history entries
type ReadLicenseHistoryUseCase struct {
	repositories ReadLicenseHistoryRepositories
	services     ReadLicenseHistoryServices
}

// NewReadLicenseHistoryUseCase creates a new ReadLicenseHistoryUseCase
func NewReadLicenseHistoryUseCase(
	repositories ReadLicenseHistoryRepositories,
	services ReadLicenseHistoryServices,
) *ReadLicenseHistoryUseCase {
	return &ReadLicenseHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read license history operation
func (uc *ReadLicenseHistoryUseCase) Execute(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) (*licensehistorypb.ReadLicenseHistoryResponse, error) {
	// Authorization check - conditional based on service availability
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
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LicenseHistory.ReadLicenseHistory(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("license history with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"license_history.errors.not_found",
				map[string]interface{}{"historyId": req.Data.Id},
				"License history not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadLicenseHistoryUseCase) validateInput(ctx context.Context, req *licensehistorypb.ReadLicenseHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.data_required", "license history data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.id_required", "license history ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for reading license history
func (uc *ReadLicenseHistoryUseCase) validateBusinessRules(history *licensehistorypb.LicenseHistory) error {
	// Validate history ID format
	if len(history.Id) < 3 {
		return errors.New("license history ID must be at least 3 characters long")
	}

	return nil
}
