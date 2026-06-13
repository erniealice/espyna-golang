package license

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

// ReadLicenseRepositories groups all repository dependencies
type ReadLicenseRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// ReadLicenseServices groups all business service dependencies
type ReadLicenseServices struct {
	Authorizer ports.Authorizer // RBAC and permissions
	Transactor ports.Transactor // Database transactions
	Translator ports.Translator // i18n error messages
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.License,
		Action: entityid.ActionRead,
	}); err != nil {
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
				uc.services.Translator,
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "license.validation.data_required", "license data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "license.validation.id_required", "license ID is required [DEFAULT]"))
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
