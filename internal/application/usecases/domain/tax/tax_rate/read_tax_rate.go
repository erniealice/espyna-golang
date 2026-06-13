package tax_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

// ReadTaxRateRepositories groups repository dependencies.
type ReadTaxRateRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// ReadTaxRateServices groups service dependencies.
type ReadTaxRateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadTaxRateUseCase handles reading a tax_rate.
type ReadTaxRateUseCase struct {
	repositories ReadTaxRateRepositories
	services     ReadTaxRateServices
}

// NewReadTaxRateUseCase creates a new ReadTaxRateUseCase.
func NewReadTaxRateUseCase(repositories ReadTaxRateRepositories, services ReadTaxRateServices) *ReadTaxRateUseCase {
	return &ReadTaxRateUseCase{repositories: repositories, services: services}
}

// Execute performs the read tax_rate operation.
func (uc *ReadTaxRateUseCase) Execute(ctx context.Context, req *taxratepb.ReadTaxRateRequest) (*taxratepb.ReadTaxRateResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxRate,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_rate.validation.id_required", "Tax Rate ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRate.ReadTaxRate(ctx, req)
}
