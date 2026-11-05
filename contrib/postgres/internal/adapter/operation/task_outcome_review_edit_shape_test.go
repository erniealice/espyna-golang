//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// AC-12 (plan 20261105-reviewer-edit-grades): with NO review-edit decision on the
// context (denied, the default), the task_outcome write/read path must emit the
// exact SQL it emitted at base commit 2cfe57b5. The golden strings below were
// captured from that commit with the same harness (a recording database/sql
// driver — no live database; the existing recordingDriver in the conditional
// audit test needs TEST_DATABASE_URL, so a self-contained stub driver is used).

// ---- recording stub driver (no database) ------------------------------------

type shapeRecorder struct {
	mu    sync.Mutex
	stmts []string
	args  [][]driver.NamedValue
}

var shapeRec = &shapeRecorder{}

func (r *shapeRecorder) reset() {
	r.mu.Lock()
	r.stmts, r.args = nil, nil
	r.mu.Unlock()
}

func (r *shapeRecorder) note(q string, a []driver.NamedValue) {
	r.mu.Lock()
	r.stmts = append(r.stmts, strings.Join(strings.Fields(q), " "))
	r.args = append(r.args, a)
	r.mu.Unlock()
}

func (r *shapeRecorder) snapshot() ([]string, [][]driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.stmts...), append([][]driver.NamedValue(nil), r.args...)
}

type shapeDriver struct{}

func (shapeDriver) Open(string) (driver.Conn, error) { return &shapeConn{}, nil }

type shapeConn struct{}

func (*shapeConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("prepare unsupported") }
func (*shapeConn) Close() error                        { return nil }
func (*shapeConn) Begin() (driver.Tx, error)           { return shapeTx{}, nil }
func (*shapeConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return shapeTx{}, nil
}
func (*shapeConn) QueryContext(_ context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	shapeRec.note(q, a)
	return shapeRows{}, nil
}
func (*shapeConn) ExecContext(_ context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	shapeRec.note(q, a)
	return driver.RowsAffected(0), nil
}

type shapeTx struct{}

func (shapeTx) Commit() error   { return nil }
func (shapeTx) Rollback() error { return nil }

type shapeRows struct{}

func (shapeRows) Columns() []string         { return []string{"x"} }
func (shapeRows) Close() error              { return nil }
func (shapeRows) Next([]driver.Value) error { return io.EOF }

var registerShapeDriver sync.Once

func openShapeDB(t *testing.T) *sql.DB {
	t.Helper()
	registerShapeDriver.Do(func() { sql.Register("reviewedit-shape-stub", shapeDriver{}) })
	db, err := sql.Open("reviewedit-shape-stub", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ---- fake dbOps (generic Read/Update) ---------------------------------------

type shapeDBOps struct {
	interfaces.DatabaseOperation // nil: any unexpected method panics, loudly
	exec                         *sql.Tx
	readRow                      map[string]any
	updateKeys                   []string
	readCalls                    int
}

func (f *shapeDBOps) GetExecutor(context.Context) sqlexec.DBExecutor { return f.exec }

func (f *shapeDBOps) Read(_ context.Context, _ string, id string) (map[string]any, error) {
	f.readCalls++
	row := map[string]any{}
	for k, v := range f.readRow {
		row[k] = v
	}
	row["id"] = id
	return row, nil
}

func (f *shapeDBOps) Update(_ context.Context, _ string, id string, data map[string]any) (map[string]any, error) {
	f.updateKeys = f.updateKeys[:0]
	for k := range data {
		f.updateKeys = append(f.updateKeys, k)
	}
	sort.Strings(f.updateKeys)
	return map[string]any{"id": id}, nil
}

// ---- contexts ----------------------------------------------------------------

const shapeWS = "ws-shape"

func shapeStaffCtx(workspaceWide bool) context.Context {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "user-shape", WorkspaceID: shapeWS,
		PrincipalType: principalscope.PrincipalTypeStaff, PrincipalID: "staff-shape",
	})
	if workspaceWide {
		ctx = identity.WithWorkspaceRowScope(ctx)
	}
	return ctx
}

func shapeOperatorCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: "op-shape", WorkspaceID: shapeWS, PrincipalType: 1, PrincipalID: "op-1",
	})
}

// ---- captures -----------------------------------------------------------------

// captureConditionalSQL runs UpdateTaskOutcomeIfUnchanged against the stub and
// returns the UPDATE statement text plus its bind count. The stub returns no row,
// so the call ends in ErrTaskOutcomeConflict — expected here.
func captureConditionalSQL(t *testing.T, ctx context.Context) (string, int) {
	t.Helper()
	db := openShapeDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	repo := &PostgresTaskOutcomeRepository{dbOps: &shapeDBOps{exec: tx}, tableName: "task_outcome"}
	v, note := 7.5, "n"
	shapeRec.reset()
	_, _ = repo.UpdateTaskOutcomeIfUnchanged(ctx,
		&pb.TaskOutcome{Id: "to-1", NumericValue: &v, DeterminationNote: &note},
		&pb.TaskOutcome{Id: "to-1"})
	stmts, args := shapeRec.snapshot()
	if len(stmts) != 1 {
		t.Fatalf("expected exactly 1 statement, got %d: %q", len(stmts), stmts)
	}
	return stmts[0], len(args[0])
}

// captureUpdateKeys runs the plain UpdateTaskOutcome with a request that carries
// recorded_by / reviewed_by / reviewed_date and returns the sorted keys handed to
// the generic Update plus the number of SQL statements the repo itself issued.
func captureUpdateKeys(t *testing.T, ctx context.Context) (string, int) {
	t.Helper()
	db := openShapeDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	ops := &shapeDBOps{exec: tx}
	repo := &PostgresTaskOutcomeRepository{dbOps: ops, tableName: "task_outcome"}
	v, by, when := 7.5, "someone", int64(1700000000000)
	shapeRec.reset()
	if _, err := repo.UpdateTaskOutcome(ctx, &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{
		Id: "to-1", NumericValue: &v, RecordedBy: "rec-1", ReviewedBy: &by, ReviewedDate: &when,
	}}); err != nil {
		t.Fatalf("UpdateTaskOutcome: %v", err)
	}
	stmts, _ := shapeRec.snapshot()
	return strings.Join(ops.updateKeys, ","), len(stmts)
}

// captureReadStatements runs ReadTaskOutcome for an outcome recorded by someone
// else and returns (found, SQL statements issued by the repo itself).
func captureReadStatements(t *testing.T, ctx context.Context) (bool, int) {
	t.Helper()
	db := openShapeDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	ops := &shapeDBOps{exec: tx, readRow: map[string]any{"recordedBy": "staff-other", "jobTaskId": "jt-1"}}
	repo := &PostgresTaskOutcomeRepository{dbOps: ops, tableName: "task_outcome"}
	shapeRec.reset()
	resp, err := repo.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "to-1"}})
	stmts, _ := shapeRec.snapshot()
	return err == nil && resp != nil && len(resp.GetData()) == 1, len(stmts)
}

// ---- base goldens (captured at 2cfe57b5) --------------------------------------

// captureMatrixSQL runs the matrix cell query (loadRowsFrom) against the stub and
// returns the cells SELECT plus its bind values.
func captureMatrixSQL(t *testing.T, ctx context.Context, scope matrixpb.OutcomeMatrixScope) (string, string) {
	t.Helper()
	db := openShapeDB(t)
	shapeRec.reset()
	a := &PostgresOutcomeMatrixQuery{db: db}
	_, err := a.loadRowsFrom(ctx, db, &matrixpb.GetOutcomeMatrixRequest{JobTemplateId: "tpl-1", Scope: scope}, shapeWS)
	if err != nil {
		t.Fatalf("loadRowsFrom: %v", err)
	}
	stmts, args := shapeRec.snapshot()
	if len(stmts) != 1 {
		t.Fatalf("expected exactly 1 statement, got %d", len(stmts))
	}
	vals := make([]string, 0, len(args[0]))
	for _, a := range args[0] {
		vals = append(vals, fmt.Sprintf("%v", a.Value))
	}
	return stmts[0], strings.Join(vals, "|")
}

