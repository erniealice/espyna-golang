package forex_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// ReadForexRateRepositories groups repository dependencies.
type ReadForexRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// ReadForexRateServices groups service dependencies.
type ReadForexRateServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadForexRateUseCase handles reading a forex_rate.
type ReadForexRateUseCase struct {
	repositories ReadForexRateRepositories
	services     ReadForexRateServices
}

// NewReadForexRateUseCase creates a new ReadForexRateUseCase.
func NewReadForexRateUseCase(repositories ReadForexRateRepositories, services ReadForexRateServices) *ReadForexRateUseCase {
	return &ReadForexRateUseCase{repositories: repositories, services: services}
}

// Execute performs the read forex_rate operation.
func (uc *ReadForexRateUseCase) Execute(ctx context.Context, req *forexratepb.ReadForexRateRequest) (*forexratepb.ReadForexRateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityForexRate, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"forex_rate.validation.id_required", "Forex Rate ID is required [DEFAULT]"))
	}
	return uc.repositories.ForexRate.ReadForexRate(ctx, req)
}
