package tax_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

// ReadTaxRateRepositories groups repository dependencies.
type ReadTaxRateRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// ReadTaxRateServices groups service dependencies.
type ReadTaxRateServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTaxRate, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"tax_rate.validation.id_required", "Tax Rate ID is required [DEFAULT]"))
	}
	return uc.repositories.TaxRate.ReadTaxRate(ctx, req)
}