// Captured from base commit 2cfe57b5 (whitespace-normalised).
const baseMatrixStaffSQL = `SELECT DISTINCT ON (j.client_id, jt.id, ttc.id) j.client_id, jt.id AS job_task_id, jp.id AS job_phase_id, j.id AS job_id, ttc.job_template_task_id AS job_template_task_id, ttc.outcome_criteria_id AS criteria_id, t.id AS outcome_id, t.numeric_value, t.text_value, t.categorical_value, t.pass_fail_value, t.determination_note, COALESCE(t.recorded_by, '') AS recorded_by, COALESCE(jt.assigned_to, '') AS assigned_to, EXISTS ( SELECT 1 FROM subscription_group_member m JOIN subscription_group sg ON sg.id = m.subscription_group_id AND sg.status = 'current' JOIN subscription_group_product_plan c ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = $2 JOIN product_plan pp ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id JOIN subscription_group_product_plan_staff e ON e.subscription_group_product_plan_id = c.id AND e.active AND e.staff_id = $5 AND e.workspace_id = $2 AND (e.job_template_phase_id IS NULL OR EXISTS ( SELECT 1 FROM job_template_phase ep, job_template_phase jpp WHERE ep.id = e.job_template_phase_id AND jpp.id = jp.template_phase_id AND ep.phase_order = jpp.phase_order )) WHERE m.client_id = j.client_id AND m.active AND m.workspace_id = $2 ) AS class_edge_editable FROM job j JOIN job_phase jp ON jp.job_id = j.id AND jp.active JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active JOIN template_task_criteria ttc ON ttc.job_template_task_id = jt.template_task_id AND ttc.active LEFT JOIN task_outcome t ON t.job_task_id = jt.id AND t.criteria_version_id = ttc.outcome_criteria_id AND t.active WHERE j.job_template_id = $1 AND j.workspace_id = $2 AND j.active AND j.id IN (SELECT jp.job_id FROM job_phase jp JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.workspace_id = $4 JOIN job jw ON jw.id = jp.job_id AND jw.workspace_id = $4 WHERE jp.workspace_id = $4 AND jt.assigned_to = $3 UNION SELECT jp2.job_id FROM job_phase jp2 JOIN job_task jt2 ON jt2.job_phase_id = jp2.id AND jt2.workspace_id = $4 JOIN task_outcome t ON t.job_task_id = jt2.id AND t.workspace_id = $4 JOIN job jw2 ON jw2.id = jp2.job_id AND jw2.workspace_id = $4 WHERE jp2.workspace_id = $4 AND (t.recorded_by = $3 OR t.reviewed_by = $3) UNION SELECT jw3.id FROM job jw3 JOIN subscription_seat ss ON ss.subscription_id = jw3.origin_id JOIN job_template tpl ON tpl.id = jw3.job_template_id AND tpl.workspace_id = $4 JOIN product_plan pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id WHERE ss.staff_id = $3 AND ss.status = 'active' AND ss.active = true AND jw3.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION' AND jw3.workspace_id = $4 AND ss.workspace_id = $4 UNION SELECT jce.id FROM subscription_group_product_plan_staff e JOIN subscription_group_member m ON m.subscription_group_id = e.subscription_group_id AND m.workspace_id = $4 AND m.active JOIN product_plan pp ON pp.id = e.product_plan_id JOIN job jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id AND jce.workspace_id = $4 LEFT JOIN product_plan_staff pps ON pps.id = e.product_plan_staff_id AND pps.workspace_id = $4 WHERE COALESCE(pps.staff_id, e.staff_id) = $3 AND (e.product_plan_staff_id IS NULL OR pps.active) AND e.active AND e.workspace_id = $4 UNION SELECT jr.id FROM product_plan_staff r JOIN subscription_group_product_plan rc ON rc.product_plan_id = r.product_plan_id AND rc.active AND rc.workspace_id = $4 JOIN subscription_group_member rm ON rm.subscription_group_id = rc.subscription_group_id AND rm.active AND rm.workspace_id = $4 JOIN product_plan rpp ON rpp.id = r.product_plan_id JOIN job jr ON jr.origin_id = rm.subscription_id AND jr.output_product_id = rpp.product_id AND jr.workspace_id = $4 WHERE r.staff_id = $3 AND r.role = 'reviewer' AND r.active AND r.workspace_id = $4) ORDER BY j.client_id, jt.id, ttc.id, t.recorded_date DESC NULLS LAST, t.id DESC`

