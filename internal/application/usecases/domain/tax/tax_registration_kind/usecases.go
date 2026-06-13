package tax_registration_kind

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

const entityTaxRegistrationKind = "tax_registration_kind"

// TaxRegistrationKindRepositories groups all repository dependencies for tax_registration_kind use cases.
type TaxRegistrationKindRepositories struct {
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// TaxRegistrationKindServices groups all business service dependencies.
type TaxRegistrationKindServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all tax_registration_kind use cases.
type UseCases struct {
	ReadTaxRegistrationKind            *ReadTaxRegistrationKindUseCase
	ListTaxRegistrationKinds           *ListTaxRegistrationKindsUseCase
	FindByPartyTypeTaxRegistrationKind *FindByPartyTypeTaxRegistrationKindUseCase
}

// NewUseCases creates a new collection of tax_registration_kind use cases.
func NewUseCases(repositories TaxRegistrationKindRepositories, services TaxRegistrationKindServices) *UseCases {
	return &UseCases{
		ReadTaxRegistrationKind: NewReadTaxRegistrationKindUseCase(
			ReadTaxRegistrationKindRepositories{TaxRegistrationKind: repositories.TaxRegistrationKind},
			ReadTaxRegistrationKindServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListTaxRegistrationKinds: NewListTaxRegistrationKindsUseCase(
			ListTaxRegistrationKindsRepositories{TaxRegistrationKind: repositories.TaxRegistrationKind},
			ListTaxRegistrationKindsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		FindByPartyTypeTaxRegistrationKind: NewFindByPartyTypeTaxRegistrationKindUseCase(
			FindByPartyTypeTaxRegistrationKindRepositories{TaxRegistrationKind: repositories.TaxRegistrationKind},
			FindByPartyTypeTaxRegistrationKindServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
