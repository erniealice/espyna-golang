//go:build postgresql

package operation

import (
	"database/sql"
	"os"
	"sort"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// Delivery-group narrow on ListJobPhases — the ROW-SET half of the contract
// (plan 20260729 Phase 1). READ-ONLY: these tests write nothing, so no rollback
// harness is needed. Gated on TEST_DATABASE_URL per repo convention.
//
// Three cases, and the third is the one that matters:
//
//	1. unnarrowed  → the row set is EXACTLY today's (the narrow is opt-in)
//	2. narrowed    → exactly that group's phases, a strict subset
//	3. narrowed to a group with NO members → EMPTY, with err == nil
//
// Case 3's name is deliberate: an EMPTY RESULT IS NOT A SKIPPED NARROW. On this
// adapter those two are different observable outcomes — a narrow that cannot be
// applied returns an ERROR (proven without a DB in
// job_phase_group_narrow_test.go), so `err == nil && len(Data) == 0` means the
// narrow ran and matched nothing.

type groupNarrowFixture struct {
	templatePhaseID string
	workspaceID     string
	groupID         string // one group that DOES cover part of the sheet
	totalPhases     int
	groupPhases     int
}

func openGroupNarrowDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL unusable: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// pickGroupNarrowFixture finds a template phase whose sheet spans AT LEAST TWO
// delivery groups — the whole point of the narrow — and small enough to fit the
// generic list's 100-row page so the assertion compares complete sets.
func pickGroupNarrowFixture(t *testing.T, db *sql.DB) groupNarrowFixture {
	t.Helper()
	const q = `
		WITH sheet AS (
			SELECT jp.template_phase_id AS pid,
			       j.workspace_id       AS ws,
			       count(DISTINCT jp.id) AS phases,
			       count(DISTINCT sgm.subscription_group_id) AS groups
			FROM job_phase jp
			JOIN job j ON j.id = jp.job_id
			JOIN subscription_group_member sgm
			  ON sgm.client_id = j.client_id
			 AND sgm.subscription_id = j.origin_id
			 AND sgm.workspace_id = j.workspace_id
			 AND sgm.active = true
			WHERE jp.active = true AND jp.template_phase_id IS NOT NULL
			GROUP BY 1, 2
			HAVING count(DISTINCT sgm.subscription_group_id) >= 2
			   AND count(DISTINCT jp.id) BETWEEN 2 AND 100
		)
		SELECT s.pid, s.ws, s.phases
		FROM sheet s
		ORDER BY s.phases ASC
		LIMIT 1`
	var fx groupNarrowFixture
	if err := db.QueryRow(q).Scan(&fx.templatePhaseID, &fx.workspaceID, &fx.totalPhases); err != nil {
		t.Skipf("no multi-group sheet on this DB: %v", err)
	}

	const gq = `
		SELECT sgm.subscription_group_id, count(DISTINCT jp.id)
		FROM job_phase jp
		JOIN job j ON j.id = jp.job_id
		JOIN subscription_group_member sgm
		  ON sgm.client_id = j.client_id
		 AND sgm.subscription_id = j.origin_id
		 AND sgm.workspace_id = j.workspace_id
		 AND sgm.active = true
		WHERE jp.active = true AND jp.template_phase_id = $1 AND j.workspace_id = $2
		GROUP BY 1
		ORDER BY 2 DESC
		LIMIT 1`
	if err := db.QueryRow(gq, fx.templatePhaseID, fx.workspaceID).Scan(&fx.groupID, &fx.groupPhases); err != nil {
		t.Skipf("cannot resolve a covering group for the fixture sheet: %v", err)
	}
	return fx
}

func listSheetPhaseIDs(t *testing.T, db *sql.DB, fx groupNarrowFixture, groupID string) []string {
	t.Helper()
	q := `
		SELECT jp.id
		FROM job_phase jp
		JOIN job j ON j.id = jp.job_id
		WHERE jp.active = true AND jp.template_phase_id = $1 AND j.workspace_id = $2`
	args := []any{fx.templatePhaseID, fx.workspaceID}
	if groupID != "" {
		q += `
		  AND EXISTS (
		    SELECT 1 FROM subscription_group_member sgm_g
		    WHERE sgm_g.client_id = j.client_id
		      AND sgm_g.subscription_id = j.origin_id
		      AND sgm_g.subscription_group_id = $3
		      AND sgm_g.workspace_id = $2
		      AND sgm_g.active = true
		  )`
		args = append(args, groupID)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		t.Fatalf("expectation query: %v", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan expectation: %v", err)
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func listViaAdapter(t *testing.T, db *sql.DB, fx groupNarrowFixture, groupID string) ([]string, error) {
	t.Helper()
	repo := NewPostgresJobPhaseRepository(postgresCore.NewWorkspaceAwareOperations(db), "job_phase")
	req := &pb.ListJobPhasesRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "template_phase_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Operator: commonpb.StringOperator_STRING_EQUALS,
				Value:    fx.templatePhaseID,
			}},
		}}},
		Pagination: &commonpb.PaginationRequest{Limit: 100},
	}
	if groupID != "" {
		req.SubscriptionGroupId = &groupID
	}
	resp, err := repo.ListJobPhases(wsCtx(fx.workspaceID), req)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resp.GetData()))
	for _, p := range resp.GetData() {
		ids = append(ids, p.GetId())
	}
	sort.Strings(ids)
	return ids, nil
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestListJobPhases_GroupNarrow_UnnarrowedRowSetUnchanged — case 1. With no group
// id the adapter must return the full sheet, exactly as before the narrow existed.
func TestListJobPhases_GroupNarrow_UnnarrowedRowSetUnchanged(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickGroupNarrowFixture(t, db)

	want := listSheetPhaseIDs(t, db, fx, "")
	got, err := listViaAdapter(t, db, fx, "")
	if err != nil {
		t.Fatalf("unnarrowed list errored: %v", err)
	}
	if !sameIDs(got, want) {
		t.Fatalf("unnarrowed row set changed: got %d ids, want %d\n got=%v\nwant=%v", len(got), len(want), got, want)
	}
	if len(got) == 0 {
		t.Fatal("fixture sheet is empty — the narrowed assertions below would be vacuous")
	}
}

