package forex_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// ReadForexRateRepositories groups repository dependencies.
type ReadForexRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// ReadForexRateServices groups service dependencies.
type ReadForexRateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityForexRate,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.id_required", "Forex Rate ID is required [DEFAULT]"))
	}
	return uc.repositories.ForexRate.ReadForexRate(ctx, req)
}