const baseMatrixWideSQL = `SELECT DISTINCT ON (j.client_id, jt.id, ttc.id) j.client_id, jt.id AS job_task_id, jp.id AS job_phase_id, j.id AS job_id, ttc.job_template_task_id AS job_template_task_id, ttc.outcome_criteria_id AS criteria_id, t.id AS outcome_id, t.numeric_value, t.text_value, t.categorical_value, t.pass_fail_value, t.determination_note, COALESCE(t.recorded_by, '') AS recorded_by, COALESCE(jt.assigned_to, '') AS assigned_to, EXISTS ( SELECT 1 FROM subscription_group_member m JOIN subscription_group sg ON sg.id = m.subscription_group_id AND sg.status = 'current' JOIN subscription_group_product_plan c ON c.subscription_group_id = m.subscription_group_id AND c.active AND c.workspace_id = $2 JOIN product_plan pp ON pp.id = c.product_plan_id AND pp.product_id = j.output_product_id JOIN subscription_group_product_plan_staff e ON e.subscription_group_product_plan_id = c.id AND e.active AND e.staff_id = $3 AND e.workspace_id = $2 AND (e.job_template_phase_id IS NULL OR EXISTS ( SELECT 1 FROM job_template_phase ep, job_template_phase jpp WHERE ep.id = e.job_template_phase_id AND jpp.id = jp.template_phase_id AND ep.phase_order = jpp.phase_order )) WHERE m.client_id = j.client_id AND m.active AND m.workspace_id = $2 ) AS class_edge_editable FROM job j JOIN job_phase jp ON jp.job_id = j.id AND jp.active JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active JOIN template_task_criteria ttc ON ttc.job_template_task_id = jt.template_task_id AND ttc.active LEFT JOIN task_outcome t ON t.job_task_id = jt.id AND t.criteria_version_id = ttc.outcome_criteria_id AND t.active WHERE j.job_template_id = $1 AND j.workspace_id = $2 AND j.active ORDER BY j.client_id, jt.id, ttc.id, t.recorded_date DESC NULLS LAST, t.id DESC`

const baseMatrixOperatorSQL = `SELECT DISTINCT ON (j.client_id, jt.id, ttc.id) j.client_id, jt.id AS job_task_id, jp.id AS job_phase_id, j.id AS job_id, ttc.job_template_task_id AS job_template_task_id, ttc.outcome_criteria_id AS criteria_id, t.id AS outcome_id, t.numeric_value, t.text_value, t.categorical_value, t.pass_fail_value, t.determination_note, COALESCE(t.recorded_by, '') AS recorded_by, COALESCE(jt.assigned_to, '') AS assigned_to, false AS class_edge_editable FROM job j JOIN job_phase jp ON jp.job_id = j.id AND jp.active JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active JOIN template_task_criteria ttc ON ttc.job_template_task_id = jt.template_task_id AND ttc.active LEFT JOIN task_outcome t ON t.job_task_id = jt.id AND t.criteria_version_id = ttc.outcome_criteria_id AND t.active WHERE j.job_template_id = $1 AND j.workspace_id = $2 AND j.active ORDER BY j.client_id, jt.id, ttc.id, t.recorded_date DESC NULLS LAST, t.id DESC`

