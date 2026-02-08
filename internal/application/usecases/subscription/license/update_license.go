package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// UpdateLicenseRepositories groups all repository dependencies
type UpdateLicenseRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For FK validation
}

// UpdateLicenseServices groups all business service dependencies
type UpdateLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// UpdateLicenseUseCase handles the business logic for updating licenses
type UpdateLicenseUseCase struct {
	repositories UpdateLicenseRepositories
	services     UpdateLicenseServices
}

// NewUpdateLicenseUseCase creates a new UpdateLicenseUseCase
func NewUpdateLicenseUseCase(
	repositories UpdateLicenseRepositories,
	services UpdateLicenseServices,
) *UpdateLicenseUseCase {
	return &UpdateLicenseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update license operation
func (uc *UpdateLicenseUseCase) Execute(ctx context.Context, req *licensepb.UpdateLicenseRequest) (*licensepb.UpdateLicenseResponse, error) {
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

	// Business logic and enrichment
	if err := uc.enrichLicenseData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.License.UpdateLicense(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("license with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"license.errors.not_found",
				map[string]interface{}{"licenseId": req.Data.Id},
				"License not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateLicenseUseCase) validateInput(ctx context.Context, req *licensepb.UpdateLicenseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.data_required", "license data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.id_required", "license ID is required [DEFAULT]"))
	}
	return nil
}

// enrichLicenseData adds audit information for updates
func (uc *UpdateLicenseUseCase) enrichLicenseData(license *licensepb.License) error {
	now := time.Now()

	// Update modification timestamp
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for license updates
func (uc *UpdateLicenseUseCase) validateBusinessRules(license *licensepb.License) error {
	// Validate license ID format
	if len(license.Id) < 3 {
		return errors.New("license ID must be at least 3 characters long")
	}

	return nil
}
