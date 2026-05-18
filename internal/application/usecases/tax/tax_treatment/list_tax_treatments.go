package tax_treatment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// ListTaxTreatmentsRepositories groups repository dependencies.
type ListTaxTreatmentsRepositories struct {
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// ListTaxTreatmentsServices groups service dependencies.
type ListTaxTreatmentsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListTaxTreatmentsUseCase handles listing tax treatments.
type ListTaxTreatmentsUseCase struct {
	repositories ListTaxTreatmentsRepositories
	services     ListTaxTreatmentsServices
}

// NewListTaxTreatmentsUseCase creates a new ListTaxTreatmentsUseCase.
func NewListTaxTreatmentsUseCase(repositories ListTaxTreatmentsRepositories, services ListTaxTreatmentsServices) *ListTaxTreatmentsUseCase {
	return &ListTaxTreatmentsUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax treatments operation.
func (uc *ListTaxTreatmentsUseCase) Execute(ctx context.Context, req *taxtreatmentpb.ListTaxTreatmentsRequest) (*taxtreatmentpb.ListTaxTreatmentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxTreatment, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_treatment.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxTreatment.ListTaxTreatments(ctx, req)
}
