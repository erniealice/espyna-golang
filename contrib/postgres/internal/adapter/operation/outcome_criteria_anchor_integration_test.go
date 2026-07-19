//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// Real-Postgres integration coverage for the Q1 criteria_group anchor work
// (codex wave1-q1 FIX-FIRST 1/2/3/6). Gated on TEST_DATABASE_URL (repo
// convention — operations_test.go); target must have esqyma migrations through
// 20260718000004 applied (criteria_group with domain columns + trigger + FK +
// uq_criteria_group_domain_code).
//
// Every test namespaces its rows with the ocitest- prefix and cleans up (tx
// rollback where possible; explicit DELETE for the two-session commit legs,
// which necessarily commit to prove the constraint).

const ocitestWS = "ocitest-ws"

func openOCIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var ok string
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.criteria_group')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("criteria_group anchor not present (migrations through 20260718000004 required)")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ocitestEnsureWorkspace upserts the namespaced test workspace the coded rows
// reference (outcome_criteria.workspace_id carries an FK to workspace). It is
// committed (the race legs commit) and removed by ocitestCleanup.
func ocitestEnsureWorkspace(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, ocitestWS); err != nil {
		t.Fatalf("ensure ocitest workspace: %v", err)
	}
	t.Cleanup(func() { ocitestCleanup(t, db) })
}

// ocitestCleanupRows removes committed ocitest- rows + anchors (keeps the test
// workspace — used between race legs).
func ocitestCleanupRows(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM outcome_criteria WHERE id LIKE 'ocitest-%'`); err != nil {
		t.Fatalf("cleanup outcome_criteria: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM criteria_group WHERE id LIKE 'ocitest-%'`); err != nil {
		t.Fatalf("cleanup criteria_group: %v", err)
	}
}

// ocitestCleanup removes all committed ocitest- residue (rows, anchors, then
// the test workspace).
func ocitestCleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ocitestCleanupRows(t, db)
	if _, err := db.Exec(`DELETE FROM workspace WHERE id = $1`, ocitestWS); err != nil {
		t.Fatalf("cleanup workspace: %v", err)
	}
}

func ocitestInsertSQL() string {
	return `INSERT INTO outcome_criteria
	        (id, criteria_group_id, name, code, workspace_id, scope, active)
	        VALUES ($1, $2, $1, $3, $4, 'CRITERIA_SCOPE_WORKSPACE', $5)`
}

// ocitestExec runs raw SQL through the ops' transaction-aware executor so the
// statement joins the active RunInTransaction tx on ctx (same seam the
// adapter's purpose-built reads use).
func ocitestExec(ctx context.Context, ops interfaces.DatabaseOperation, query string, args ...any) (sql.Result, error) {
	ep, ok := ops.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	})
	if !ok {
		return nil, fmt.Errorf("ops does not expose GetExecutor")
	}
	return ep.GetExecutor(ctx).ExecContext(ctx, query, args...)
}

