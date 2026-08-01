package outcome_completion

import (
	"context"
	"errors"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	ocpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/outcome_completion"
)

// permissionSubject is the Gate 1 permission subject for this aggregate.
// `outcome_completion` is a service-read noun, NOT an entityid (there is no
// outcome_completion table or entity — copya.md §C-2 precedent: the
// verb-folding family). Combined with entityid.ActionRead it yields the Q7
// locked code `outcome_completion:read`, seeded + tagged {1,2,7}.
const permissionSubject = "outcome_completion"

// Principal kinds this read defines behavior for. Mirrors the esqyma
// domain.entity.v1.PrincipalType integer values as re-declared by
// pyeza render/types.go and contrib/postgres principalscope (kind 7 = staff);
// declared locally so this package imports neither (application layer stays
// proto/adapter-free for identity vocabulary).
const (
	principalKindOperatorOwner int32 = 1
	principalKindOperatorStaff int32 = 2
	principalKindStaff         int32 = 7
)

// CategoryPeriodCount is ONE flat repository row: the completion + approval
// counts for a single (category, phase_order) slice of the current-window
// aggregate. The adapter returns rows ordered by (category sort_order asc,
// category id asc, phase_order asc); the use case folds consecutive rows of
// one category into a proto OutcomeCompletionCategoryRow.
//
// HasSlice=false marks a category row with ZERO reachable cells (the adapter
// emits every active category so tab rendering shares this single read —
// copya.md §C-1); such a row carries no period slice and all-zero counts.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED
// 2026-05-20):** the postgres adapter MUST return exactly this named type via
// a `type CategoryPeriodCount = ...` alias — see
// contrib/postgres/internal/adapter/operation/outcome_completion_dashboard.go
// for the compile-time guard.
type CategoryPeriodCount struct {
	CategoryID   string // empty for the uncategorized residue row
	CategoryName string
	HasSlice     bool  // false = zero-cell category (no phase_order)
	PhaseOrder   int32 // job_phase.phase_order (A2 period dimension)

	ExpectedCells int64
	RecordedCells int64

	// Approval-ladder phase-row counts (disjoint; series 2 + the F5
	// ride-along buckets). See the proto OutcomeCompletionApprovalCounts
	// contract.
	NotStarted int64
	InProgress int64
	ForReview  int64
	Verified   int64
	Published  int64
	Returned   int64
}

// OutcomeCompletionSummaryRepository is the slice of the postgres job adapter
// this use case consumes: the live, current-window, cell-grain aggregate.
//
// The adapter self-scopes rows from ctx identity (Q-EIB-BRIDGE invariant —
// identity is never on the wire): principal kind 7 applies the A1
// most-specific-wins responsibility predicate on top of the mirrored
// reachable-job union; kinds 1/2 read at workspace grain; any other kind is
// fail-closed to zero rows (belt-and-suspenders under this use case's own
// kind check).
type OutcomeCompletionSummaryRepository interface {
	CompletionSummary(ctx context.Context, workspaceID string) ([]CategoryPeriodCount, error)
}

// GetOutcomeCompletionSummaryRepositories groups the per-repository
// dependencies (one today; struct kept for symmetry with the sibling
// dashboard candidates).
type GetOutcomeCompletionSummaryRepositories struct {
	OutcomeCompletion OutcomeCompletionSummaryRepository
}

// GetOutcomeCompletionSummaryUseCase serves the persona-aware home dashboard's
// completion + approval widgets (plan.md §4.3, Q3 lock as amended 2026-08-01:
// A1 most-specific-wins responsibility, A2 per-phase_order period dimension).
//
// **Gate 1 (Action) check.** The response carries real, principal-scoped
// operational counts, so Execute runs the gatekeeper on the dedicated
// `outcome_completion:read` capability (Q7 lock) BEFORE touching the
// repository; a principal lacking the code is denied fail-closed.
//
// **Identity check.** After the gate, Execute requires a ctx RequestIdentity
// and defines behavior ONLY for kinds 1/2 (workspace grain) and 7
// (staff-scoped inside the adapter). An absent identity or any other kind —
// including the unresolved kind-0 sentinel — is denied fail-closed (plan.md
// Phase 2: "kind 0 → deny").
type GetOutcomeCompletionSummaryUseCase struct {
	repositories     GetOutcomeCompletionSummaryRepositories
	actionGatekeeper *actiongate.ActionGatekeeper
}

// NewGetOutcomeCompletionSummaryUseCase wires the use case from grouped
// dependencies.
func NewGetOutcomeCompletionSummaryUseCase(
	repositories GetOutcomeCompletionSummaryRepositories,
	actionGate *actiongate.ActionGatekeeper,
) *GetOutcomeCompletionSummaryUseCase {
	return &GetOutcomeCompletionSummaryUseCase{
		repositories:     repositories,
		actionGatekeeper: actionGate,
	}
}

