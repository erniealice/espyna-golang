package tax_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxRate, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_rate.validation.id_required", "Tax Rate ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRate.ReadTaxRate(ctx, req)
}