const baseCondStaffSQL = `UPDATE task_outcome SET numeric_value = $1, determination_note = $2, date_modified = $3 WHERE id = $4 AND workspace_id = $5 AND active = true AND numeric_value IS NOT DISTINCT FROM $6::double precision AND determination_note IS NOT DISTINCT FROM $7::text AND text_value IS NOT DISTINCT FROM $9::text AND categorical_value IS NOT DISTINCT FROM $10::text AND pass_fail_value IS NOT DISTINCT FROM $11::boolean AND floor(extract(epoch FROM date_modified) * 1000)::bigint IS NOT DISTINCT FROM $8::bigint AND (recorded_by = $12 OR reviewed_by = $12) RETURNING id, COALESCE(job_task_id, '') AS job_task_id, COALESCE(criteria_version_id, '') AS criteria_version_id, COALESCE(criteria_type, '') AS criteria_type, COALESCE(is_ad_hoc, false) AS is_ad_hoc, numeric_value, text_value, categorical_value, pass_fail_value, COALESCE(determination, '') AS determination, COALESCE(determination_source, '') AS determination_source, determination_note, auto_proposed_determination, COALESCE(recorded_by, '') AS recorded_by, recorded_date, reviewed_by, reviewed_date, COALESCE(attachment_ids, '') AS attachment_ids, revision_of_id, revision_number, active, date_created, date_modified`

const baseCondWideSQL = `UPDATE task_outcome SET numeric_value = $1, determination_note = $2, date_modified = $3 WHERE id = $4 AND workspace_id = $5 AND active = true AND numeric_value IS NOT DISTINCT FROM $6::double precision AND determination_note IS NOT DISTINCT FROM $7::text AND text_value IS NOT DISTINCT FROM $9::text AND categorical_value IS NOT DISTINCT FROM $10::text AND pass_fail_value IS NOT DISTINCT FROM $11::boolean AND floor(extract(epoch FROM date_modified) * 1000)::bigint IS NOT DISTINCT FROM $8::bigint RETURNING id, COALESCE(job_task_id, '') AS job_task_id, COALESCE(criteria_version_id, '') AS criteria_version_id, COALESCE(criteria_type, '') AS criteria_type, COALESCE(is_ad_hoc, false) AS is_ad_hoc, numeric_value, text_value, categorical_value, pass_fail_value, COALESCE(determination, '') AS determination, COALESCE(determination_source, '') AS determination_source, determination_note, auto_proposed_determination, COALESCE(recorded_by, '') AS recorded_by, recorded_date, reviewed_by, reviewed_date, COALESCE(attachment_ids, '') AS attachment_ids, revision_of_id, revision_number, active, date_created, date_modified`

const baseCondOperatorSQL = `UPDATE task_outcome SET numeric_value = $1, determination_note = $2, date_modified = $3 WHERE id = $4 AND workspace_id = $5 AND active = true AND numeric_value IS NOT DISTINCT FROM $6::double precision AND determination_note IS NOT DISTINCT FROM $7::text AND text_value IS NOT DISTINCT FROM $9::text AND categorical_value IS NOT DISTINCT FROM $10::text AND pass_fail_value IS NOT DISTINCT FROM $11::boolean AND floor(extract(epoch FROM date_modified) * 1000)::bigint IS NOT DISTINCT FROM $8::bigint RETURNING id, COALESCE(job_task_id, '') AS job_task_id, COALESCE(criteria_version_id, '') AS criteria_version_id, COALESCE(criteria_type, '') AS criteria_type, COALESCE(is_ad_hoc, false) AS is_ad_hoc, numeric_value, text_value, categorical_value, pass_fail_value, COALESCE(determination, '') AS determination, COALESCE(determination_source, '') AS determination_source, determination_note, auto_proposed_determination, COALESCE(recorded_by, '') AS recorded_by, recorded_date, reviewed_by, reviewed_date, COALESCE(attachment_ids, '') AS attachment_ids, revision_of_id, revision_number, active, date_created, date_modified`

