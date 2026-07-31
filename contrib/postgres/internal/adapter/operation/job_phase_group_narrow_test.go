//go:build postgresql

package operation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"

	"database/sql"
)

// Delivery-group narrow on ListJobPhases (plan 20260729 Phase 1 / Q1a).
//
// These are the NO-DB halves of the contract: the unnarrowed path must be
// untouched, and every "narrow could not be applied" state must be an ERROR, not
// a silently unnarrowed success. The row-set halves (unnarrowed set unchanged /
// only that group's phases / narrowed-to-an-empty-group is EMPTY) need real
// membership rows and live in job_phase_group_narrow_integration_test.go.

// ---- fakes -----------------------------------------------------------------

// probeRecorder is a DBExecutor that records every query it is handed and always
// fails. Failing is the point: a recorded call proves the probe FIRED, and the
// propagated error proves the adapter does not swallow a failed narrow into an
// unnarrowed or empty success.
type probeRecorder struct {
	queries []string
	args    [][]any
}

var errProbeRecorder = errors.New("probeRecorder: no real database")

func (p *probeRecorder) ExecContext(_ context.Context, q string, args ...any) (sql.Result, error) {
	p.queries = append(p.queries, q)
	p.args = append(p.args, args)
	return nil, errProbeRecorder
}
func (p *probeRecorder) QueryContext(_ context.Context, q string, args ...any) (*sql.Rows, error) {
	p.queries = append(p.queries, q)
	p.args = append(p.args, args)
	return nil, errProbeRecorder
}
func (p *probeRecorder) QueryRowContext(_ context.Context, q string, args ...any) *sql.Row {
	p.queries = append(p.queries, q)
	p.args = append(p.args, args)
	return nil
}

// narrowFakeDBOps returns a fixed row set from List and exposes a recording
// executor, so a test can assert BOTH the ListParams handed down and whether any
// probe was issued.
type narrowFakeDBOps struct {
	rows           []map[string]any
	lastListParams *interfaces.ListParams
	listCalls      int
	exec           *probeRecorder
}

func (f *narrowFakeDBOps) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *narrowFakeDBOps) Read(context.Context, string, string) (map[string]any, error) {
	return nil, nil
}
func (f *narrowFakeDBOps) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *narrowFakeDBOps) Delete(context.Context, string, string) error     { return nil }
func (f *narrowFakeDBOps) HardDelete(context.Context, string, string) error { return nil }
func (f *narrowFakeDBOps) List(_ context.Context, _ string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	f.listCalls++
	f.lastListParams = params
	return &interfaces.ListResult{Data: f.rows}, nil
}
func (f *narrowFakeDBOps) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (f *narrowFakeDBOps) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}
func (f *narrowFakeDBOps) GetExecutor(context.Context) sqlexec.DBExecutor { return f.exec }

func newNarrowFake(rows ...map[string]any) *narrowFakeDBOps {
	return &narrowFakeDBOps{rows: rows, exec: &probeRecorder{}}
}

// ---- the unnarrowed path is untouched --------------------------------------

// TestListJobPhases_AbsentNarrow_TakesTheIdenticalPreChangePath is requirement 1:
// with no subscription_group_id the adapter must issue the SAME single dbOps.List
// with the SAME ListParams and NO additional query. The recorder proves the
// "no additional query" half — the narrow probe is the only thing that could
// appear there.
func TestListJobPhases_AbsentNarrow_TakesTheIdenticalPreChangePath(t *testing.T) {
	filters := &commonpb.FilterRequest{}
	sort := &commonpb.SortRequest{}
	pag := &commonpb.PaginationRequest{Limit: 25}

	for _, tc := range []struct {
		name string
		req  *pb.ListJobPhasesRequest
	}{
		{"field absent", &pb.ListJobPhasesRequest{Filters: filters, Sort: sort, Pagination: pag}},
		{"field present but empty", &pb.ListJobPhasesRequest{Filters: filters, Sort: sort, Pagination: pag, SubscriptionGroupId: strptr("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newNarrowFake(
				map[string]any{"id": "p1", "jobId": "j1"},
				map[string]any{"id": "p2", "jobId": "j2"},
			)
			r := NewPostgresJobPhaseRepository(fake, "job_phase")

			resp, err := r.ListJobPhases(context.Background(), tc.req)
			if err != nil {
				t.Fatalf("unnarrowed list must not error: %v", err)
			}
			if fake.listCalls != 1 {
				t.Fatalf("expected exactly one dbOps.List, got %d", fake.listCalls)
			}
			if fake.lastListParams == nil {
				t.Fatal("ListParams dropped")
			}
			if fake.lastListParams.Filters != filters || fake.lastListParams.Sort != sort || fake.lastListParams.Pagination != pag {
				t.Errorf("ListParams must forward the caller's exact pointers, got %+v", fake.lastListParams)
			}
			if len(fake.exec.queries) != 0 {
				t.Errorf("an absent/empty narrow must issue NO extra query, got %d: %v", len(fake.exec.queries), fake.exec.queries)
			}
			if got := len(resp.GetData()); got != 2 {
				t.Errorf("unnarrowed row set changed: got %d rows want 2", got)
			}
		})
	}
}

