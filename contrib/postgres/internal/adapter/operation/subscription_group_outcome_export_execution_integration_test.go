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

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
	bindingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
	_ "github.com/lib/pq"
)

// The production composite query is deliberately exercised through an ambient
// PostgreSQL transaction.  The fixture uses temporary tables, so every row is
// both rollback-only and independent of the target database's resident data.
// This is important for this query: its graph is too broad for committed test
// rows, and the adapter must observe the same transaction snapshot as callers.
const exportExecutionDSNEnv = "TEST_DATABASE_URL"

const exportExecutionProfile = "RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1"

func openExportExecutionDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv(exportExecutionDSNEnv)
	if dsn == "" {
		t.Skip(exportExecutionDSNEnv + " not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL integration database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func execExportFixture(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, statements ...string) error {
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func createExportExecutionTempTables(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	// Keep these definitions intentionally minimal.  PostgreSQL resolves the
	// temporary relation before the resident public relation, while the real
	// adapter still executes the exact production SQL and joins.
	return execExportFixture(ctx, db,
		`CREATE TEMP TABLE workspace_user (id text, workspace_id text, user_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_group_workspace_user (workspace_user_id text, workspace_id text, subscription_group_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_group (id text, workspace_id text, name text, active boolean, price_schedule_id text, plan_id text) ON COMMIT DROP`,
		`CREATE TEMP TABLE price_schedule (id text, workspace_id text, name text) ON COMMIT DROP`,
		`CREATE TEMP TABLE plan (id text, workspace_id text, name text) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_group_member (subscription_group_id text, workspace_id text, subscription_id text, client_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription (id text, workspace_id text, client_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE client (id text, workspace_id text, name text, first_name text, last_name text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job (id text, workspace_id text, origin_type text, origin_id text, client_id text, job_template_id text, job_category_id text, output_product_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_template (id text, workspace_id text, name text, job_category_id text, output_product_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_category (id text, workspace_id text, code text, name text, sort_order integer, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_template_phase (id text, workspace_id text, job_template_id text, code text, name text, phase_order integer, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_phase (id text, workspace_id text, job_id text, template_phase_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE phase_outcome_summary (id text, workspace_id text, job_id text, job_phase_id text, scaled_label text, scaled_score double precision, date_created timestamptz, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_outcome_summary (id text, workspace_id text, job_id text, scaled_label text, scaled_score double precision, date_created timestamptz, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE job_task (id text, workspace_id text, job_phase_id text, assigned_to text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE task_outcome (id text, workspace_id text, job_task_id text, recorded_by text, reviewed_by text, numeric_value double precision, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_seat (subscription_id text, workspace_id text, product_plan_id text, staff_id text, status text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE product_plan (id text, product_id text) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_group_product_plan_staff (id text, workspace_id text, subscription_group_id text, product_plan_id text, product_plan_staff_id text, staff_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE product_plan_staff (id text, workspace_id text, staff_id text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE document_template (id text, workspace_id text, name text, template_type text, document_purpose text, storage_container text, storage_key text, status text, active boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE subscription_group_document_template (id text, workspace_id text, document_template_id text, render_profile text, job_category_id text, plan_id text, price_schedule_id text, version integer, version_status text, validity_start timestamptz, validity_end timestamptz, active boolean) ON COMMIT DROP`,
	)
}

func seedExportExecutionFixture(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	if err := createExportExecutionTempTables(ctx, db); err != nil {
		return err
	}
	return execExportFixture(ctx, db,
		`INSERT INTO workspace_user VALUES ('wu-main','ws-main','user-main',true), ('wu-other','ws-main','user-other',true)`,
		`INSERT INTO subscription_group_workspace_user VALUES ('wu-main','ws-main','sg-main',true)`,
		`INSERT INTO subscription_group VALUES
 ('sg-main','ws-main','Main Section',true,'sched-main','plan-main'),
 ('sg-historical','ws-main','Historical Section',false,NULL,NULL),
 ('sg-foreign','ws-foreign','Foreign Section',true,NULL,NULL)`,
		`INSERT INTO price_schedule VALUES ('sched-main','ws-main','AY 2026'), ('sched-foreign','ws-foreign','Foreign AY')`,
		`INSERT INTO plan VALUES ('plan-main','ws-main','Main Plan'), ('plan-foreign','ws-foreign','Foreign Plan')`,
		`INSERT INTO job_category VALUES
 ('cat-a','ws-main','academic','Academics',1,true),
 ('cat-b','ws-main','arts','Arts',2,true),
 ('cat-foreign','ws-foreign','academic','Foreign Academics',1,true)`,
		`INSERT INTO subscription_group_member VALUES
 ('sg-main','ws-main','sub-1','client-1',true),
 ('sg-main','ws-main','sub-2','client-2',true),
 ('sg-main','ws-foreign','sub-foreign','client-foreign',true),
 ('sg-historical','ws-main','sub-historical','client-historical',false)`,
		`INSERT INTO subscription VALUES
 ('sub-1','ws-main','client-1',true), ('sub-2','ws-main','client-2',true),
 ('sub-foreign','ws-foreign','client-foreign',true), ('sub-historical','ws-main','client-historical',false)`,
		`INSERT INTO client VALUES
 ('client-1','ws-main','Zed Learner','Zed','Learner',true),
 ('client-2','ws-main','Amy Learner','Amy','Learner',true),
 ('client-foreign','ws-foreign','Foreign Learner','Foreign','Learner',true),
 ('client-historical','ws-main','Historical Learner','Historical','Learner',true)`,
		`INSERT INTO job_template VALUES
 ('jt-alpha','ws-main',' zeta ','cat-a','product-a',true),
 ('jt-beta','ws-main','Alpha','cat-a','product-b',true),
 ('jt-arts','ws-main','Arts Job','cat-b','product-c',true),
 ('jt-root-phase','ws-main','Grade Root Phase','cat-a','product-p',true),
 ('jt-root-zero','ws-main','Grade Root','cat-a','product-z',true),
 ('jt-final-direct','ws-main','Grade Final Direct','cat-a','product-f',true),
 ('jt-foreign','ws-foreign','Foreign Job','cat-a','product-a',true)`,
		`INSERT INTO job_template_phase VALUES
		 ('jtp-alpha-1','ws-main','jt-alpha','P1','Term 1',1,true),
		 ('jtp-alpha-2','ws-main','jt-alpha','P2','Term 2',2,true),
		 ('jtp-beta-1','ws-main','jt-beta','P1','Term 1',1,true),
		 ('jtp-arts-1','ws-main','jt-arts','A1','Arts Term',1,true),
		 ('jtp-root-phase-1','ws-main','jt-root-phase','P1','Root Phase',1,true),
		 ('jtp-historical','ws-main','jt-missing','H1','Historical Term',1,true)`,
		`INSERT INTO job VALUES
 ('job-alpha-001','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-alpha','cat-a','product-a',true),
 ('job-alpha-999','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-alpha','cat-a','product-a',true),
 ('job-beta-001','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-beta','cat-a','product-b',true),
 ('job-alpha-002','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-2','client-2','jt-alpha','cat-a','product-a',true),
 ('job-arts-002','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-2','client-2','jt-arts','cat-b','product-c',true),
 ('job-root-phase-001','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-root-phase','cat-a','product-p',true),
 ('job-root-phase-002','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-2','client-2','jt-root-phase','cat-a','product-p',true),
 ('job-root-zero-001','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-root-zero','cat-a','product-z',true),
 ('job-final-direct-001','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-1','client-1','jt-final-direct','cat-a','product-f',true),
 ('job-foreign','ws-foreign','ORIGIN_TYPE_SUBSCRIPTION','sub-foreign','client-foreign','jt-foreign','cat-a','product-a',true),
 ('job-historical','ws-main','ORIGIN_TYPE_SUBSCRIPTION','sub-historical','client-historical','jt-missing','cat-missing','product-h',false)`,
		`INSERT INTO job_phase VALUES
 ('jp-alpha-001-p1','ws-main','job-alpha-001','jtp-alpha-1',true),
 ('jp-alpha-001-p2','ws-main','job-alpha-001','jtp-alpha-2',true),
 ('jp-alpha-999-p1','ws-main','job-alpha-999','jtp-alpha-1',true),
 ('jp-beta-001-p1','ws-main','job-beta-001','jtp-beta-1',true),
 ('jp-alpha-002-p1','ws-main','job-alpha-002','jtp-alpha-1',true),
 ('jp-arts-002-a1','ws-main','job-arts-002','jtp-arts-1',true),
 ('jp-root-phase-001-p1','ws-main','job-root-phase-001','jtp-root-phase-1',true),
 ('jp-root-phase-002-p1','ws-main','job-root-phase-002','jtp-root-phase-1',true),
 ('jp-historical-h1','ws-main','job-historical','jtp-historical',true),
 ('jp-foreign','ws-foreign','job-foreign','jtp-alpha-1',true)`,
		`INSERT INTO phase_outcome_summary VALUES
 ('pos-chosen','ws-main','job-alpha-001','jp-alpha-001-p1','chosen',0.5,'2026-01-02T00:00:00Z',true),
 ('pos-old','ws-main','job-alpha-001','jp-alpha-001-p1','old',0.1,'2026-01-01T00:00:00Z',true),
 ('pos-duplicate','ws-main','job-alpha-999','jp-alpha-999-p1','wrong-duplicate',0.9,'2026-01-03T00:00:00Z',true),
 ('pos-client-2','ws-main','job-alpha-002','jp-alpha-002-p1','client-two',0.7,'2026-01-02T00:00:00Z',true),
 ('pos-foreign','ws-foreign','job-foreign','jp-foreign','foreign',1.0,'2026-01-05T00:00:00Z',true)`,
		`INSERT INTO job_outcome_summary VALUES
		 ('jos-alpha','ws-main','job-alpha-001','Final A',0.8,'2026-01-04T00:00:00Z',true),
		 ('jos-duplicate','ws-main','job-alpha-999','Wrong Final',0.2,'2026-01-05T00:00:00Z',true),
		 ('jos-final-direct','ws-main','job-final-direct-001','Direct Final',0.9,'2026-01-06T00:00:00Z',true)`,
		`INSERT INTO job_task VALUES
 ('task-alpha-001','ws-main','jp-alpha-001-p1','staff-1',true),
 ('task-alpha-002','ws-main','jp-alpha-002-p1',NULL,true),
 ('task-foreign','ws-foreign','jp-foreign','staff-1',true)`,
		`INSERT INTO task_outcome VALUES
 ('outcome-zero','ws-main','task-alpha-001','staff-1',NULL,0,true),
 ('outcome-positive','ws-main','task-alpha-001','staff-1',NULL,1,true),
 ('outcome-foreign','ws-foreign','task-foreign','staff-1',NULL,1,true)`,
		`INSERT INTO document_template VALUES
 ('dt-exact','ws-main','Exact','docx','subscription_group_outcome_summary','templates','templates/exact.docx','active',true),
 ('dt-broad','ws-main','Broad','docx','subscription_group_outcome_summary','templates','templates/broad.docx','active',true),
 ('dt-dead','ws-main','Dead','docx','subscription_group_outcome_summary','templates','templates/dead.docx','active',false),
 ('dt-foreign','ws-foreign','Foreign','docx','subscription_group_outcome_summary','foreign','templates/foreign.docx','active',true)`,
		`INSERT INTO subscription_group_document_template VALUES
 ('bind-exact','ws-main','dt-exact',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-a','plan-main','sched-main',3,'VERSION_STATUS_PUBLISHED', '2026-01-01T00:00:00Z',NULL,true),
 ('bind-broad','ws-main','dt-broad',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-a',NULL,NULL,2,'VERSION_STATUS_PUBLISHED', '2026-01-01T00:00:00Z',NULL,true),
 ('bind-dead-a','ws-main','dt-dead',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-a','plan-main','sched-main',7,'VERSION_STATUS_PUBLISHED', '2026-01-01T00:00:00Z',NULL,true),
 ('bind-dead','ws-main','dt-dead',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-b','plan-main','sched-main',4,'VERSION_STATUS_PUBLISHED', '2026-01-01T00:00:00Z',NULL,true),
 ('bind-foreign','ws-main','dt-foreign',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-a','plan-main','sched-main',5,'VERSION_STATUS_PUBLISHED', '2026-01-01T00:00:00Z',NULL,true)`,
	)
}

func exportIdentityContext(workspaceID, userID string, principalType int32, principalID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID: workspaceID, UserID: userID, PrincipalType: principalType, PrincipalID: principalID,
	})
}