// TestListJobPhases_GroupNarrow_ReturnsOnlyThatGroupsPhases — case 2. The result
// must equal the group's own phases AND be a strict subset of the sheet (the
// fixture spans >= 2 groups, so a non-strict subset means the narrow did nothing).
func TestListJobPhases_GroupNarrow_ReturnsOnlyThatGroupsPhases(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickGroupNarrowFixture(t, db)

	full := listSheetPhaseIDs(t, db, fx, "")
	want := listSheetPhaseIDs(t, db, fx, fx.groupID)
	got, err := listViaAdapter(t, db, fx, fx.groupID)
	if err != nil {
		t.Fatalf("narrowed list errored: %v", err)
	}
	if !sameIDs(got, want) {
		t.Fatalf("narrowed row set wrong: got %v want %v", got, want)
	}
	if len(got) == 0 {
		t.Fatalf("covering group %s returned nothing — fixture selection is broken", fx.groupID)
	}
	if len(got) >= len(full) {
		t.Fatalf("narrow did not narrow: %d of %d phases on a sheet spanning >= 2 groups", len(got), len(full))
	}
}

// TestListJobPhases_GroupNarrow_NoMemberGroupReturnsEmpty_NotUnnarrowed — case 3,
// and the name is the assertion. Narrowing to a group with no members must yield
// ZERO rows with NO error. That is DISTINCT from "the narrow was not applied":
// an unappliable narrow is an ERROR on this adapter (see
// TestListJobPhases_NarrowWithoutTrustedWorkspace_FailsClosed), never a quiet
// full set and never a quiet empty one. So a caller reading err == nil with zero
// rows knows the narrow RAN and MATCHED NOTHING.
func TestListJobPhases_GroupNarrow_NoMemberGroupReturnsEmpty_NotUnnarrowed(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickGroupNarrowFixture(t, db)

	full := listSheetPhaseIDs(t, db, fx, "")
	if len(full) == 0 {
		t.Skip("fixture sheet is empty")
	}

	// A syntactically valid id that owns no subscription_group_member row: a group
	// with no members, deterministically, on any database.
	const noMemberGroup = "00000000-0000-0000-0000-0000deadbeef"
	var memberRows int
	if err := db.QueryRow(
		`SELECT count(*) FROM subscription_group_member WHERE subscription_group_id = $1`,
		noMemberGroup,
	).Scan(&memberRows); err != nil {
		t.Fatalf("member-count precondition: %v", err)
	}
	if memberRows != 0 {
		t.Skipf("sentinel group id unexpectedly has %d members", memberRows)
	}

	got, err := listViaAdapter(t, db, fx, noMemberGroup)
	if err != nil {
		t.Fatalf("a narrow that matches nothing must SUCCEED with zero rows, not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("member-less group must return EMPTY, got %d rows: %v", len(got), got)
	}
	// The distinction, made mechanical: had the narrow been skipped, this call
	// would have returned the full sheet.
	if len(full) == 0 {
		t.Fatal("cannot distinguish empty-narrow from skipped-narrow on an empty sheet")
	}
}

// TestListJobPhases_GroupNarrow_ForeignWorkspaceGroupIsEmptyNotUnscoped: the
// predicate binds sgm_g.workspace_id to the CALLER's workspace, so a real group
// belonging to another tenant selects nothing rather than leaking its roster.
func TestListJobPhases_GroupNarrow_ForeignWorkspaceGroupIsEmptyNotUnscoped(t *testing.T) {
	db := openGroupNarrowDB(t)
	fx := pickGroupNarrowFixture(t, db)

	var foreignGroup string
	err := db.QueryRow(
		`SELECT sgm.subscription_group_id
		   FROM subscription_group_member sgm
		  WHERE sgm.active = true AND sgm.workspace_id <> $1
		  LIMIT 1`, fx.workspaceID).Scan(&foreignGroup)
	if err != nil {
		t.Skipf("no foreign-workspace group on this DB: %v", err)
	}

	got, listErr := listViaAdapter(t, db, fx, foreignGroup)
	if listErr != nil {
		t.Fatalf("foreign-workspace group must be empty, not an error: %v", listErr)
	}
	if len(got) != 0 {
		t.Fatalf("foreign-workspace group leaked %d rows: %v", len(got), got)
	}
}
