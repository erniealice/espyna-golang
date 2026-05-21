package tax_treatment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// ReadTaxTreatmentRepositories groups repository dependencies.
type ReadTaxTreatmentRepositories struct {
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// ReadTaxTreatmentServices groups service dependencies.
type ReadTaxTreatmentServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadTaxTreatmentUseCase handles reading a tax_treatment.
type ReadTaxTreatmentUseCase struct {
	repositories ReadTaxTreatmentRepositories
	services     ReadTaxTreatmentServices
}

// NewReadTaxTreatmentUseCase creates a new ReadTaxTreatmentUseCase.
func NewReadTaxTreatmentUseCase(repositories ReadTaxTreatmentRepositories, services ReadTaxTreatmentServices) *ReadTaxTreatmentUseCase {
	return &ReadTaxTreatmentUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_treatment operation.
func (uc *ReadTaxTreatmentUseCase) Execute(ctx context.Context, req *taxtreatmentpb.ReadTaxTreatmentRequest) (*taxtreatmentpb.ReadTaxTreatmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxTreatment, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_treatment.validation.id_required", "Tax Treatment ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxTreatment.ReadTaxTreatment(ctx, req)
}
