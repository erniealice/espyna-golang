package tax_registration

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
)

// ListTaxRegistrationsRepositories groups repository dependencies.
type ListTaxRegistrationsRepositories struct {
	TaxRegistration taxregistrationpb.TaxRegistrationDomainServiceServer
}

// ListTaxRegistrationsServices groups service dependencies.
type ListTaxRegistrationsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListTaxRegistrationsUseCase handles listing tax registrations.
type ListTaxRegistrationsUseCase struct {
	repositories ListTaxRegistrationsRepositories
	services     ListTaxRegistrationsServices
}

// NewListTaxRegistrationsUseCase creates a new ListTaxRegistrationsUseCase.
func NewListTaxRegistrationsUseCase(repositories ListTaxRegistrationsRepositories, services ListTaxRegistrationsServices) *ListTaxRegistrationsUseCase {
	return &ListTaxRegistrationsUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax_registrations operation.
func (uc *ListTaxRegistrationsUseCase) Execute(ctx context.Context, req *taxregistrationpb.ListTaxRegistrationsRequest) (*taxregistrationpb.ListTaxRegistrationsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxRegistration, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_registration.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistration.ListTaxRegistrations(ctx, req)
}