// TestIntegration_AnchorReads_And_PaginationOver100Rows is the real >100-row
// proof (finding 6A/6B): 150 same-group coded rows inserted in ONE rolled-back
// transaction; the REAL repository + generic core List must (a) return a
// DIFFERENT page 2, and (b) yield all 150 rows when pagination-exhausted with
// the code-validation loop's exact request shape. The purpose-built anchor
// point lookups must resolve the lineage code and domain owner from inside the
// same uncommitted transaction (executor(ctx) tx participation).
func TestIntegration_AnchorReads_And_PaginationOver100Rows(t *testing.T) {
	db := openOCIntegrationDB(t)
	ocitestEnsureWorkspace(t, db)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresOutcomeCriteriaRepository(wsOps, "outcome_criteria")

	// The purpose-built seam must be satisfied by the real adapter (structural
	// mirror of the unexported use-case interface).
	ar, ok := repo.(interface {
		LineageEstablishedCode(ctx context.Context, criteriaGroupID string) (string, error)
		CodeOwnerGroup(ctx context.Context, scopeKey, workspaceKey, industryKey, code string) (string, error)
		LineageClaimedDomain(ctx context.Context, criteriaGroupID string) (string, string, string, bool, error)
	})
	if !ok {
		t.Fatal("PostgresOutcomeCriteriaRepository must implement the lineageAnchorReader seam")
	}

	rollback := fmt.Errorf("ocitest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		// Insert 150 rows in one statement (trigger anchors on the first row;
		// same-statement FK visibility is the CASE3a-proven behavior).
		if _, err := ocitestExec(txCtx, wsOps,
			`INSERT INTO outcome_criteria (id, criteria_group_id, name, code, workspace_id, scope, active)
			 SELECT 'ocitest-pag-'||g, 'ocitest-grp-pag', 'ocitest '||g, 'ocitestpagcode', $1, 'CRITERIA_SCOPE_WORKSPACE', true
			 FROM generate_series(1,150) g`, ocitestWS); err != nil {
			return fmt.Errorf("seed 150 rows: %w", err)
		}

		// (a) page 2 must differ from page 1 through the REAL adapter+core.
		page := func(p int32) (*pb.ListOutcomeCriteriasResponse, error) {
			return repo.ListOutcomeCriterias(txCtx, &pb.ListOutcomeCriteriasRequest{
				Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
					{Field: "code", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: "ocitestpagcode", Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}},
					{Field: "active", FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: true}}},
				}},
				Pagination: &commonpb.PaginationRequest{Limit: 100, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: p}}},
			})
		}
		p1, err := page(1)
		if err != nil {
			return fmt.Errorf("page 1: %w", err)
		}
		p2, err := page(2)
		if err != nil {
			return fmt.Errorf("page 2: %w", err)
		}
		if len(p1.GetData()) != 100 {
			return fmt.Errorf("page 1 returned %d rows, want 100", len(p1.GetData()))
		}
		if len(p2.GetData()) != 50 {
			return fmt.Errorf("page 2 returned %d rows, want 50 — pagination NOT forwarded (finding 6A)", len(p2.GetData()))
		}
		if p1.GetData()[0].GetId() == p2.GetData()[0].GetId() {
			return fmt.Errorf("page 2 re-read page 1 (finding 6A row-cap regression)")
		}

		// (b) pagination-exhausting read (the fallback loop's exact shape):
		// active=true + active=false passes, 100-row pages, must total 150.
		total := 0
		for _, active := range []bool{true, false} {
			for pg := int32(1); ; pg++ {
				if pg > 50 {
					return fmt.Errorf("page bound exceeded")
				}
				resp, err := repo.ListOutcomeCriterias(txCtx, &pb.ListOutcomeCriteriasRequest{
					Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
						{Field: "code", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: "ocitestpagcode", Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}},
						{Field: "active", FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: active}}},
					}},
					Pagination: &commonpb.PaginationRequest{Limit: 100, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: pg}}},
				})
				if err != nil {
					return err
				}
				total += len(resp.GetData())
				if len(resp.GetData()) < 100 {
					break
				}
			}
		}
		if total != 150 {
			return fmt.Errorf("pagination-exhausted read returned %d rows, want 150 (truncation)", total)
		}

		// (c) purpose-built point lookups, inside the SAME uncommitted tx.
		code, err := ar.LineageEstablishedCode(txCtx, "ocitest-grp-pag")
		if err != nil {
			return fmt.Errorf("LineageEstablishedCode: %w", err)
		}
		if code != "ocitestpagcode" {
			return fmt.Errorf("LineageEstablishedCode = %q, want ocitestpagcode", code)
		}
		owner, err := ar.CodeOwnerGroup(txCtx, "CRITERIA_SCOPE_WORKSPACE", ocitestWS, "", "ocitestpagcode")
		if err != nil {
			return fmt.Errorf("CodeOwnerGroup: %w", err)
		}
		if owner != "ocitest-grp-pag" {
			return fmt.Errorf("CodeOwnerGroup = %q, want ocitest-grp-pag", owner)
		}
		// NEW-1: the domain-claim read must resolve the trigger-stamped claim
		// from inside the same uncommitted transaction.
		scopeKey, wsKey, indKey, claimed, err := ar.LineageClaimedDomain(txCtx, "ocitest-grp-pag")
		if err != nil {
			return fmt.Errorf("LineageClaimedDomain: %w", err)
		}
		if !claimed || scopeKey != "CRITERIA_SCOPE_WORKSPACE" || wsKey != ocitestWS || indKey != "" {
			return fmt.Errorf("LineageClaimedDomain = (%q,%q,%q,claimed=%v), want (CRITERIA_SCOPE_WORKSPACE,%s,'',true)", scopeKey, wsKey, indKey, claimed, ocitestWS)
		}
		// An unanchored lineage reports claimed=false.
		if _, _, _, claimed, err := ar.LineageClaimedDomain(txCtx, "ocitest-grp-none"); err != nil || claimed {
			return fmt.Errorf("LineageClaimedDomain on an unanchored lineage must be (claimed=false, nil err), got claimed=%v err=%v", claimed, err)
		}
		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}

	// Rollback must leave zero residue.
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM outcome_criteria WHERE id LIKE 'ocitest-pag-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM criteria_group WHERE id = 'ocitest-grp-pag'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback anchor residue: %d rows (err=%v)", n, err)
	}
}

