//go:build postgresql

package operation

import (
	"context"
	"strings"
	"testing"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// Gate-rollup read (plan 20260729 Phase 2) — the NO-DB halves of the contract:
// the emitted SQL hosts the EXACT shared groupNarrowPredicate exactly once per
// statement with no paging clause, and every "cannot apply" input state is an
// ERROR before any query runs. The row-set halves live in
// phase_approval_gate_query_integration_test.go.

// TestGateRollupSQL_HostsTheSharedPredicateVerbatimOnce pins the load-bearing
// extraction decision: both statements must embed the string
// groupNarrowPredicate(groupID, 3, 2) emits — string equality against the
// function itself (the job_phase_group_narrow_test.go technique), so the gate
// query can never drift from the four transitions and the matrix roll-up about
// which jobs belong to a group. Exactly ONCE per statement: the roll-up's
// historical `+ narrow + narrow` double-append is not to be reproduced.
func TestGateRollupSQL_HostsTheSharedPredicateVerbatimOnce(t *testing.T) {
	narrow, narrowArgs := groupNarrowPredicate("grp-A", 3, 2)
	if narrow == "" {
		t.Fatal("predicate must emit for a non-empty group")
	}
	if len(narrowArgs) != 1 || narrowArgs[0] != "grp-A" {
		t.Fatalf("predicate must contribute exactly the group id, got %v", narrowArgs)
	}

	for name, q := range map[string]string{
		"statusSQL":  gateRollupStatusSQL(narrow),
		"hasDataSQL": gateRollupHasDataSQL(narrow),
	} {
		if got := strings.Count(q, narrow); got != 1 {
			t.Errorf("%s must host the shared predicate EXACTLY once, got %d:\n%s", name, got, q)
		}
		// The five predicate terms, spelled out so a hollow Count match cannot
		// hide a truncated embed.
		for _, frag := range []string{
			"sgm_g.client_id = j.client_id",
			"sgm_g.subscription_id = j.origin_id", // the delivery pin the cells predicate lacks
			"sgm_g.subscription_group_id = $3",
			"sgm_g.workspace_id = $2",
			"sgm_g.active = true",
		} {
			if !strings.Contains(q, frag) {
				t.Errorf("%s missing predicate term %q:\n%s", name, frag, q)
			}
		}
		// H-1 absent by construction: the aggregate reads the whole narrowed
		// relation — no page boundary, no user-controlled ordering.
		upper := strings.ToUpper(q)
		for _, forbidden := range []string{"LIMIT", "OFFSET", "ORDER BY"} {
			if strings.Contains(upper, forbidden) {
				t.Errorf("%s must not contain %s:\n%s", name, forbidden, q)
			}
		}
		// Shared binds: the id-set and trusted-workspace placeholders.
		for _, frag := range []string{"= ANY($1)", "j.workspace_id = $2", "jp.active = true"} {
			if !strings.Contains(q, frag) {
				t.Errorf("%s missing %q:\n%s", name, frag, q)
			}
		}
	}

	// The status statement must aggregate the parity-pinned expression verbatim
	// (the integration parity test evaluates the same constant).
	if !strings.Contains(gateRollupStatusSQL(narrow), gateAnyWorkflowEnteredSQLExpr) {
		t.Error("statusSQL must embed gateAnyWorkflowEnteredSQLExpr verbatim")
	}
	// has_data joins must pin activity on both hops.
	for _, frag := range []string{"jt.active = true", "t.active = true"} {
		if !strings.Contains(gateRollupHasDataSQL(narrow), frag) {
			t.Errorf("hasDataSQL missing %q", frag)
		}
	}
}

// TestGetPhaseApprovalGateRollup_UnappliableInputsAreErrors pins the fail-closed
// input contract: nil request, empty group, empty template-phase ids, missing
// identity, and empty workspace must each refuse with an ERROR — never an empty
// success a document-integrity consumer would read as "no sheets". All five
// refuse BEFORE any query: the receiver's db is nil, so reaching a query would
// panic the test instead of passing it.
func TestGetPhaseApprovalGateRollup_UnappliableInputsAreErrors(t *testing.T) {
	a := &PostgresOutcomeMatrixQuery{}
	validReq := func() *matrixpb.GetPhaseApprovalGateRollupRequest {
		return &matrixpb.GetPhaseApprovalGateRollupRequest{
			SubscriptionGroupId: "grp-A",
			JobTemplatePhaseIds: []string{"tp-1"},
		}
	}

	for _, tc := range []struct {
		name string
		ctx  context.Context
		req  *matrixpb.GetPhaseApprovalGateRollupRequest
	}{
		{"nil request", wsCtx("ws-1"), nil},
		{"empty group id", wsCtx("ws-1"), &matrixpb.GetPhaseApprovalGateRollupRequest{JobTemplatePhaseIds: []string{"tp-1"}}},
		{"empty template phase ids", wsCtx("ws-1"), &matrixpb.GetPhaseApprovalGateRollupRequest{SubscriptionGroupId: "grp-A"}},
		{"no identity at all", context.Background(), validReq()},
		{"identity with empty workspace", wsCtx(""), validReq()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := a.GetPhaseApprovalGateRollup(tc.ctx, tc.req)
			if err == nil {
				t.Fatalf("must refuse (fail closed), got success: %+v", resp)
			}
			if resp != nil {
				t.Errorf("a refused read must not return a response, got %+v", resp)
			}
		})
	}
}

// TestAssembleGateRollups_EchoIsTheRequestGroupVerbatim pins the echo contract:
// every assembled rollup carries the REQUEST's group id, verbatim — the echo is
// the applied-narrow proof the consumer compares against its requested id.
func TestAssembleGateRollups_EchoIsTheRequestGroupVerbatim(t *testing.T) {
	aggs := []gateAgg{
		{templatePhaseID: "tp-1", targetCount: 27, anyWorkflowEntered: true, allPublished: false},
		{templatePhaseID: "tp-2", targetCount: 30, anyWorkflowEntered: false, allPublished: false},
	}
	got := assembleGateRollups("grp-Echo", aggs, map[string]bool{"tp-1": true})
	if len(got) != 2 {
		t.Fatalf("want one rollup per aggregate row, got %d", len(got))
	}
	for _, r := range got {
		if r.GetAppliedSubscriptionGroupId() != "grp-Echo" {
			t.Errorf("rollup %s echo = %q, want the request group verbatim", r.GetJobTemplatePhaseId(), r.GetAppliedSubscriptionGroupId())
		}
	}
	if !got[0].GetHasData() || got[0].GetTargetCount() != 27 || !got[0].GetAnyWorkflowEntered() {
		t.Errorf("tp-1 rollup lost fields: %+v", got[0])
	}
	if got[1].GetHasData() {
		t.Errorf("tp-2 has no data probe hit — HasData must be false: %+v", got[1])
	}

	// A phase with zero members is ABSENT (no aggregate row), never a zero-count
	// present row — the consumer's coverage check owns that absence.
	if empty := assembleGateRollups("grp-Echo", nil, nil); len(empty) != 0 {
		t.Errorf("no aggregate rows must assemble to no rollups, got %d", len(empty))
	}
}
