package forex_rate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// ForexRateMutator is a port for extended forex_rate operations not in the standard gRPC server.
// The PostgreSQL adapter implements this interface via ForexRateQueries.
type ForexRateMutator interface {
	// FindMostRecent returns the most recent ACTIVE row for a currency pair in the workspace.
	FindMostRecent(ctx context.Context, workspaceID, fromCurrency, toCurrency string) (*forexratepb.ForexRate, error)
	// Insert appends a new forex_rate row.
	Insert(ctx context.Context, rate *forexratepb.ForexRate) error
	// SupersedePrior marks a prior ACTIVE row as SUPERSEDED with the given effectiveTo timestamp.
	SupersedePrior(ctx context.Context, priorID string, effectiveTo time.Time) error
}

// RecordOperatorRateRepositories groups repository dependencies.
type RecordOperatorRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
	Mutator   ForexRateMutator
}

// RecordOperatorRateServices groups service dependencies.
type RecordOperatorRateServices struct {
	Authorizer  ports.Authorizer
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// RecordOperatorRateRequest is the input for recording a new operator-sourced forex rate.
type RecordOperatorRateRequest struct {
	WorkspaceID     string
	FromCurrency    string
	ToCurrency      string
	RateMicroUnits  int64
	EffectiveFrom   time.Time
	CreatedByUserID string
	Notes           string
}

// RecordOperatorRateUseCase handles recording a new operator-sourced forex rate.
// It appends a new ACTIVE row and supersedes the prior ACTIVE row for the same pair.
type RecordOperatorRateUseCase struct {
	repositories RecordOperatorRateRepositories
	services     RecordOperatorRateServices
}

// NewRecordOperatorRateUseCase creates a new RecordOperatorRateUseCase.
func NewRecordOperatorRateUseCase(repositories RecordOperatorRateRepositories, services RecordOperatorRateServices) *RecordOperatorRateUseCase {
	return &RecordOperatorRateUseCase{repositories: repositories, services: services}
}

// Execute records a new operator-sourced forex rate, superseding the prior ACTIVE row if one exists.
// It is idempotent in the sense that it will not create a duplicate if the rate value is identical
// to the most recent ACTIVE row for the same currency pair.
//
// System-authorized: no operator permission check. RecordOperatorRate is an internal
// side-effect of recognize-revenue and invoice-run — the plan explicitly does not expose
// a "create forex_rate" operator action. Permission seeds only carry forex_rate:read|list.
// See docs/plan/20260509-tax-integration/codex-review-phase2.md C4 for the decision.
func (uc *RecordOperatorRateUseCase) Execute(ctx context.Context, req *RecordOperatorRateRequest) (*forexratepb.ForexRate, error) {
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
	if req.RateMicroUnits <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.rate_required", "Rate must be positive [DEFAULT]"))
	}

	now := time.Now()
	effectiveFrom := req.EffectiveFrom
	if effectiveFrom.IsZero() {
		effectiveFrom = now
	}

	// Check for existing ACTIVE row — skip if rate is identical (idempotent).
	prior, err := uc.repositories.Mutator.FindMostRecent(ctx, req.WorkspaceID, req.FromCurrency, req.ToCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to check prior forex_rate: %w", err)
	}
	if prior != nil && prior.GetRateMicroUnits() == req.RateMicroUnits {
		// Identical rate already active — no-op.
		return prior, nil
	}

	// Supersede the prior ACTIVE row before inserting the new one.
	if prior != nil {
		if err := uc.repositories.Mutator.SupersedePrior(ctx, prior.GetId(), effectiveFrom); err != nil {
			return nil, fmt.Errorf("failed to supersede prior forex_rate: %w", err)
		}
	}

	// Build the new ACTIVE row.
	newRate := &forexratepb.ForexRate{
		Id:                 uc.services.IDGenerator.GenerateID(),
		WorkspaceId:        req.WorkspaceID,
		FromCurrency:       req.FromCurrency,
		ToCurrency:         req.ToCurrency,
		RateMicroUnits:     req.RateMicroUnits,
		Source:             forexratepb.ForexRateSource_FOREX_RATE_SOURCE_OPERATOR,
		EffectiveFrom:      effectiveFrom.Format(time.RFC3339),
		Status:             forexratepb.ForexRateStatus_FOREX_RATE_STATUS_ACTIVE,
		Active:             true,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
	}
	if req.CreatedByUserID != "" {
		newRate.CreatedByUserId = &req.CreatedByUserID
	}
	if req.Notes != "" {
		newRate.Notes = &req.Notes
	}
	if prior != nil {
		priorID := prior.GetId()
		newRate.SupersedesId = &priorID
	}

	if err := uc.repositories.Mutator.Insert(ctx, newRate); err != nil {
		return nil, fmt.Errorf("failed to insert forex_rate: %w", err)
	}

	return newRate, nil
}
