package forex_rate

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// FindMostRecentForexRateRepositories groups repository dependencies.
type FindMostRecentForexRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
	// Mutator provides FindMostRecent — same interface as RecordOperatorRate uses.
	Mutator ForexRateMutator
}

// FindMostRecentForexRateServices groups service dependencies.
type FindMostRecentForexRateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// FindMostRecentForexRateRequest is the input for finding the most recent forex rate.
type FindMostRecentForexRateRequest struct {
	WorkspaceID  string
	FromCurrency string
	ToCurrency   string
}

// FindMostRecentForexRateUseCase wraps the adapter's FindMostRecent method.
// Returns the most recent ACTIVE forex_rate row for a currency pair in a workspace.
// Used by the recognize-revenue drawer to pre-fill the FX rate input.
type FindMostRecentForexRateUseCase struct {
	repositories FindMostRecentForexRateRepositories
	services     FindMostRecentForexRateServices
}

// NewFindMostRecentForexRateUseCase creates the use case.
func NewFindMostRecentForexRateUseCase(
	repositories FindMostRecentForexRateRepositories,
	services FindMostRecentForexRateServices,
) *FindMostRecentForexRateUseCase {
	return &FindMostRecentForexRateUseCase{repositories: repositories, services: services}
}

// Execute returns the most recent ACTIVE forex_rate row for the given currency pair.
func (uc *FindMostRecentForexRateUseCase) Execute(ctx context.Context, req *FindMostRecentForexRateRequest) (*forexratepb.ForexRate, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityForexRate, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.WorkspaceID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.workspace_id_required", "Workspace ID is required [DEFAULT]"))
	}
	if req.FromCurrency == "" || req.ToCurrency == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.currency_pair_required", "Currency pair is required [DEFAULT]"))
	}

	if uc.repositories.Mutator == nil {
		return nil, fmt.Errorf("forex_rate repository does not support FindMostRecent")
	}
	return uc.repositories.Mutator.FindMostRecent(ctx, req.WorkspaceID, req.FromCurrency, req.ToCurrency)
}