// TestListJobPhases_AbsentNarrow_NilRequestStillSafe pins that the new getter is
// nil-safe — req == nil must behave exactly as before (nil ListParams, no probe).
func TestListJobPhases_AbsentNarrow_NilRequestStillSafe(t *testing.T) {
	fake := newNarrowFake()
	r := NewPostgresJobPhaseRepository(fake, "job_phase")
	if _, err := r.ListJobPhases(context.Background(), nil); err != nil {
		t.Fatalf("nil request must not error: %v", err)
	}
	if fake.lastListParams != nil {
		t.Errorf("nil request must still yield nil ListParams, got %+v", fake.lastListParams)
	}
	if len(fake.exec.queries) != 0 {
		t.Errorf("nil request must issue no probe, got %v", fake.exec.queries)
	}
}

// ---- the narrow, when set, is applied or refused — never skipped ------------

// TestNarrowPhasesToGroup_ProbeBindsIdsWorkspaceGroup pins the bind sequence:
// $1 the DEDUPED owning-job ids, $2 the trusted workspace taken from
// identity.FromContext (the same source jobWorkspaceScope /
// filterPhasesByWorkspace / requireTrustedJobWorkspace use — NOT identity.Must),
// $3 the group id contributed by groupNarrowPredicate.
//
// Driven directly rather than through ListJobPhases because a workspace-bearing
// context makes filterPhasesByWorkspace probe FIRST, and this fake executor
// cannot manufacture a *sql.Rows to satisfy it. The end-to-end row sets are the
// DB-gated tests' job.
func TestNarrowPhasesToGroup_ProbeBindsIdsWorkspaceGroup(t *testing.T) {
	fake := newNarrowFake()
	r := &PostgresJobPhaseRepository{dbOps: fake}
	phases := []*pb.JobPhase{
		{Id: "p1", JobId: "j1"},
		{Id: "p2", JobId: "j1"}, // same job → deduped to ONE probed id
		{Id: "p3", JobId: "j2"},
	}

	out, err := r.narrowPhasesToGroup(wsCtx("ws-1"), phases, "grp-A")
	if err == nil {
		t.Fatal("a failing narrow probe must propagate as an error, never as a partial or unnarrowed result")
	}
	if out != nil {
		t.Errorf("a failed narrow must return no rows, got %d", len(out))
	}
	if len(fake.exec.queries) != 1 {
		t.Fatalf("expected exactly one narrow probe, got %d: %v", len(fake.exec.queries), fake.exec.queries)
	}
	if !strings.Contains(fake.exec.queries[0], "sgm_g.subscription_group_id = $3") {
		t.Errorf("probe did not carry the group predicate:\n%s", fake.exec.queries[0])
	}
	args := fake.exec.args[0]
	if len(args) != 3 {
		t.Fatalf("probe must bind exactly (ids, workspace, group), got %d args: %v", len(args), args)
	}
	ids, ok := args[0].(*pq.StringArray)
	if !ok {
		t.Fatalf("$1 must be a bound id array, got %T", args[0])
	}
	if len(*ids) != 2 {
		t.Errorf("$1 must carry DEDUPED owning-job ids, got %v", *ids)
	}
	if args[1] != "ws-1" {
		t.Errorf("$2 must be the trusted context workspace, got %v", args[1])
	}
	if args[2] != "grp-A" {
		t.Errorf("$3 must be the requested group, got %v", args[2])
	}
}

// TestListJobPhases_NarrowSet_ProbeFailureIsNotSwallowed: end to end, a narrowed
// list whose probe fails must surface the error. It must NOT degrade to the
// unnarrowed set and must NOT return an empty success that a caller would read
// as "this group has no phases".
func TestListJobPhases_NarrowSet_ProbeFailureIsNotSwallowed(t *testing.T) {
	fake := newNarrowFake(
		map[string]any{"id": "p1", "jobId": "j1"},
		map[string]any{"id": "p2", "jobId": "j2"},
	)
	r := NewPostgresJobPhaseRepository(fake, "job_phase")

	resp, err := r.ListJobPhases(wsCtx("ws-1"), &pb.ListJobPhasesRequest{SubscriptionGroupId: strptr("grp-A")})
	if err == nil {
		t.Fatalf("probe failure must propagate; got success with %d rows", len(resp.GetData()))
	}
	if resp != nil {
		t.Errorf("a failed narrowed list must not return a response, got %+v", resp)
	}
}