func exportCategoryByID(resp *exportpb.GetSubscriptionGroupOutcomeExportResponse, id string) *exportpb.JobCategoryOption {
	for _, category := range resp.GetJobCategories() {
		if category.GetJobCategoryId() == id {
			return category
		}
	}
	return nil
}

func exportColumnIDs(resp *exportpb.GetSubscriptionGroupOutcomeExportResponse) []string {
	ids := make([]string, 0, len(resp.GetJobTemplateColumns()))
	for _, column := range resp.GetJobTemplateColumns() {
		ids = append(ids, column.GetJobTemplateId())
	}
	return ids
}

func exportRowByID(resp *exportpb.GetSubscriptionGroupOutcomeExportResponse, id string) *exportpb.SubscriptionGroupOutcomeClientRow {
	for _, row := range resp.GetClientRows() {
		if row.GetClientId() == id {
			return row
		}
	}
	return nil
}

func exportCellByTemplate(row *exportpb.SubscriptionGroupOutcomeClientRow, id string) *exportpb.SubscriptionGroupOutcomeCell {
	for _, cell := range row.GetCells() {
		if cell.GetJobTemplateId() == id {
			return cell
		}
	}
	return nil
}

func TestIntegration_SubscriptionGroupOutcomeExport_RollbackMatrixScopeAndEvidence(t *testing.T) {
	db := openExportExecutionDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	query := NewPostgresSubscriptionGroupOutcomeExportQuery(db)
	rollback := errorsNewExportRollback()

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := postgresCore.ExecutorFromContext(txCtx, db)
		if err := seedExportExecutionFixture(txCtx, exec); err != nil {
			return fmt.Errorf("seed temporary export fixture: %w", err)
		}

		wideCtx := exportIdentityContext("ws-main", "user-main", 1, "")
		wideCtx = transplantExportTransaction(wideCtx, txCtx)
		options, err := query.GetSubscriptionGroupOutcomeExportScoped(wideCtx, &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-main"}, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil {
			return fmt.Errorf("options query: %w", err)
		}
		if options.GetContext() == nil || len(options.GetJobCategories()) != 2 {
			return fmt.Errorf("options must return context and two local categories: %#v", options)
		}
		catA := exportCategoryByID(options, "cat-a")
		catB := exportCategoryByID(options, "cat-b")
		if catA == nil || catB == nil || !catA.GetFinalOutcomeAvailable() || catB.GetFinalOutcomeAvailable() {
			return fmt.Errorf("final availability must be category-local: catA=%v catB=%v", catA.GetFinalOutcomeAvailable(), catB.GetFinalOutcomeAvailable())
		}
		if len(catA.GetJobTemplatePhases()) != 2 || catA.GetJobTemplatePhases()[0].GetCode() != "P1" || catA.GetJobTemplatePhases()[1].GetCode() != "P2" {
			return fmt.Errorf("category A phases not complete/ordered: %#v", catA.GetJobTemplatePhases())
		}
		if len(catB.GetJobTemplatePhases()) != 1 || catB.GetJobTemplatePhases()[0].GetCode() != "A1" {
			return fmt.Errorf("category B phase leaked or missing: %#v", catB.GetJobTemplatePhases())
		}

		categoryID := "cat-a"
		phaseReq := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
			SubscriptionGroupId: "sg-main", JobCategoryId: &categoryID,
			OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{JobTemplatePhaseCode: "P1"},
		}
		matrix, err := query.GetSubscriptionGroupOutcomeExportScoped(wideCtx, phaseReq, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil {
			return fmt.Errorf("phase matrix query: %w", err)
		}
		if got := strings.Join(exportColumnIDs(matrix), ","); got != "jt-beta,jt-alpha" {
			return fmt.Errorf("columns must use normalized display-name order, got %q", got)
		}
		if len(matrix.GetClientRows()) != 2 || len(matrix.GetClientRows()[0].GetCells()) != 3 || len(matrix.GetClientRows()[1].GetCells()) != 3 {
			return fmt.Errorf("matrix must be a complete two-by-two rectangle")
		}
		for _, required := range []string{"jt-beta", "jt-alpha", "jt-root-phase"} {
			if !strings.Contains(strings.Join(exportColumnIDs(matrix), ","), required) {
				return fmt.Errorf("required phase-capable template %q missing in matrix columns: %q", required, strings.Join(exportColumnIDs(matrix), ","))
			}
		}
		if strings.Contains(strings.Join(exportColumnIDs(matrix), ","), "jt-root-zero") {
			return fmt.Errorf("zero-phase root template leaked into phase matrix: %q", strings.Join(exportColumnIDs(matrix), ","))
		}
		row1 := exportRowByID(matrix, "client-1")
		if row1 == nil {
			return fmt.Errorf("client-1 row missing")
		}
		chosen := exportCellByTemplate(row1, "jt-alpha")
		if chosen == nil || chosen.GetScaledLabel() != "chosen" || chosen.GetScaledScore() != 0.5 {
			return fmt.Errorf("duplicate job/summary selection was not deterministic: %#v", chosen)
		}
		if chosen.GetEnrollmentEvidence() == nil || !chosen.GetEnrollmentEvidence().GetHasMarks() || !chosen.GetEnrollmentEvidence().GetHasPositiveMark() {
			return fmt.Errorf("phase evidence must retain zero and positive marks")
		}

		finalReq := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
			SubscriptionGroupId: "sg-main", JobCategoryId: &categoryID,
			OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome{FinalOutcome: true},
		}
		finalMatrix, err := query.GetSubscriptionGroupOutcomeExportScoped(wideCtx, finalReq, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil {
			return fmt.Errorf("final matrix query: %w", err)
		}
		finalCell := exportCellByTemplate(exportRowByID(finalMatrix, "client-1"), "jt-alpha")
		if finalCell == nil || finalCell.GetScaledLabel() != "Final A" {
			return fmt.Errorf("final summary selection mismatch: %#v", finalCell)
		}
		if rootPhaseCell := exportCellByTemplate(exportRowByID(finalMatrix, "client-1"), "jt-root-phase"); rootPhaseCell != nil && rootPhaseCell.GetScaledLabel() != "" {
			return fmt.Errorf("substantive phase root must remain blank on final when no final summary: %#v", rootPhaseCell)
		}
		if finalDirectCell := exportCellByTemplate(exportRowByID(finalMatrix, "client-1"), "jt-final-direct"); finalDirectCell == nil || finalDirectCell.GetScaledLabel() != "Direct Final" {
			return fmt.Errorf("direct final-only template must remain on final outcome: %#v", finalDirectCell)
		}

		// SGWU is the narrow non-staff servicing gate.  A user without that
		// edge receives the same empty context shape as a foreign group.
		narrowCtx := transplantExportTransaction(exportIdentityContext("ws-main", "user-main", 1, ""), txCtx)
		narrow, err := query.GetSubscriptionGroupOutcomeExportScoped(narrowCtx, &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-main"}, ports.SubscriptionGroupOutcomeExportScope{})
		if err != nil || narrow.GetContext() == nil {
			return fmt.Errorf("narrow SGWU scope should resolve group: err=%v context=%v", err, narrow.GetContext())
		}
		noGrantCtx := transplantExportTransaction(exportIdentityContext("ws-main", "user-other", 1, ""), txCtx)
		noGrant, err := query.GetSubscriptionGroupOutcomeExportScoped(noGrantCtx, &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-main"}, ports.SubscriptionGroupOutcomeExportScope{})
		if err != nil || noGrant.GetContext() != nil {
			return fmt.Errorf("missing SGWU scope must be empty: err=%v context=%v", err, noGrant.GetContext())
		}

		// Staff reachability is independent of SGWU: the assigned task reaches
		// job-alpha-001, while job-beta-001 is intentionally not assigned.
		staffCtx := transplantExportTransaction(exportIdentityContext("ws-main", "user-main", principalscope.PrincipalTypeStaff, "staff-1"), txCtx)
		staffMatrix, err := query.GetSubscriptionGroupOutcomeExportScoped(staffCtx, phaseReq, ports.SubscriptionGroupOutcomeExportScope{})
		if err != nil {
			return fmt.Errorf("staff matrix query: %w", err)
		}
		if got := strings.Join(exportColumnIDs(staffMatrix), ","); got != "jt-alpha" {
			return fmt.Errorf("staff scope must retain only assigned job, got columns %q", got)
		}

		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}
}

func TestIntegration_SubscriptionGroupOutcomeExport_RollbackHistoricalSentinelAndResolver(t *testing.T) {
	db := openExportExecutionDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	query := NewPostgresSubscriptionGroupOutcomeExportQuery(db)
	rollback := errorsNewExportRollback()

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := postgresCore.ExecutorFromContext(txCtx, db)
		if err := seedExportExecutionFixture(txCtx, exec); err != nil {
			return fmt.Errorf("seed temporary export fixture: %w", err)
		}
		ctx := transplantExportTransaction(exportIdentityContext("ws-main", "user-main", 1, ""), txCtx)
		historical, err := query.GetSubscriptionGroupOutcomeExportScoped(ctx, &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-historical"}, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil {
			return fmt.Errorf("historical options query: %w", err)
		}
		uncategorized := exportCategoryByID(historical, "uncategorized")
		if historical.GetContext() == nil || uncategorized == nil || len(uncategorized.GetJobTemplatePhases()) != 1 || uncategorized.GetJobTemplatePhases()[0].GetCode() != "H1" {
			return fmt.Errorf("historical missing-template branch must preserve uncategorized sentinel phase: %#v", historical)
		}

		categoryID := "cat-a"
		resolveReq := &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
			SubscriptionGroupId: "sg-main", JobCategoryId: categoryID,
			RenderProfile:  bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			ExpectedPlanId: stringPtr("plan-main"), ExpectedPriceScheduleId: stringPtr("sched-main"),
		}
		resolved, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, resolveReq, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil || !resolved.GetFound() || resolved.GetDocument().GetStorageKey() != "templates/exact.docx" {
			return fmt.Errorf("exact resolver precedence mismatch: err=%v response=%v", err, resolved)
		}

		// The foreign artifact is joined with workspace equality and cannot
		// shadow the valid exact candidate. A plan/schedule mismatch is the
		// same non-enumerating miss and carries no locator.
		badAxes := *resolveReq
		badPlan := "plan-other"
		badAxes.ExpectedPlanId = &badPlan
		miss, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &badAxes, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil || miss.GetFound() || miss.GetDocument() != nil {
			return fmt.Errorf("axis mismatch must be locator-free miss: err=%v response=%v", err, miss)
		}
		foreignGroup := *resolveReq
		foreignGroup.SubscriptionGroupId = "sg-foreign"
		foreignMiss, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &foreignGroup, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil || foreignMiss.GetFound() || foreignMiss.GetDocument() != nil {
			return fmt.Errorf("foreign group must be locator-free miss: err=%v response=%v", err, foreignMiss)
		}
		noBinding := *resolveReq
		noBinding.JobCategoryId = "cat-b"
		noBindingMiss, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &noBinding, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil || noBindingMiss.GetFound() || noBindingMiss.GetDocument() != nil {
			return fmt.Errorf("no binding must be locator-free miss: err=%v response=%v", err, noBindingMiss)
		}

		// Remove the exact candidate, then prove inactive and foreign
		// more-specific artifacts cannot shadow a valid broad candidate.
		fallbackReq := *resolveReq
		fallbackReq.JobCategoryId = "cat-a"
		if _, err := exec.ExecContext(txCtx, `DELETE FROM subscription_group_document_template WHERE id = 'bind-exact'`); err != nil {
			return fmt.Errorf("remove exact resolver row: %w", err)
		}
		fallback, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &fallbackReq, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err != nil || !fallback.GetFound() || fallback.GetDocument().GetStorageKey() != "templates/broad.docx" {
			return fmt.Errorf("valid broad fallback must survive invalid specific artifact: err=%v response=%v", err, fallback)
		}

		// Equal-rank candidates are an error, not arbitrary version selection.
		if _, err := exec.ExecContext(txCtx, `INSERT INTO document_template VALUES ('dt-broad-2','ws-main','Broad 2','docx','subscription_group_outcome_summary','templates','templates/broad-2.docx','active',true)`); err != nil {
			return fmt.Errorf("insert ambiguity artifact: %w", err)
		}
		if _, err := exec.ExecContext(txCtx, `INSERT INTO subscription_group_document_template VALUES ('bind-broad-2','ws-main','dt-broad-2',`+fmt.Sprintf("'%s'", exportExecutionProfile)+`,'cat-a',NULL,NULL,8,'VERSION_STATUS_PUBLISHED','2026-01-01T00:00:00Z',NULL,true)`); err != nil {
			return fmt.Errorf("insert ambiguity binding: %w", err)
		}
		_, err = query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &fallbackReq, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
		if err == nil || !strings.Contains(err.Error(), "ambiguous candidates") {
			return fmt.Errorf("equal-rank resolver candidates must fail closed, got %v", err)
		}

		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}
}

func errorsNewExportRollback() error {
	return fmt.Errorf("subscription_group_outcome_export: intentional rollback")
}

// Transaction identity is deliberately copied without exposing the internal
// transaction key to this test package. The core transaction manager preserves
// the transaction context in the callback; this helper keeps the request
// identity and transaction on the same context by using the callback context's
// value carrier.
func transplantExportTransaction(identityCtx, txCtx context.Context) context.Context {
	id, ok := identity.FromContext(identityCtx)
	if !ok || id == nil {
		return txCtx
	}
	return identity.WithRequestIdentity(txCtx, id)
}