const (
	baseMatrixStaffArgs    = "tpl-1|ws-shape|staff-shape|ws-shape|staff-shape"
	baseMatrixWideArgs     = "tpl-1|ws-shape|staff-shape"
	baseMatrixOperatorArgs = "tpl-1|ws-shape"
	baseCondStaffBinds     = 12
	baseCondWideBinds      = 11
	baseCondOperatorBinds  = 11
	baseUpdateKeys         = "id,numericValue,recordedBy,reviewedBy,reviewedDate"
)

// ---- AC-12 tests ----------------------------------------------------------------

func TestReviewEdit_DeniedSQLUnchanged_ConditionalUpdate(t *testing.T) {
	for _, c := range []struct {
		name  string
		ctx   context.Context
		sql   string
		binds int
	}{
		{"staff_row_scoped", shapeStaffCtx(false), baseCondStaffSQL, baseCondStaffBinds},
		{"staff_workspace_wide", shapeStaffCtx(true), baseCondWideSQL, baseCondWideBinds},
		{"operator", shapeOperatorCtx(), baseCondOperatorSQL, baseCondOperatorBinds},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, binds := captureConditionalSQL(t, c.ctx)
			if got != c.sql {
				t.Errorf("conditional UPDATE drifted from base 2cfe57b5 with no review-edit decision:\n got: %s\nwant: %s", got, c.sql)
			}
			if binds != c.binds {
				t.Errorf("bind count = %d, want %d", binds, c.binds)
			}
			if strings.Contains(got, "EXISTS") || strings.Contains(got, "rr.role") || strings.Contains(got, "product_plan_staff") {
				t.Errorf("denied conditional UPDATE must not mention the reviewer edge: %s", got)
			}
		})
	}
}

// Single read: the base path issues no SQL of its own (generic dbOps.Read plus a
// post-read owner guard). Denied ⇒ a non-owner staff is still refused and still no
// extra statement is run; workspace-wide staff and operators are unaffected.
func TestReviewEdit_DeniedSQLUnchanged_SingleRead(t *testing.T) {
	for _, c := range []struct {
		name      string
		ctx       context.Context
		wantFound bool
	}{
		{"staff_row_scoped_non_owner_refused", shapeStaffCtx(false), false},
		{"staff_workspace_wide_found", shapeStaffCtx(true), true},
		{"operator_found", shapeOperatorCtx(), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			found, stmts := captureReadStatements(t, c.ctx)
			if found != c.wantFound {
				t.Errorf("found = %v, want %v", found, c.wantFound)
			}
			if stmts != 0 {
				t.Errorf("denied single read must issue no repo SQL (base issued none), got %d", stmts)
			}
		})
	}
}

// Plain update: the denied path hands the generic Update the same field set as
// base — recorded_by / reviewed_by / reviewed_date are NOT stripped without a
// decision, and no extra SQL is issued.
func TestReviewEdit_DeniedSQLUnchanged_PlainUpdate(t *testing.T) {
	keys, stmts := captureUpdateKeys(t, shapeStaffCtx(false))
	if keys != baseUpdateKeys {
		t.Errorf("plain update field set = %q, want base %q", keys, baseUpdateKeys)
	}
	if stmts != 0 {
		t.Errorf("denied plain update must issue no repo SQL (base issued none), got %d", stmts)
	}
}
