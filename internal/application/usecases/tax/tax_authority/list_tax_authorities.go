package tax_authority

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
)

// ListTaxAuthoritiesRepositories groups repository dependencies.
type ListTaxAuthoritiesRepositories struct {
	TaxAuthority taxauthoritypb.TaxAuthorityDomainServiceServer
}

// ListTaxAuthoritiesServices groups service dependencies.
type ListTaxAuthoritiesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListTaxAuthoritiesUseCase handles listing tax authorities.
type ListTaxAuthoritiesUseCase struct {
	repositories ListTaxAuthoritiesRepositories
	services     ListTaxAuthoritiesServices
}

// NewListTaxAuthoritiesUseCase creates a new ListTaxAuthoritiesUseCase.
func NewListTaxAuthoritiesUseCase(repositories ListTaxAuthoritiesRepositories, services ListTaxAuthoritiesServices) *ListTaxAuthoritiesUseCase {
	return &ListTaxAuthoritiesUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax authorities operation.
func (uc *ListTaxAuthoritiesUseCase) Execute(ctx context.Context, req *taxauthoritypb.ListTaxAuthoritiesRequest) (*taxauthoritypb.ListTaxAuthoritiesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxAuthority, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_authority.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxAuthority.ListTaxAuthorities(ctx, req)
}
