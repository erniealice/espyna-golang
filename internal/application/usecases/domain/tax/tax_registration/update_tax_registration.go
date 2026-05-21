package tax_registration

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// UpdateTaxRegistrationRepositories groups repository dependencies.
type UpdateTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// UpdateTaxRegistrationServices groups service dependencies.
type UpdateTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateTaxRegistrationUseCase handles updating a tax_registration.
type UpdateTaxRegistrationUseCase struct {
	repositories UpdateTaxRegistrationRepositories
	services     UpdateTaxRegistrationServices
}

// NewUpdateTaxRegistrationUseCase creates a new UpdateTaxRegistrationUseCase.
func NewUpdateTaxRegistrationUseCase(repositories UpdateTaxRegistrationRepositories, services UpdateTaxRegistrationServices) *UpdateTaxRegistrationUseCase {
	return &UpdateTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the update tax_registration operation.
func (uc *UpdateTaxRegistrationUseCase) Execute(ctx context.Context, req *taxregistrationpb.UpdateTaxRegistrationRequest) (*taxregistrationpb.UpdateTaxRegistrationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.id_required", "Tax Registration ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistration.UpdateTaxRegistration(ctx, req)
}
