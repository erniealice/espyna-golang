package consumer

// adapter_treasury_advances.go — public surface for the
// 20260517-advance-cash-events Plan B Phase 2, 3, 7 advance workflows
// (Amortize / Settle / Refund / Cancel / Recognize-Milestone / Dashboard).
//
// The internal use cases live under espyna's internal/ tree and are not
// importable by consumer apps. This adapter exposes a thin proto-typed
// wrapper that delegates each call straight to the use case the container
// already constructed — no struct conversion needed because the use cases
// now accept the same proto Request/Response types this adapter publishes.

import (
	"context"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	advancesdashboardpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/advances_dashboard"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// TreasuryAdvancesAdapter exposes Plan B Phase 2 + Phase 7 advance workflow
// use cases (amortize / settle / refund / cancel / recognize-milestone /
// dashboard) to consumer apps via a thin proto-typed wrapper.
type TreasuryAdvancesAdapter struct {
	useCases *UseCases
}

// NewTreasuryAdvancesAdapter wires the adapter from a use case aggregate.
// Returns nil when useCases is nil or the Treasury sub-aggregate is missing.
func NewTreasuryAdvancesAdapter(useCases *UseCases) *TreasuryAdvancesAdapter {
	if useCases == nil || useCases.Treasury == nil {
		return nil
	}
	return &TreasuryAdvancesAdapter{useCases: useCases}
}

// NewTreasuryAdvancesAdapterFromContainer pulls the use case aggregate off a
// container and wires the adapter. Returns nil when the container or use
// cases are not initialized.
func NewTreasuryAdvancesAdapterFromContainer(container *Container) *TreasuryAdvancesAdapter {
	if container == nil {
		return nil
	}
	return NewTreasuryAdvancesAdapter(container.GetUseCases())
}

// IsEnabled reports whether the adapter has a working Treasury aggregate.
func (a *TreasuryAdvancesAdapter) IsEnabled() bool {
	return a != nil && a.useCases != nil && a.useCases.Treasury != nil
}

// ErrTreasuryAdvanceUseCaseUnwired is the sentinel returned when the consumer
// attempts an advance workflow call while the underlying espyna use case is
// nil (e.g. a build/config that didn't initialize the Treasury repositories).
var ErrTreasuryAdvanceUseCaseUnwired = treasuryAdvanceUnwiredErr("treasury advance use case is not wired")

type treasuryAdvanceUnwiredErr string

func (e treasuryAdvanceUnwiredErr) Error() string { return string(e) }

// ---- Selling side (TreasuryCollection) -----------------------------------

// AmortizeAdvanceCollection routes the call to Plan B's selling-side
// AmortizeAdvanceCollection use case. The use case already consumes the proto
// Request directly, so no struct translation is needed.
func (a *TreasuryAdvancesAdapter) AmortizeAdvanceCollection(
	ctx context.Context,
	req *collectionpb.AmortizeAdvanceCollectionRequest,
) (*collectionpb.AmortizeAdvanceCollectionResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.AmortizeAdvanceCollection == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.AmortizeAdvanceCollection.Execute(ctx, req)
}

// SettleUnscheduledAdvanceCollection routes the call to the selling-side
// UNSCHEDULED settle use case.
func (a *TreasuryAdvancesAdapter) SettleUnscheduledAdvanceCollection(
	ctx context.Context,
	req *collectionpb.SettleUnscheduledAdvanceCollectionRequest,
) (*collectionpb.SettleUnscheduledAdvanceCollectionResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.SettleUnscheduledAdvanceCollection == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.SettleUnscheduledAdvanceCollection.Execute(ctx, req)
}

// RefundUnscheduledAdvanceCollection routes the call to the selling-side
// UNSCHEDULED refund use case.
func (a *TreasuryAdvancesAdapter) RefundUnscheduledAdvanceCollection(
	ctx context.Context,
	req *collectionpb.RefundUnscheduledAdvanceCollectionRequest,
) (*collectionpb.RefundUnscheduledAdvanceCollectionResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.RefundUnscheduledAdvanceCollection == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.RefundUnscheduledAdvanceCollection.Execute(ctx, req)
}

// CancelAdvanceCollection routes the call to the selling-side cancel use case.
func (a *TreasuryAdvancesAdapter) CancelAdvanceCollection(
	ctx context.Context,
	req *collectionpb.CancelAdvanceCollectionRequest,
) (*collectionpb.CancelAdvanceCollectionResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.CancelAdvanceCollection == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.CancelAdvanceCollection.Execute(ctx, req)
}

// RecognizeMilestoneAdvanceCollection routes the call to the selling-side
// MILESTONE recognize use case (Plan B Phase 7).
func (a *TreasuryAdvancesAdapter) RecognizeMilestoneAdvanceCollection(
	ctx context.Context,
	req *collectionpb.RecognizeMilestoneAdvanceCollectionRequest,
) (*collectionpb.RecognizeMilestoneAdvanceCollectionResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.RecognizeMilestoneAdvanceCollection == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.RecognizeMilestoneAdvanceCollection.Execute(ctx, req)
}

// ---- Buying side (TreasuryDisbursement) ----------------------------------

// AmortizeAdvanceDisbursement routes the call to Plan B's buying-side
// AmortizeAdvanceDisbursement use case.
func (a *TreasuryAdvancesAdapter) AmortizeAdvanceDisbursement(
	ctx context.Context,
	req *disbursementpb.AmortizeAdvanceDisbursementRequest,
) (*disbursementpb.AmortizeAdvanceDisbursementResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.AmortizeAdvanceDisbursement == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.AmortizeAdvanceDisbursement.Execute(ctx, req)
}

// SettleUnscheduledAdvanceDisbursement routes the call to the buying-side
// UNSCHEDULED settle use case.
func (a *TreasuryAdvancesAdapter) SettleUnscheduledAdvanceDisbursement(
	ctx context.Context,
	req *disbursementpb.SettleUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.SettleUnscheduledAdvanceDisbursementResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.SettleUnscheduledAdvanceDisbursement == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.SettleUnscheduledAdvanceDisbursement.Execute(ctx, req)
}

// RefundUnscheduledAdvanceDisbursement routes the call to the buying-side
// UNSCHEDULED refund use case.
func (a *TreasuryAdvancesAdapter) RefundUnscheduledAdvanceDisbursement(
	ctx context.Context,
	req *disbursementpb.RefundUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.RefundUnscheduledAdvanceDisbursementResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.RefundUnscheduledAdvanceDisbursement == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.RefundUnscheduledAdvanceDisbursement.Execute(ctx, req)
}

// CancelAdvanceDisbursement routes the call to the buying-side cancel use case.
func (a *TreasuryAdvancesAdapter) CancelAdvanceDisbursement(
	ctx context.Context,
	req *disbursementpb.CancelAdvanceDisbursementRequest,
) (*disbursementpb.CancelAdvanceDisbursementResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.CancelAdvanceDisbursement == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.CancelAdvanceDisbursement.Execute(ctx, req)
}

// RecognizeMilestoneAdvanceDisbursement routes the call to the buying-side
// MILESTONE recognize use case (Plan B Phase 7).
func (a *TreasuryAdvancesAdapter) RecognizeMilestoneAdvanceDisbursement(
	ctx context.Context,
	req *disbursementpb.RecognizeMilestoneAdvanceDisbursementRequest,
) (*disbursementpb.RecognizeMilestoneAdvanceDisbursementResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.RecognizeMilestoneAdvanceDisbursement == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.RecognizeMilestoneAdvanceDisbursement.Execute(ctx, req)
}

// ---- Workspace dashboard --------------------------------------------------

// GetAdvancesDashboard returns the workspace-level Advances Dashboard
// projection (Plan B Phase 3). When the use case is unwired, returns the
// sentinel error so callers can decide whether to render empty state.
func (a *TreasuryAdvancesAdapter) GetAdvancesDashboard(
	ctx context.Context,
	req *advancesdashboardpb.GetAdvancesDashboardRequest,
) (*advancesdashboardpb.GetAdvancesDashboardResponse, error) {
	if !a.IsEnabled() || a.useCases.Treasury.GetAdvancesDashboard == nil {
		return nil, ErrTreasuryAdvanceUseCaseUnwired
	}
	return a.useCases.Treasury.GetAdvancesDashboard.Execute(ctx, req)
}

// ---- Re-exports for view-layer translation --------------------------------

// AdvanceAmortizeOutcome is re-exported so consumer apps can compare the
// outcome enum on Amortize / RecognizeMilestone responses without having to
// import esqyma proto packages directly.
type AdvanceAmortizeOutcome = advancekindpb.AdvanceAmortizeOutcome

// AdvanceStatus is re-exported for the same reason as AdvanceAmortizeOutcome.
type AdvanceStatus = advancekindpb.AdvanceStatus

// AdvanceKind is re-exported so view code can branch on the kind enum.
type AdvanceKind = advancekindpb.AdvanceKind

// Outcome constants — re-exported for switch ergonomics.
const (
	AdvanceAmortizeOutcomeUnspecified = advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_UNSPECIFIED
	AdvanceAmortizeOutcomeCreated     = advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_CREATED
	AdvanceAmortizeOutcomeSkipped     = advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_SKIPPED
	AdvanceAmortizeOutcomeErrored     = advancekindpb.AdvanceAmortizeOutcome_ADVANCE_AMORTIZE_OUTCOME_ERRORED
)
