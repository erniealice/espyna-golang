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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The sheet-family (grade-sheet) binding resolver integration test (JOSDT sibling,
// 20260720). Exercises FindApplicableJobTemplateDocumentTemplate against Postgres
// with the exact post-publish data shape the publish transaction produces, plus
// the NEW job_category axis: most-specific-wins across (category, schedule) and
// the per-(schedule,category)-bucket version uniqueness backstop.
//
// Gated on TEST_DATABASE_URL. ROLLBACK-ONLY: every row is written inside one
// RunInTransaction that ends with an intentional-rollback sentinel; the resolver
// participates via the adapter's tx-aware executor(ctx) seam, so no committed
// mutation ever reaches the target database. (Publish/Delete use r.db directly
// — a fresh transaction — so, exactly as the JOSDT integration test does, they
// are covered by the DB-free lifecycle tests + the unique-index backstop below,
// never committed here.)

const jtdttestWS = "jtdttest-ws"

func openJTDTIntegrationDB(t *testing.T) *sql.DB {
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
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.job_template_document_template')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("job_template_document_template not present")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestIntegration_JTDT_AsOfResolver_BeforeAfterPublishBoundary drives the REAL
// resolver through the half-open [validity_start, validity_end) boundary of the
// workspace-wide (category NULL, schedule NULL) bucket, then the two-candidate
// newest-wins leg, the DB-level equal-version ambiguity backstop
// (uq_jt_doc_tmpl_pub_version), and the foreign-workspace fail-closed leg.
func TestIntegration_JTDT_AsOfResolver_BeforeAfterPublishBoundary(t *testing.T) {
	db := openJTDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresJobTemplateDocumentTemplateRepository(wsOps, "job_template_document_template").(*PostgresJobTemplateDocumentTemplateRepository)

	boundary := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // T — v2's validity_start
	v1Start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rollback := fmt.Errorf("jtdttest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := ocitestExec(txCtx, wsOps, q, args...)
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, jtdttestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('jtdttest-dt-1', $1, 'jtdttest v1', 'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/jtdttest-dt-1.docx', 'active', true),
		        ('jtdttest-dt-2', $1, 'jtdttest v2', 'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/jtdttest-dt-2.docx', 'active', true)`,
			jtdttestWS); err != nil {
			return fmt.Errorf("seed document_templates: %w", err)
		}
		if err := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES
		        ('jtdttest-b-v1', $1, 'jtdttest-dt-1', NULL, NULL, 1, $2, $3, $4, true),
		        ('jtdttest-b-v2', $1, 'jtdttest-dt-2', NULL, NULL, 2, $2, $4, NULL, true)`,
			jtdttestWS, versionStatusPublished, v1Start, boundary); err != nil {
			return fmt.Errorf("seed bindings: %w", err)
		}

		idCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "jtdttest-user", WorkspaceID: jtdttestWS})
		resolve := func(asOf time.Time) (*pb.FindApplicableJobTemplateDocumentTemplateResponse, error) {
			return repo.FindApplicableJobTemplateDocumentTemplate(idCtx, &pb.FindApplicableJobTemplateDocumentTemplateRequest{
				AsOf: timestamppb.New(asOf),
			})
		}

		// (a) as_of = T-1ms -> v1 (validity_end is EXCLUSIVE).
		resp, err := resolve(boundary.Add(-time.Millisecond))
		if err != nil {
			return fmt.Errorf("resolve before boundary: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jtdttest-b-v1" {
			return fmt.Errorf("as_of=T-1ms must resolve v1, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		// (b) as_of = T -> v2 (validity_start is INCLUSIVE).
		resp, err = resolve(boundary)
		if err != nil {
			return fmt.Errorf("resolve at boundary: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jtdttest-b-v2" {
			return fmt.Errorf("as_of=T must resolve v2, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		// (c) as_of before v1's validity_start -> no applicable binding.
		resp, err = resolve(v1Start.Add(-time.Millisecond))
		if err != nil {
			return fmt.Errorf("resolve before v1 start: %w", err)
		}
		if resp.GetFound() {
			return fmt.Errorf("as_of before v1's validity_start must resolve nothing, got %q", resp.GetBinding().GetId())
		}

		// (d) Two valid candidates -> newest version deterministically wins.
		if err := exec(`UPDATE job_template_document_template SET validity_end = NULL WHERE id = 'jtdttest-b-v1'`); err != nil {
			return fmt.Errorf("open v1 validity: %w", err)
		}
		resp, err = resolve(boundary)
		if err != nil {
			return fmt.Errorf("resolve with two candidates: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jtdttest-b-v2" {
			return fmt.Errorf("with two valid candidates the newest version must win, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		// (e) Equal-rank equal-version ambiguity is STRUCTURALLY impossible: the
		// null-safe unique index rejects a second PUBLISHED version 2 in the same
		// (workspace, schedule, category) bucket.
		if err := exec(`SAVEPOINT amb`); err != nil {
			return fmt.Errorf("savepoint: %w", err)
		}
		dupErr := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES ('jtdttest-b-dup', $1, 'jtdttest-dt-2', NULL, NULL, 2, $2, $3, NULL, true)`,
			jtdttestWS, versionStatusPublished, boundary)
		if dupErr == nil {
			return fmt.Errorf("an equal-version PUBLISHED sibling must be rejected by uq_jt_doc_tmpl_pub_version")
		}
		if !strings.Contains(dupErr.Error(), "uq_jt_doc_tmpl_pub_version") {
			return fmt.Errorf("ambiguous sibling must fail on uq_jt_doc_tmpl_pub_version, got: %v", dupErr)
		}
		if err := exec(`ROLLBACK TO SAVEPOINT amb`); err != nil {
			return fmt.Errorf("rollback to savepoint: %w", err)
		}

		// (f) A foreign-workspace identity resolves nothing (tenant gate in SQL).
		foreignCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "jtdttest-user", WorkspaceID: "jtdttest-other-ws"})
		resp, err = repo.FindApplicableJobTemplateDocumentTemplate(foreignCtx, &pb.FindApplicableJobTemplateDocumentTemplateRequest{
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

	assertNoJTDTResidue(t, db)
}

// TestIntegration_JTDT_CategoryRankOrdering pins the 4-tier most-specific-wins
// resolution across the (category, schedule) axes (category ≻ schedule, per Q2):
//
//	rank 1  category exact  + schedule fallback   > rank 2  category fallback + schedule exact
//	rank 0  category exact  + schedule exact       wins over all
//
// so a category-specific binding beats a schedule-specific one, and an exact
// (category+schedule) binding beats both.
func TestIntegration_JTDT_CategoryRankOrdering(t *testing.T) {
	db := openJTDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresJobTemplateDocumentTemplateRepository(wsOps, "job_template_document_template").(*PostgresJobTemplateDocumentTemplateRepository)

	asOf := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	rollback := fmt.Errorf("jtdttest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := ocitestExec(txCtx, wsOps, q, args...)
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, jtdttestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, name, workspace_id, code) VALUES
		        ('jtdttest-cat-academic', 'Academic', $1, 'academic')`, jtdttestWS); err != nil {
			return fmt.Errorf("seed job_category: %w", err)
		}
		if err := exec(`INSERT INTO price_schedule (id, workspace_id, name) VALUES
		        ('jtdttest-ps-2026', $1, 'AY 2026')`, jtdttestWS); err != nil {
			return fmt.Errorf("seed price_schedule: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('jtdttest-dt-cat',  $1, 'cat', 'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/cat.docx',  'active', true),
		        ('jtdttest-dt-sch',  $1, 'sch', 'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/sch.docx',  'active', true),
		        ('jtdttest-dt-ws',   $1, 'ws',  'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/ws.docx',   'active', true),
		        ('jtdttest-dt-both', $1, 'both','docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/both.docx', 'active', true)`,
			jtdttestWS); err != nil {
			return fmt.Errorf("seed document_templates: %w", err)
		}
		// Three published candidates: category-scoped (rank 1), schedule-scoped
		// (rank 2), workspace-wide (rank 3).
		if err := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, active)
		        VALUES
		        ('jtdttest-b-cat', $1, 'jtdttest-dt-cat', NULL,               'jtdttest-cat-academic', 1, $2, true),
		        ('jtdttest-b-sch', $1, 'jtdttest-dt-sch', 'jtdttest-ps-2026', NULL,                    1, $2, true),
		        ('jtdttest-b-ws',  $1, 'jtdttest-dt-ws',  NULL,               NULL,                    1, $2, true)`,
			jtdttestWS, versionStatusPublished); err != nil {
			return fmt.Errorf("seed rank bindings: %w", err)
		}

		idCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "jtdttest-user", WorkspaceID: jtdttestWS})
		resolve := func() (*pb.FindApplicableJobTemplateDocumentTemplateResponse, error) {
			return repo.FindApplicableJobTemplateDocumentTemplate(idCtx, &pb.FindApplicableJobTemplateDocumentTemplateRequest{
				PriceScheduleId: strptr("jtdttest-ps-2026"),
				JobCategoryId:   strptr("jtdttest-cat-academic"),
				AsOf:            timestamppb.New(asOf),
			})
		}

		// category-specific (rank 1) beats schedule-specific (rank 2) beats
		// workspace-wide (rank 3).
		resp, err := resolve()
		if err != nil {
			return fmt.Errorf("resolve rank ordering: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jtdttest-b-cat" {
			return fmt.Errorf("category-specific binding (rank 1) must win over schedule-specific (rank 2), got found=%v id=%q",
				resp.GetFound(), resp.GetBinding().GetId())
		}
		if resp.GetBinding().GetJobCategory().GetCode() != "academic" {
			return fmt.Errorf("resolved binding must hydrate its job_category (code=academic), got %q",
				resp.GetBinding().GetJobCategory().GetCode())
		}

		// Add the exact (category+schedule) binding: rank 0 wins over all.
		if err := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, active)
		        VALUES ('jtdttest-b-both', $1, 'jtdttest-dt-both', 'jtdttest-ps-2026', 'jtdttest-cat-academic', 1, $2, true)`,
			jtdttestWS, versionStatusPublished); err != nil {
			return fmt.Errorf("seed exact binding: %w", err)
		}
		resp, err = resolve()
		if err != nil {
			return fmt.Errorf("resolve exact: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "jtdttest-b-both" {
			return fmt.Errorf("exact (category+schedule) binding (rank 0) must win over all, got found=%v id=%q",
				resp.GetFound(), resp.GetBinding().GetId())
		}

		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}

	assertNoJTDTResidue(t, db)
}

// TestIntegration_JTDT_VersionBucketIsPerCategory pins that the published-version
// uniqueness bucket includes COALESCE(job_category_id,'') — the same tuple the
// publish transaction allocates MAX+1 within. A version-1 PUBLISHED binding in
// the category-academic bucket coexists with a version-1 PUBLISHED binding in the
// workspace-wide (category NULL) bucket (independent buckets), but a SECOND
// version-1 in the SAME category bucket is rejected by uq_jt_doc_tmpl_pub_version.
func TestIntegration_JTDT_VersionBucketIsPerCategory(t *testing.T) {
	db := openJTDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)

	rollback := fmt.Errorf("jtdttest: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := ocitestExec(txCtx, wsOps, q, args...)
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, jtdttestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, name, workspace_id, code) VALUES
		        ('jtdttest-cat-academic', 'Academic', $1, 'academic')`, jtdttestWS); err != nil {
			return fmt.Errorf("seed job_category: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES ('jtdttest-dt-1', $1, 'dt', 'docx', 'outcome_matrix', 'templates', 'templates/outcome_matrix/dt.docx', 'active', true)`,
			jtdttestWS); err != nil {
			return fmt.Errorf("seed document_template: %w", err)
		}
		// v1 in the workspace-wide bucket AND v1 in the category-academic bucket:
		// distinct buckets → both accepted.
		if err := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, active)
		        VALUES
		        ('jtdttest-b-ws-v1',  $1, 'jtdttest-dt-1', NULL, NULL,                    1, $2, true),
		        ('jtdttest-b-cat-v1', $1, 'jtdttest-dt-1', NULL, 'jtdttest-cat-academic', 1, $2, true)`,
			jtdttestWS, versionStatusPublished); err != nil {
			return fmt.Errorf("distinct-bucket v1s must both be accepted: %w", err)
		}

		// A SECOND v1 in the category-academic bucket → rejected.
		if err := exec(`SAVEPOINT dup`); err != nil {
			return fmt.Errorf("savepoint: %w", err)
		}
		dupErr := exec(`INSERT INTO job_template_document_template
		        (id, workspace_id, document_template_id, price_schedule_id, job_category_id, version, version_status, active)
		        VALUES ('jtdttest-b-cat-dup', $1, 'jtdttest-dt-1', NULL, 'jtdttest-cat-academic', 1, $2, true)`,
			jtdttestWS, versionStatusPublished)
		if dupErr == nil {
			return fmt.Errorf("a duplicate version-1 in the same category bucket must be rejected by uq_jt_doc_tmpl_pub_version")
		}
		if !strings.Contains(dupErr.Error(), "uq_jt_doc_tmpl_pub_version") {
			return fmt.Errorf("same-bucket duplicate must fail on uq_jt_doc_tmpl_pub_version, got: %v", dupErr)
		}
		if err := exec(`ROLLBACK TO SAVEPOINT dup`); err != nil {
			return fmt.Errorf("rollback to savepoint: %w", err)
		}

		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected the intentional rollback sentinel, got: %v", err)
	}

	assertNoJTDTResidue(t, db)
}

// TestJTDTDelete_FailsClosedWithoutIdentity pins the adapter Delete's fail-closed
// tenant gate: with no request identity on the context the soft-delete is refused
// before any SQL runs (draft-only delete is workspace + status scoped; a missing
// workspace can never delete). DB-free.
func TestJTDTDelete_FailsClosedWithoutIdentity(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	_, err := r.DeleteJobTemplateDocumentTemplate(context.Background(), &pb.DeleteJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{Id: "b-1"},
	})
	if err == nil {
		t.Fatal("delete must fail closed when no workspace identity is present")
	}
	if !strings.Contains(err.Error(), "workspace identity required") {
		t.Errorf("expected workspace-identity fail-closed, got: %v", err)
	}
}

func assertNoJTDTResidue(t *testing.T, db *sql.DB) {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM job_template_document_template WHERE id LIKE 'jtdttest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in bindings: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM document_template WHERE id LIKE 'jtdttest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in document_template: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM job_category WHERE id LIKE 'jtdttest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in job_category: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM price_schedule WHERE id LIKE 'jtdttest-%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in price_schedule: %d rows (err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM workspace WHERE id = $1`, jtdttestWS).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback residue in workspace: %d rows (err=%v)", n, err)
	}
}