// TestListJobPhases_NarrowWithoutTrustedWorkspace_FailsClosed is the R-1
// regression (docs/plan/20260729-report-card-render-gate-group-grain/progress.md).
// filterPhasesByWorkspace PASSES THROUGH when the context carries no workspace;
// the narrow must NOT, because the predicate binds sgm_g.workspace_id and a
// pass-through would hand back the UNNARROWED set under success=true.
func TestListJobPhases_NarrowWithoutTrustedWorkspace_FailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{
		{"no identity at all", context.Background()},
		{"identity with empty workspace", wsCtx("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newNarrowFake(map[string]any{"id": "p1", "jobId": "j1"})
			r := NewPostgresJobPhaseRepository(fake, "job_phase")

			resp, err := r.ListJobPhases(tc.ctx, &pb.ListJobPhasesRequest{SubscriptionGroupId: strptr("grp-A")})
			if err == nil {
				t.Fatalf("an unappliable narrow must error; got a success with %d rows", len(resp.GetData()))
			}
			if !strings.Contains(err.Error(), "trusted workspace") {
				t.Errorf("error must name the missing workspace, got %q", err)
			}
			if len(fake.exec.queries) != 0 {
				t.Errorf("must refuse BEFORE probing, got %v", fake.exec.queries)
			}
		})
	}
}

// TestNarrowPhasesToGroup_EmptyGroupIsRefusedNotIgnored keeps the R-1 hole shut at
// the seam itself: reaching narrowPhasesToGroup with "" must never mean "no
// narrow". groupNarrowPredicate("") returns an EMPTY fragment, so a caller that
// forwarded it blindly would run a plain workspace query and call the result
// narrowed.
func TestNarrowPhasesToGroup_EmptyGroupIsRefusedNotIgnored(t *testing.T) {
	fake := newNarrowFake()
	r := &PostgresJobPhaseRepository{dbOps: fake}
	phases := []*pb.JobPhase{{Id: "p1", JobId: "j1"}}

	out, err := r.narrowPhasesToGroup(wsCtx("ws-1"), phases, "")
	if err == nil {
		t.Fatalf("empty group must be refused, got %d rows back", len(out))
	}
	if len(fake.exec.queries) != 0 {
		t.Errorf("must refuse before probing, got %v", fake.exec.queries)
	}
}

// TestJobGroupNarrowProbeSQL_ShapeAndPlaceholders pins the emitted probe: it must
// reuse groupNarrowPredicate verbatim (all five terms), bind ids/workspace/group
// at $1/$2/$3, bind the workspace to BOTH j.workspace_id and sgm_g.workspace_id,
// and refuse to emit for an empty group.
func TestJobGroupNarrowProbeSQL_ShapeAndPlaceholders(t *testing.T) {
	if q, args, ok := jobGroupNarrowProbeSQL(""); ok || q != "" || args != nil {
		t.Fatalf("empty group must decline to emit, got ok=%v q=%q args=%v", ok, q, args)
	}

	q, args, ok := jobGroupNarrowProbeSQL("grp-A")
	if !ok {
		t.Fatal("a present group must emit a probe")
	}
	if len(args) != 1 || args[0] != "grp-A" {
		t.Fatalf("predicate must contribute exactly the group id, got %v", args)
	}
	for _, frag := range []string{
		"j.id = ANY($1)",
		"j.workspace_id = $2",
		"sgm_g.client_id = j.client_id",
		"sgm_g.subscription_id = j.origin_id", // the load-bearing delivery pin
		"sgm_g.subscription_group_id = $3",
		"sgm_g.workspace_id = $2", // same trusted workspace bind, no second arg
		"sgm_g.active = true",
	} {
		if !strings.Contains(q, frag) {
			t.Errorf("probe SQL missing %q:\n%s", frag, q)
		}
	}
	// The narrow narrows by GROUP and nothing else — no smuggled lifecycle terms.
	for _, forbidden := range []string{"jp.active", "approval_status", "template_phase_id"} {
		if strings.Contains(q, forbidden) {
			t.Errorf("probe SQL must not add %q — the narrow is group-only:\n%s", forbidden, q)
		}
	}
}

// TestNarrowPhasesToGroup_EmptyInputIsEmptyWithoutProbing: narrowing nothing is
// nothing, and needs no round trip. This is the ONLY no-probe success path — it
// cannot fail open because there was no row to wrongly keep.
func TestNarrowPhasesToGroup_EmptyInputIsEmptyWithoutProbing(t *testing.T) {
	fake := newNarrowFake()
	r := &PostgresJobPhaseRepository{dbOps: fake}

	out, err := r.narrowPhasesToGroup(wsCtx("ws-1"), nil, "grp-A")
	if err != nil {
		t.Fatalf("empty input must not error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("empty input must stay empty, got %d", len(out))
	}
	if len(fake.exec.queries) != 0 {
		t.Errorf("empty input must not probe, got %v", fake.exec.queries)
	}

	// Rows with no resolvable owning job cannot be PROVEN in-group → dropped.
	out2, err := r.narrowPhasesToGroup(wsCtx("ws-1"), []*pb.JobPhase{{Id: "p1"}, nil}, "grp-A")
	if err != nil {
		t.Fatalf("unresolvable-job input must not error: %v", err)
	}
	if len(out2) != 0 {
		t.Fatalf("phases with no owning job must be dropped fail-closed, got %d", len(out2))
	}
}