// twoSessionRace runs writerA and writerB on two REAL connections: A begins and
// executes, B begins and executes concurrently (expected to block on unique/PK
// arbitration), then A commits or rolls back. Returns B's terminal error (nil =
// B's statement succeeded after A resolved). B's tx is always rolled back.
func twoSessionRace(t *testing.T, db *sql.DB, commitA bool, insertA, insertB func(tx *sql.Tx) error) error {
	t.Helper()
	ctx := context.Background()

	connA, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("connA: %v", err)
	}
	defer connA.Close()
	connB, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("connB: %v", err)
	}
	defer connB.Close()

	txA, err := connA.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("beginA: %v", err)
	}
	if err := insertA(txA); err != nil {
		txA.Rollback()
		t.Fatalf("insertA must succeed first: %v", err)
	}

	txB, err := connB.BeginTx(ctx, nil)
	if err != nil {
		txA.Rollback()
		t.Fatalf("beginB: %v", err)
	}
	bDone := make(chan error, 1)
	go func() { bDone <- insertB(txB) }()

	// B must be blocked behind A's uncommitted claim, not already resolved.
	select {
	case err := <-bDone:
		txA.Rollback()
		txB.Rollback()
		t.Fatalf("session B resolved (%v) WITHOUT waiting on session A's uncommitted claim — no DB serialization point", err)
	case <-time.After(500 * time.Millisecond):
		// blocked, as required
	}

	if commitA {
		if err := txA.Commit(); err != nil {
			t.Fatalf("commitA: %v", err)
		}
	} else {
		if err := txA.Rollback(); err != nil {
			t.Fatalf("rollbackA: %v", err)
		}
	}

	var bErr error
	select {
	case bErr = <-bDone:
	case <-time.After(10 * time.Second):
		txB.Rollback()
		t.Fatal("session B still blocked 10s after session A resolved")
	}
	txB.Rollback()
	return bErr
}

// TestIntegration_TwoSession_CrossGroup_SameDomainCode is the genuine
// two-connection proof for finding 2B: two DIFFERENT groups claiming the SAME
// normalized (scope, workspace, industry, code) concurrently. The DB (not app
// code) must serialize them on uq_criteria_group_domain_code: loser fails when
// the winner commits; loser proceeds when the winner rolls back.
func TestIntegration_TwoSession_CrossGroup_SameDomainCode(t *testing.T) {
	db := openOCIntegrationDB(t)
	ocitestEnsureWorkspace(t, db)
	ocitestCleanupRows(t, db)

	insA := func(tx *sql.Tx) error {
		_, err := tx.Exec(ocitestInsertSQL(), "ocitest-race-a", "ocitest-grp-xa", "ocitestracecode", ocitestWS, true)
		return err
	}
	insB := func(tx *sql.Tx) error {
		_, err := tx.Exec(ocitestInsertSQL(), "ocitest-race-b", "ocitest-grp-xb", "ocitestracecode", ocitestWS, true)
		return err
	}

	// Winner commits -> loser MUST fail on the domain claim.
	if bErr := twoSessionRace(t, db, true, insA, insB); bErr == nil {
		t.Fatal("cross-group same-domain+code concurrent draft was ACCEPTED after the winner committed — finding 2B backstop missing")
	} else if !strings.Contains(bErr.Error(), "uq_criteria_group_domain_code") {
		t.Fatalf("loser must fail on uq_criteria_group_domain_code, got: %v", bErr)
	}
	ocitestCleanupRows(t, db)

	// Winner rolls back -> loser legitimately becomes the owner.
	if bErr := twoSessionRace(t, db, false, insA, insB); bErr != nil {
		t.Fatalf("after the winner rolled back, the second session must own the claim, got: %v", bErr)
	}
}

// TestIntegration_TwoSession_SameGroup_DifferentCode covers finding 2C's other
// leg on two REAL connections: one new group, two divergent codes. The PK
// arbitration on criteria_group.id serializes the anchor; the loser's divergent
// code must then fail the composite FK when the winner commits, and must win
// legitimately when the winner rolls back.
func TestIntegration_TwoSession_SameGroup_DifferentCode(t *testing.T) {
	db := openOCIntegrationDB(t)
	ocitestEnsureWorkspace(t, db)
	ocitestCleanupRows(t, db)

	insA := func(tx *sql.Tx) error {
		_, err := tx.Exec(ocitestInsertSQL(), "ocitest-sg-a", "ocitest-grp-sg", "ocitestcodea", ocitestWS, true)
		return err
	}
	insB := func(tx *sql.Tx) error {
		_, err := tx.Exec(ocitestInsertSQL(), "ocitest-sg-b", "ocitest-grp-sg", "ocitestcodeb", ocitestWS, true)
		return err
	}

	// Winner commits -> loser's divergent code trips the composite FK.
	if bErr := twoSessionRace(t, db, true, insA, insB); bErr == nil {
		t.Fatal("same-group divergent-code concurrent draft was ACCEPTED after the winner committed")
	} else if !strings.Contains(bErr.Error(), "fk_outcome_criteria_group_anchor") {
		t.Fatalf("loser must fail on the composite FK, got: %v", bErr)
	}
	ocitestCleanupRows(t, db)

	// Winner rolls back -> loser's code legitimately anchors the group.
	if bErr := twoSessionRace(t, db, false, insA, insB); bErr != nil {
		t.Fatalf("after the winner rolled back, the second code must anchor the group, got: %v", bErr)
	}
}

