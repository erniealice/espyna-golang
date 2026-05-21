package tax_registration

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// DeleteTaxRegistrationRepositories groups repository dependencies.
type DeleteTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// DeleteTaxRegistrationServices groups service dependencies.
type DeleteTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteTaxRegistrationUseCase handles deleting a tax_registration.
type DeleteTaxRegistrationUseCase struct {
	repositories DeleteTaxRegistrationRepositories
	services     DeleteTaxRegistrationServices
}

// NewDeleteTaxRegistrationUseCase creates a new DeleteTaxRegistrationUseCase.
func NewDeleteTaxRegistrationUseCase(repositories DeleteTaxRegistrationRepositories, services DeleteTaxRegistrationServices) *DeleteTaxRegistrationUseCase {
	return &DeleteTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the delete tax_registration operation.
func (uc *DeleteTaxRegistrationUseCase) Execute(ctx context.Context, req *taxregistrationpb.DeleteTaxRegistrationRequest) (*taxregistrationpb.DeleteTaxRegistrationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.id_required", "Tax Registration ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistration.DeleteTaxRegistration(ctx, req)
}
