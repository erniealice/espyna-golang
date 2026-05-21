package tax_treatment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

const entityTaxTreatment = "tax_treatment"

// TaxTreatmentRepositories groups all repository dependencies for tax_treatment use cases.
type TaxTreatmentRepositories struct {
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// TaxTreatmentServices groups all business service dependencies.
type TaxTreatmentServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UseCases contains all tax_treatment use cases.
type UseCases struct {
	ReadTaxTreatment  *ReadTaxTreatmentUseCase
	ListTaxTreatments *ListTaxTreatmentsUseCase
}

// NewUseCases creates a new collection of tax_treatment use cases.
func NewUseCases(repositories TaxTreatmentRepositories, services TaxTreatmentServices) *UseCases {
	return &UseCases{
		ReadTaxTreatment: NewReadTaxTreatmentUseCase(
			ReadTaxTreatmentRepositories{TaxTreatment: repositories.TaxTreatment},
			ReadTaxTreatmentServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListTaxTreatments: NewListTaxTreatmentsUseCase(
			ListTaxTreatmentsRepositories{TaxTreatment: repositories.TaxTreatment},
			ListTaxTreatmentsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
