package tax_registration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

const entityTaxRegistration = "tax_registration"

// TaxRegistrationRepositories groups all repository dependencies for tax_registration use cases.
type TaxRegistrationRepositories struct {
	TaxRegistration     taxregistrationpb.TaxRegistrationDomainServiceServer
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// TaxRegistrationServices groups all business service dependencies.
type TaxRegistrationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all tax_registration use cases.
// NOTE: UpdateTaxRegistration and DeleteTaxRegistration use cases have been
// removed and replaced by SupersedeTaxRegistration and RevokeTaxRegistration
// to enforce the immutable-row + self-FK supersession contract from plan.md §2.
// Route names "update" and "delete" still map to these new use cases for CRUD
// permission compatibility (tax_registration:update|delete permissions still work).
type UseCases struct {
	CreateTaxRegistration    *CreateTaxRegistrationUseCase
	ReadTaxRegistration      *ReadTaxRegistrationUseCase
	SupersedeTaxRegistration *SupersedeTaxRegistrationUseCase
	RevokeTaxRegistration    *RevokeTaxRegistrationUseCase
	FindActiveTaxRegistration *FindActiveTaxRegistrationUseCase
	ListTaxRegistrations     *ListTaxRegistrationsUseCase
}

// NewUseCases creates a new collection of tax_registration use cases.
func NewUseCases(repositories TaxRegistrationRepositories, services TaxRegistrationServices) *UseCases {
	return &UseCases{
		CreateTaxRegistration: NewCreateTaxRegistrationUseCase(
			CreateTaxRegistrationRepositories{
				TaxRegistration:     repositories.TaxRegistration,
				TaxRegistrationKind: repositories.TaxRegistrationKind,
			},
			CreateTaxRegistrationServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadTaxRegistration: NewReadTaxRegistrationUseCase(
			ReadTaxRegistrationRepositories{TaxRegistration: repositories.TaxRegistration},
			ReadTaxRegistrationServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		SupersedeTaxRegistration: NewSupersedeTaxRegistrationUseCase(
			SupersedeTaxRegistrationRepositories{
				TaxRegistration:     repositories.TaxRegistration,
				TaxRegistrationKind: repositories.TaxRegistrationKind,
			},
			SupersedeTaxRegistrationServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		RevokeTaxRegistration: NewRevokeTaxRegistrationUseCase(
			RevokeTaxRegistrationRepositories{TaxRegistration: repositories.TaxRegistration},
			RevokeTaxRegistrationServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		FindActiveTaxRegistration: NewFindActiveTaxRegistrationUseCase(
			FindActiveTaxRegistrationRepositories{TaxRegistration: repositories.TaxRegistration},
			FindActiveTaxRegistrationServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListTaxRegistrations: NewListTaxRegistrationsUseCase(
			ListTaxRegistrationsRepositories{TaxRegistration: repositories.TaxRegistration},
			ListTaxRegistrationsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
