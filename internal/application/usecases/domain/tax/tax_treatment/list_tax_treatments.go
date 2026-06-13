package tax_treatment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// ListTaxTreatmentsRepositories groups repository dependencies.
type ListTaxTreatmentsRepositories struct {
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// ListTaxTreatmentsServices groups service dependencies.
type ListTaxTreatmentsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxTreatment,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_treatment.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxTreatment.ListTaxTreatments(ctx, req)
}
