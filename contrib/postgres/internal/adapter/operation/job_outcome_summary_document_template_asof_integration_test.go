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
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Q8 (attendance-v2 follow-up): the REAL before/after `as_of` resolver
// integration test. The pre-existing coverage
// (TestFindApplicableSQL_HasFailClosedPredicateShape) is SQL-shape-only — it
// never opens a connection or resolves against real rows. This test exercises
// FindApplicableJobOutcomeSummaryDocumentTemplate against Postgres with the
// exact post-publish data shape PublishJobOutcomeSummaryDocumentTemplate
// produces (v1's validity_end clipped to v2's validity_start T, v1 left
// PUBLISHED), asserting the half-open [validity_start, validity_end)
// semantics: as_of=T-1ms resolves v1, as_of=T resolves v2.
//
// Gated on TEST_DATABASE_URL (repo convention — operations_test.go /
// outcome_criteria_anchor_integration_test.go). ROLLBACK-ONLY: every row is
// written inside one RunInTransaction that ends with an intentional-rollback
// sentinel; the resolver participates via the adapter's tx-aware executor(ctx)
// seam, so no committed mutation ever reaches the target database.

const jodtestWS = "jodtest-ws"

func openJODTIntegrationDB(t *testing.T) *sql.DB {
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
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.job_outcome_summary_document_template')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("job_outcome_summary_document_template not present")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestIntegration_AsOfResolver_BeforeAfterPublishBoundary seeds (in ONE
// rolled-back transaction) the exact two-published-version lineage the publish
// transaction leaves behind, then drives the REAL resolver through the
// boundary:
//
//	v1: PUBLISHED, validity [2026-01-01, T)   (validity_end clipped by publish)
//	v2: PUBLISHED, validity [T, NULL)
//
//	as_of = T-1ms   -> v1
//	as_of = T       -> v2   (half-open boundary: start inclusive, end exclusive)
//	as_of < v1 start -> not found
//
// Then the two-candidate LIMIT-2 leg (overlapping validity -> newest version
// deterministically wins), the DB-level ambiguity backstop (an equal-version
// PUBLISHED sibling is REJECTED by uq_jos_doc_tmpl_pub_version — the
// structural guarantee the resolver's LIMIT-2 guard is defense-in-depth for),
// and the foreign-workspace fail-closed leg.
func TestIntegration_AsOfResolver_BeforeAfterPublishBoundary(t *testing.T) {
	db := openJODTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(wsOps, "job_outcome_summary_document_template").(*PostgresJobOutcomeSummaryDocumentTemplateRepository)

	boundary := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // T — v2's validity_start
	v1Start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rollback := fmt.Errorf("jodtest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := ocitestExec(txCtx, wsOps, q, args...)
			return err
		}

		// Seed inside the tx: workspace, one doc-template artifact per version,
		// and the post-publish two-version lineage (workspace-default bucket,
		// price_schedule_id NULL).
		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, jodtestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('jodtest-dt-1', $1, 'jodtest v1', 'docx', 'report_card', 'templates', 'templates/report_card/jodtest-dt-1.docx', 'active', true),
		        ('jodtest-dt-2', $1, 'jodtest v2', 'docx', 'report_card', 'templates', 'templates/report_card/jodtest-dt-2.docx', 'active', true)`,
			jodtestWS); err != nil {
			return fmt.Errorf("seed document_templates: %w", err)
		}
		if err := exec(`INSERT INTO job_outcome_summary_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, version, version_status, validity_start, validity_end, active)
		        VALUES
		        ('jodtest-b-v1', $1, 'jodtest-dt-1', NULL, 1, $2, $3, $4, true),
		        ('jodtest-b-v2', $1, 'jodtest-dt-2', NULL, 2, $2, $4, NULL, true)`,
			jodtestWS, versionStatusPublished, v1Start, boundary); err != nil {
			return fmt.Errorf("seed bindings: %w", err)
		}

		idCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "jodtest-user", WorkspaceID: jodtestWS})
		resolve := func(asOf time.Time) (*pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse, error) {
			return repo.FindApplicableJobOutcomeSummaryDocumentTemplate(idCtx, &pb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{
				AsOf: timestamppb.New(asOf),
			})
		}

		// (a) as_of = T-1ms -> v1 (validity_end is EXCLUSIVE).
		resp, err := resolve(boundary.Add(-time.Millisecond))
		if err != nil {
			return fmt.Errorf("resolve before boundary: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jodtest-b-v1" {
			return fmt.Errorf("as_of=T-1ms must resolve v1, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}
		if resp.GetBinding().GetVersion() != 1 || resp.GetBinding().GetDocumentTemplate().GetId() != "jodtest-dt-1" {
			return fmt.Errorf("as_of=T-1ms hydration mismatch: version=%d dt=%q", resp.GetBinding().GetVersion(), resp.GetBinding().GetDocumentTemplate().GetId())
		}

		// (b) as_of = T -> v2 (validity_start is INCLUSIVE).
		resp, err = resolve(boundary)
		if err != nil {
			return fmt.Errorf("resolve at boundary: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jodtest-b-v2" {
			return fmt.Errorf("as_of=T must resolve v2, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}
		if resp.GetBinding().GetVersion() != 2 || resp.GetBinding().GetDocumentTemplate().GetId() != "jodtest-dt-2" {
			return fmt.Errorf("as_of=T hydration mismatch: version=%d dt=%q", resp.GetBinding().GetVersion(), resp.GetBinding().GetDocumentTemplate().GetId())
		}

		// (c) as_of before v1's validity_start -> no applicable binding.
		resp, err = resolve(v1Start.Add(-time.Millisecond))
		if err != nil {
			return fmt.Errorf("resolve before v1 start: %w", err)
		}
		if resp.GetFound() {
			return fmt.Errorf("as_of before v1's validity_start must resolve nothing, got %q", resp.GetBinding().GetId())
		}

		// (d) Two candidates inside the LIMIT-2 window: open v1's validity_end so
		// BOTH versions are valid at as_of=T. Distinct versions are NOT ambiguous
		// — the newest deterministically wins (ORDER BY match_rank, version DESC).
		if err := exec(`UPDATE job_outcome_summary_document_template SET validity_end = NULL WHERE id = 'jodtest-b-v1'`); err != nil {
			return fmt.Errorf("open v1 validity: %w", err)
		}
		resp, err = resolve(boundary)
		if err != nil {
			return fmt.Errorf("resolve with two candidates: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jodtest-b-v2" {
			return fmt.Errorf("with two valid candidates the newest version must win, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		// (e) Equal-rank equal-version ambiguity is STRUCTURALLY impossible: the
		// null-safe unique index rejects a second PUBLISHED version 2 in the same
		// (workspace, price_schedule) bucket. This is the DB-level backstop the
		// resolver's LIMIT-2 guard is defense-in-depth for. Savepoint so the tx
		// survives the expected constraint violation.
		if err := exec(`SAVEPOINT amb`); err != nil {
			return fmt.Errorf("savepoint: %w", err)
		}
		dupErr := exec(`INSERT INTO job_outcome_summary_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, version, version_status, validity_start, validity_end, active)
		        VALUES ('jodtest-b-dup', $1, 'jodtest-dt-2', NULL, 2, $2, $3, NULL, true)`,
			jodtestWS, versionStatusPublished, boundary)
		if dupErr == nil {
			return fmt.Errorf("an equal-version PUBLISHED sibling must be rejected by uq_jos_doc_tmpl_pub_version")
		}
		if !strings.Contains(dupErr.Error(), "uq_jos_doc_tmpl_pub_version") {
			return fmt.Errorf("ambiguous sibling must fail on uq_jos_doc_tmpl_pub_version, got: %v", dupErr)
		}
		if err := exec(`ROLLBACK TO SAVEPOINT amb`); err != nil {
			return fmt.Errorf("rollback to savepoint: %w", err)
		}

		// (f) A foreign-workspace identity resolves nothing (tenant gate in the
		// SQL predicate, sourced from trusted context).
		foreignCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "jodtest-user", WorkspaceID: "jodtest-other-ws"})
		resp, err = repo.FindApplicableJobOutcomeSummaryDocumentTemplate(foreignCtx, &pb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{
			AsOf: timestamppb.New(boundary),
		})
		if err != nil {
			return fmt.Errorf("foreign-workspace resolve: %w", err)
		}
		if resp.GetFound() {
			return fmt.Errorf("a foreign workspace identity must resolve nothing, got %q", resp.GetBinding().GetId())
		}

		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}

	// Rollback must leave zero residue.
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM job_outcome_summary_document_template WHERE id LIKE 'jodtest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in bindings: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM document_template WHERE id LIKE 'jodtest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in document_template: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM workspace WHERE id = $1`, jodtestWS).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in workspace: %d rows (err=%v)", n, err)
	}
}
