package tax_registration

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// ReadTaxRegistrationRepositories groups repository dependencies.
type ReadTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// ReadTaxRegistrationServices groups service dependencies.
type ReadTaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadTaxRegistrationUseCase handles reading a tax_registration.
type ReadTaxRegistrationUseCase struct {
	repositories ReadTaxRegistrationRepositories
	services     ReadTaxRegistrationServices
}

// NewReadTaxRegistrationUseCase creates a new ReadTaxRegistrationUseCase.
func NewReadTaxRegistrationUseCase(repositories ReadTaxRegistrationRepositories, services ReadTaxRegistrationServices) *ReadTaxRegistrationUseCase {
	return &ReadTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_registration operation.
func (uc *ReadTaxRegistrationUseCase) Execute(ctx context.Context, req *taxregistrationpb.ReadTaxRegistrationRequest) (*taxregistrationpb.ReadTaxRegistrationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistration, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration.validation.id_required", "Tax Registration ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistration.ReadTaxRegistration(ctx, req)
}
