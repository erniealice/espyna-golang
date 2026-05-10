package tax_registration_kind

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

// ListTaxRegistrationKindsRepositories groups repository dependencies.
type ListTaxRegistrationKindsRepositories struct {
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// ListTaxRegistrationKindsServices groups service dependencies.
type ListTaxRegistrationKindsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListTaxRegistrationKindsUseCase handles listing tax registration kinds.
type ListTaxRegistrationKindsUseCase struct {
	repositories ListTaxRegistrationKindsRepositories
	services     ListTaxRegistrationKindsServices
}

// NewListTaxRegistrationKindsUseCase creates a new ListTaxRegistrationKindsUseCase.
func NewListTaxRegistrationKindsUseCase(repositories ListTaxRegistrationKindsRepositories, services ListTaxRegistrationKindsServices) *ListTaxRegistrationKindsUseCase {
	return &ListTaxRegistrationKindsUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax_registration_kinds operation.
func (uc *ListTaxRegistrationKindsUseCase) Execute(ctx context.Context, req *taxregistrationkindpb.ListTaxRegistrationKindsRequest) (*taxregistrationkindpb.ListTaxRegistrationKindsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistrationKind, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration_kind.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxRegistrationKind.ListTaxRegistrationKinds(ctx, req)
}
