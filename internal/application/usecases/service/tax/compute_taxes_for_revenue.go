package tax

import (
	"context"
	"errors"
	"time"

	taxcompute "github.com/erniealice/espyna-golang/internal/application/usecases/tax/compute_taxes_for_revenue"

	taxpb "github.com/erniealice/esqyma/pkg/schema/v1/service/tax"
)

// ComputeTaxesForRevenueUseCase is the proto-shaped wrapper over the
// entity-layer compute_taxes_for_revenue.ComputeTaxesForRevenueUseCase.
//
// It translates the proto Request/Response shape defined in
// proto/v1/service/tax/compute.proto into the Go-shaped Request/Response
// the entity-layer use case carries. The wrapper does NOT re-implement the
// algorithm — it delegates to the entity-layer use case to preserve the
// tax-integration plan §4 Phase C invariants (fail-closed STANDARD/REDUCED,
// asOf pinning, transactional DELETE+INSERT, multi-currency guard).
//
// Per Q-SDM-TAX (LOCKED 2026-05-20), this wrapper is the formal proto
// contract for the service-driven tax compute. RecognizeRevenueFromSubscription
// (usecases/revenue/revenue/recognize_revenue_from_subscription.go:123)
// satisfies its narrow ComputeTaxesForRevenueInvoker contract by calling
// ExecuteForRevenue on THIS wrapper — the composition root resolves the
// wrapper via the service registry (servicetax.From) and installs it via
// SetComputeTaxes. Failure semantics are preserved: ExecuteForRevenue
// returns the same error shape the entity-layer use case did, so the
// recognize flow's "tax_compute_failed: <err>" warning path is unchanged.
type ComputeTaxesForRevenueUseCase struct {
	entityCompute *taxcompute.ComputeTaxesForRevenueUseCase
}

// NewComputeTaxesForRevenueUseCase wires the wrapper.
func NewComputeTaxesForRevenueUseCase(entityCompute *taxcompute.ComputeTaxesForRevenueUseCase) *ComputeTaxesForRevenueUseCase {
	return &ComputeTaxesForRevenueUseCase{entityCompute: entityCompute}
}

// Execute runs the entity-layer compute algorithm with proto-shaped IO.
//
// The proto AsOfDate is a string formatted YYYY-MM-DD; empty string
// indicates "fall back to revenue.revenue_date" (same semantics as the
// entity-layer zero-value time.Time).
func (uc *ComputeTaxesForRevenueUseCase) Execute(
	ctx context.Context,
	req *taxpb.ComputeTaxesForRevenueRequest,
) (*taxpb.ComputeTaxesForRevenueResponse, error) {
	if uc == nil || uc.entityCompute == nil {
		return nil, errors.New("tax compute use case is not wired (no SQL provider registered)")
	}
	if req == nil {
		return nil, errors.New("ComputeTaxesForRevenueRequest is nil")
	}

	asOf := time.Time{}
	if req.AsOfDate != "" {
		parsed, err := time.Parse("2006-01-02", req.AsOfDate)
		if err != nil {
			return nil, errors.New("invalid AsOfDate: must be YYYY-MM-DD")
		}
		asOf = parsed
	}

	entityResp, err := uc.entityCompute.Execute(ctx, &taxcompute.ComputeTaxesRequest{
		RevenueID:   req.RevenueId,
		WorkspaceID: req.WorkspaceId,
		AsOf:        asOf,
		DryRun:      req.DryRun,
		IsRecompute: req.IsRecompute,
	})
	if err != nil {
		return nil, err
	}

	return &taxpb.ComputeTaxesForRevenueResponse{
		Lines: entityResp.Lines,
	}, nil
}

// ExecuteForRevenue is the narrow 3-arg shape that satisfies the
// ComputeTaxesForRevenueInvoker interface declared by
// usecases/revenue/revenue/recognize_revenue_from_subscription.go:123.
//
// It exists so the composition root can wire this service-layer wrapper
// (not the entity-layer use case) into RecognizeRevenueFromSubscription's
// post-persist hook. The body forwards to the entity-layer use case with
// IsRecompute=false (first-time compute path) — preserving the exact
// failure semantics the entity-layer use case had when it was wired
// directly, so the recognize flow's "tax_compute_failed: <err>" warning
// behavior is unchanged.
//
// nil-safe: returns an error rather than panicking when the wrapper or
// the captured entity-layer compute use case is nil.
func (uc *ComputeTaxesForRevenueUseCase) ExecuteForRevenue(ctx context.Context, revenueID, workspaceID string) error {
	if uc == nil || uc.entityCompute == nil {
		return errors.New("tax compute use case is not wired (no SQL provider registered)")
	}
	return uc.entityCompute.ExecuteForRevenue(ctx, revenueID, workspaceID)
}
