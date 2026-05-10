package tax_registration_kind

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

// FindByPartyTypeQueries is the narrow interface from the adapter.
type FindByPartyTypeQueries interface {
	FindByPartyType(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error)
}

// FindByPartyTypeTaxRegistrationKindRepositories groups repository dependencies.
type FindByPartyTypeTaxRegistrationKindRepositories struct {
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// FindByPartyTypeTaxRegistrationKindServices groups service dependencies.
type FindByPartyTypeTaxRegistrationKindServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// FindByPartyTypeTaxRegistrationKindUseCase wraps the adapter's FindByPartyType method.
// Used by the Tax Registration form's Kind dropdown to show only kinds whose
// applicable_party_types includes the current party's type (CLIENT or WORKSPACE).
type FindByPartyTypeTaxRegistrationKindUseCase struct {
	repositories FindByPartyTypeTaxRegistrationKindRepositories
	services     FindByPartyTypeTaxRegistrationKindServices
}

// NewFindByPartyTypeTaxRegistrationKindUseCase creates the use case.
func NewFindByPartyTypeTaxRegistrationKindUseCase(
	repositories FindByPartyTypeTaxRegistrationKindRepositories,
	services FindByPartyTypeTaxRegistrationKindServices,
) *FindByPartyTypeTaxRegistrationKindUseCase {
	return &FindByPartyTypeTaxRegistrationKindUseCase{repositories: repositories, services: services}
}

// Execute returns all TaxRegistrationKind rows applicable for the given party type.
func (uc *FindByPartyTypeTaxRegistrationKindUseCase) Execute(ctx context.Context, partyType string) ([]*taxregistrationkindpb.TaxRegistrationKind, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRegistrationKind, ports.ActionList); err != nil {
		return nil, err
	}
	if partyType == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_registration_kind.validation.party_type_required", "Party type is required [DEFAULT]"))
	}

	q, ok := uc.repositories.TaxRegistrationKind.(FindByPartyTypeQueries)
	if !ok {
		// Adapter doesn't implement FindByPartyType — fall back to list-all.
		resp, err := uc.repositories.TaxRegistrationKind.ListTaxRegistrationKinds(ctx,
			&taxregistrationkindpb.ListTaxRegistrationKindsRequest{})
		if err != nil {
			return nil, fmt.Errorf("list tax_registration_kinds (fallback): %w", err)
		}
		if resp == nil {
			return nil, nil
		}
		return resp.GetData(), nil
	}

	kinds, err := q.FindByPartyType(ctx, partyType)
	if err != nil {
		return nil, fmt.Errorf("find tax_registration_kinds by party_type %q: %w", partyType, err)
	}
	return kinds, nil
}
