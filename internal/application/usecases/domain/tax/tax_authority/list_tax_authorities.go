package tax_authority

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
)

// ListTaxAuthoritiesRepositories groups repository dependencies.
type ListTaxAuthoritiesRepositories struct {
	TaxAuthority taxauthoritypb.TaxAuthorityDomainServiceServer
}

// ListTaxAuthoritiesServices groups service dependencies.
type ListTaxAuthoritiesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxAuthority,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_authority.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxAuthority.ListTaxAuthorities(ctx, req)
}
