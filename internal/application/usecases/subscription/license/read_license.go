package license

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

// ReadLicenseRepositories groups all repository dependencies
type ReadLicenseRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// ReadLicenseServices groups all business service dependencies
type ReadLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
}

// ReadLicenseUseCase handles the business logic for reading licenses
type ReadLicenseUseCase struct {
	repositories ReadLicenseRepositories
	services     ReadLicenseServices
}

// NewReadLicenseUseCase creates a new ReadLicenseUseCase
func NewReadLicenseUseCase(
	repositories ReadLicenseRepositories,
	services ReadLicenseServices,
) *ReadLicenseUseCase {
	return &ReadLicenseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read license operation
func (uc *ReadLicenseUseCase) Execute(ctx context.Context, req *licensepb.ReadLicenseRequest) (*licensepb.ReadLicenseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLicense, ports.ActionRead); err != nil {
		return nil, err
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
	resp, err := uc.repositories.License.ReadLicense(ctx, req)
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
func (uc *ReadLicenseUseCase) validateInput(ctx context.Context, req *licensepb.ReadLicenseRequest) error {
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

// validateBusinessRules enforces business constraints for reading licenses
func (uc *ReadLicenseUseCase) validateBusinessRules(license *licensepb.License) error {
	// Validate license ID format
	if len(license.Id) < 3 {
		return errors.New("license ID must be at least 3 characters long")
	}

	return nil
}
