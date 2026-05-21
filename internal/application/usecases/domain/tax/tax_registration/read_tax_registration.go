package tax_registration

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// ReadTaxRegistrationRepositories groups repository dependencies.
type ReadTaxRegistrationRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// ReadTaxRegistrationServices groups service dependencies.
type ReadTaxRegistrationServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxRegistration, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_registration.validation.id_required", "Tax Registration ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistration.ReadTaxRegistration(ctx, req)
}