// Execute runs the aggregate read and assembles the proto response.
//
// Repository errors PROPAGATE (the T-9 lesson: the view renders a designed
// error/empty state — no silent partial widgets). A nil repository (mock /
// non-postgres builds, or a failed initializer type assertion) degrades to a
// zero-valued successful response.
//
// req.now_millis is accepted per the wire contract but unused: the A2
// amendment made denominator windowing STRUCTURAL (per phase_order slice,
// chosen by the consumer) — phase dates are NULL and cannot drive a window.
func (uc *GetOutcomeCompletionSummaryUseCase) Execute(
	ctx context.Context,
	req *ocpb.GetOutcomeCompletionSummaryRequest,
) (*ocpb.GetOutcomeCompletionSummaryResponse, error) {
	// Gate 1 (Action): the dedicated Q7 capability. Fail-closed on
	// missing/insufficient permission; nil gatekeeper denies.
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: permissionSubject,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Identity from ctx, never the wire (Q-EIB-BRIDGE). Fail-closed: no
	// identity, or a kind this read does not define (incl. kind 0), denies.
	id, err := identity.Require(ctx)
	if err != nil {
		return nil, err
	}
	switch id.PrincipalType {
	case principalKindOperatorOwner, principalKindOperatorStaff, principalKindStaff:
		// defined kinds — proceed
	default:
		return nil, errors.New("authorization denied: principal kind not permitted for this read")
	}

	// Workspace scope: prefer the SESSION workspace (unspoofable); the wire
	// workspace_id is only a fallback for identity-bearing contexts that did
	// not resolve a workspace. An empty scope matches no row (fail-closed).
	workspaceID := id.WorkspaceID
	if workspaceID == "" && req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &ocpb.GetOutcomeCompletionSummaryResponse{
		Success: true,
		AllRollup: &ocpb.OutcomeCompletionCategoryRow{
			ApprovalCounts: &ocpb.OutcomeCompletionApprovalCounts{},
		},
	}

	if uc.repositories.OutcomeCompletion == nil {
		return resp, nil
	}

	rows, err := uc.repositories.OutcomeCompletion.CompletionSummary(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Fold flat (category, phase_order) rows — already ordered by the adapter
	// (category sort_order asc, id asc, phase_order asc) — into per-category
	// proto rows plus the All rollup. Row totals = sum of slices by
	// construction (the proto contract).
	allByPhase := map[int32]*ocpb.OutcomeCompletionPeriodSlice{}
	var cur *ocpb.OutcomeCompletionCategoryRow
	curSet := false
	for _, r := range rows {
		if !curSet || cur.GetCategoryId() != r.CategoryID {
			cur = &ocpb.OutcomeCompletionCategoryRow{
				CategoryId:     r.CategoryID,
				CategoryName:   r.CategoryName,
				ApprovalCounts: &ocpb.OutcomeCompletionApprovalCounts{},
			}
			resp.CategoryRows = append(resp.CategoryRows, cur)
			curSet = true
		}
		if !r.HasSlice {
			continue
		}
		counts := &ocpb.OutcomeCompletionApprovalCounts{
			NotStarted: r.NotStarted,
			InProgress: r.InProgress,
			ForReview:  r.ForReview,
			Verified:   r.Verified,
			Published:  r.Published,
			Returned:   r.Returned,
		}
		cur.PeriodSlices = append(cur.PeriodSlices, &ocpb.OutcomeCompletionPeriodSlice{
			PhaseOrder:     r.PhaseOrder,
			ExpectedCells:  r.ExpectedCells,
			RecordedCells:  r.RecordedCells,
			ApprovalCounts: counts,
		})
		accumulateRow(cur, r)
		accumulateRow(resp.AllRollup, r)

		as, ok := allByPhase[r.PhaseOrder]
		if !ok {
			as = &ocpb.OutcomeCompletionPeriodSlice{
				PhaseOrder:     r.PhaseOrder,
				ApprovalCounts: &ocpb.OutcomeCompletionApprovalCounts{},
			}
			allByPhase[r.PhaseOrder] = as
		}
		as.ExpectedCells += r.ExpectedCells
		as.RecordedCells += r.RecordedCells
		addCounts(as.ApprovalCounts, counts)
	}

	// All-rollup slices, ordered by phase_order asc (the same window contract
	// as the category rows). category_id/name stay empty — the All tab label
	// comes from lyngua, never the wire.
	phaseOrders := make([]int, 0, len(allByPhase))
	for po := range allByPhase {
		phaseOrders = append(phaseOrders, int(po))
	}
	sort.Ints(phaseOrders)
	for _, po := range phaseOrders {
		resp.AllRollup.PeriodSlices = append(resp.AllRollup.PeriodSlices, allByPhase[int32(po)])
	}

	return resp, nil
}

// accumulateRow adds one slice row's counts into a category (or rollup) row's
// totals.
func accumulateRow(row *ocpb.OutcomeCompletionCategoryRow, r CategoryPeriodCount) {
	row.ExpectedCells += r.ExpectedCells
	row.RecordedCells += r.RecordedCells
	if row.ApprovalCounts == nil {
		row.ApprovalCounts = &ocpb.OutcomeCompletionApprovalCounts{}
	}
	row.ApprovalCounts.NotStarted += r.NotStarted
	row.ApprovalCounts.InProgress += r.InProgress
	row.ApprovalCounts.ForReview += r.ForReview
	row.ApprovalCounts.Verified += r.Verified
	row.ApprovalCounts.Published += r.Published
	row.ApprovalCounts.Returned += r.Returned
}

// addCounts adds src's approval counts into dst.
func addCounts(dst, src *ocpb.OutcomeCompletionApprovalCounts) {
	dst.NotStarted += src.NotStarted
	dst.InProgress += src.InProgress
	dst.ForReview += src.ForReview
	dst.Verified += src.Verified
	dst.Published += src.Published
	dst.Returned += src.Returned
}