// TestIntegration_MixedNullLineage pins the redefined NULL contract (finding
// 1A resolution) at the DATABASE level: an anchored group ACCEPTS a NULL-code
// version (MATCH SIMPLE exemption — coded-versions-agree contract), while a
// divergent CODED version is rejected by the composite FK. Fully rolled back.
func TestIntegration_MixedNullLineage(t *testing.T) {
	db := openOCIntegrationDB(t)
	ocitestEnsureWorkspace(t, db)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(ocitestInsertSQL(), "ocitest-null-1", "ocitest-grp-null", "ocitestnullcode", ocitestWS, true); err != nil {
		t.Fatalf("coded row must anchor the group: %v", err)
	}
	// NULL-code sibling inside the anchored group -> PERMITTED by contract.
	if _, err := tx.Exec(`INSERT INTO outcome_criteria (id, criteria_group_id, name, workspace_id, scope, active)
	                      VALUES ('ocitest-null-2', 'ocitest-grp-null', 'ocitest null sibling', $1, 'CRITERIA_SCOPE_WORKSPACE', true)`, ocitestWS); err != nil {
		t.Fatalf("a NULL-code version inside an anchored group is permitted (coded-versions-agree contract), got: %v", err)
	}
	// Divergent CODED sibling -> rejected by the composite FK (savepoint so the
	// tx survives for the assertion).
	if _, err := tx.Exec(`SAVEPOINT diverge`); err != nil {
		t.Fatalf("savepoint: %v", err)
	}
	_, err = tx.Exec(ocitestInsertSQL(), "ocitest-null-3", "ocitest-grp-null", "ocitestothercode", ocitestWS, true)
	if err == nil {
		t.Fatal("a DIVERGENT coded version inside an anchored group must be rejected")
	}
	if !strings.Contains(err.Error(), "fk_outcome_criteria_group_anchor") {
		t.Fatalf("divergent coded version must fail on the composite FK, got: %v", err)
	}
	if _, err := tx.Exec(`ROLLBACK TO SAVEPOINT diverge`); err != nil {
		t.Fatalf("rollback to savepoint: %v", err)
	}
}

// TestIntegration_TrustedWorkspaceSeam proves the WorkspaceAware List seam that
// makes finding 10 exploitable-then-fixed: with identity workspace w1 on the
// context, the repository returns w1's coded rows regardless of any
// client-claimed workspace (the decorator injects the TRUSTED filter), so the
// use case comparing a client-supplied workspace against these rows was the
// bug; stamping from the same context source closes it (unit-tested in
// code_validation_test.go). Also proves a foreign-workspace identity cannot
// see the rows at all. Fully rolled back.
func TestIntegration_TrustedWorkspaceSeam(t *testing.T) {
	db := openOCIntegrationDB(t)
	ocitestEnsureWorkspace(t, db)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresOutcomeCriteriaRepository(wsOps, "outcome_criteria")

	rollback := fmt.Errorf("ocitest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		if _, err := ocitestExec(txCtx, wsOps, ocitestInsertSQL(), "ocitest-seam-1", "ocitest-grp-seam", "ocitestseamcode", ocitestWS, true); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
		listWith := func(ws string) (int, error) {
			idCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "ocitest-user", WorkspaceID: ws})
			resp, err := repo.ListOutcomeCriterias(idCtx, &pb.ListOutcomeCriteriasRequest{
				Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
					{Field: "code", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: "ocitestseamcode", Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}},
				}},
			})
			if err != nil {
				return 0, err
			}
			return len(resp.GetData()), nil
		}
		if n, err := listWith(ocitestWS); err != nil || n != 1 {
			return fmt.Errorf("owner workspace must see its coded row (n=%d err=%v)", n, err)
		}
		if n, err := listWith("ocitest-other-ws"); err != nil || n != 0 {
			return fmt.Errorf("a foreign workspace identity must see nothing (n=%d err=%v)", n, err)
		}
		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}
}
