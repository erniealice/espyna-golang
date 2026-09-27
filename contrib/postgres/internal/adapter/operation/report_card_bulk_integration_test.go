//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	phaseoutcomesummarypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

// Read-only clone oracle: both bulk readers retain their per-item row set and
// return nothing for a real id under a different trusted workspace.
func TestReportCardBulk_WorkspaceAndParity(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	var jobID, templateID, workspaceID string
	if err := db.QueryRow(`SELECT pos.job_id, j.job_template_id, j.workspace_id
		FROM `+entityid.PhaseOutcomeSummary+` pos
		JOIN `+entityid.Job+` j ON j.id = pos.job_id
		JOIN `+entityid.JobTemplatePhase+` jtp ON jtp.job_template_id = j.job_template_id AND jtp.active
		WHERE pos.active AND j.workspace_id IS NOT NULL
		LIMIT 1`).Scan(&jobID, &templateID, &workspaceID); err != nil {
		t.Skipf("clone has no job with phase summary and template phase: %v", err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspaceID})
	foreign := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "not-this-workspace"})
	phase := NewPostgresPhaseOutcomeSummaryRepository(postgresCore.NewWorkspaceAwareOperations(db), "phase_outcome_summary").(*PostgresPhaseOutcomeSummaryRepository)
	template := NewPostgresJobTemplatePhaseRepository(postgresCore.NewWorkspaceAwareOperations(db), "job_template_phase").(*PostgresJobTemplatePhaseRepository)

	oldPhase, err := phase.ListByJob(ctx, &phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobRequest{JobId: jobID})
	if err != nil {
		t.Fatal(err)
	}
	bulkPhase, err := phase.ListByJobs(ctx, []string{jobID})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldPhase.GetPhaseOutcomeSummarys()) == 0 || !reflect.DeepEqual(oldPhase.GetPhaseOutcomeSummarys(), bulkPhase) {
		t.Fatalf("phase rows differ: per-item=%d bulk=%d", len(oldPhase.GetPhaseOutcomeSummarys()), len(bulkPhase))
	}
	oldTemplate, err := template.ListByJobTemplate(ctx, &jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: templateID})
	if err != nil {
		t.Fatal(err)
	}
	bulkTemplate, err := template.ListByTemplates(ctx, []string{templateID})
	if err != nil {
		t.Fatal(err)
	}
	if len(oldTemplate.GetJobTemplatePhases()) == 0 || len(oldTemplate.GetJobTemplatePhases()) != len(bulkTemplate) {
		t.Fatalf("template rows differ: per-item=%d bulk=%d", len(oldTemplate.GetJobTemplatePhases()), len(bulkTemplate))
	}
	for i, old := range oldTemplate.GetJobTemplatePhases() {
		if old.GetId() != bulkTemplate[i].GetId() || old.GetCode() != bulkTemplate[i].GetCode() {
			t.Fatalf("template phase %d code/id changed", i)
		}
	}
	if rows, err := phase.ListByJobs(foreign, []string{jobID}); err != nil || len(rows) != 0 {
		t.Fatalf("foreign workspace phase rows=%d err=%v", len(rows), err)
	}
	if rows, err := template.ListByTemplates(foreign, []string{templateID}); err != nil || len(rows) != 0 {
		t.Fatalf("foreign workspace template rows=%d err=%v", len(rows), err)
	}
	staffWithoutID := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspaceID, PrincipalType: 7})
	if rows, err := phase.ListByJobs(staffWithoutID, []string{jobID}); err != nil || len(rows) != 0 {
		t.Fatalf("empty staff principal phase rows=%d err=%v", len(rows), err)
	}
}

// countConnector observes driver statement execution, including database/sql's
// prepare fallback. It counts actual calls into the PostgreSQL driver, not Go
// closure invocations or estimated EXPLAIN nodes.
type countConnector struct {
	base driver.Connector
	n    *atomic.Int64
}

func (c countConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return countConn{Conn: conn, n: c.n}, nil
}
func (c countConnector) Driver() driver.Driver { return c.base.Driver() }

type countConn struct {
	driver.Conn
	n *atomic.Int64
}

func (c countConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return countStmt{Stmt: stmt, n: c.n}, nil
}

type countStmt struct {
	driver.Stmt
	n *atomic.Int64
}

func (s countStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.n.Add(1)
	return s.Stmt.Query(args)
}

func TestReportCardBulk_DriverStatementCount(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	base, err := pq.NewConnector(dsn)
	if err != nil {
		t.Fatal(err)
	}
	var statements atomic.Int64
	db := sql.OpenDB(countConnector{base: base, n: &statements})
	defer db.Close()
	var readOnly string
	if err := db.QueryRow(`SELECT current_setting('default_transaction_read_only')`).Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only clone required: setting=%q err=%v", readOnly, err)
	}
	var workspaceID, subscriptionID string
	if err := db.QueryRow(`SELECT j.workspace_id, j.origin_id FROM `+entityid.PhaseOutcomeSummary+` pos JOIN `+entityid.Job+` j ON j.id=pos.job_id WHERE pos.active AND j.origin_type='ORIGIN_TYPE_SUBSCRIPTION' GROUP BY j.workspace_id, j.origin_id HAVING count(DISTINCT j.id)>=24 ORDER BY count(DISTINCT j.id) DESC LIMIT 1`).Scan(&workspaceID, &subscriptionID); err != nil {
		t.Skipf("no subscription with 24 phase-summary jobs: %v", err)
	}
	rows, err := db.Query(`SELECT DISTINCT pos.job_id, j.job_template_id FROM `+entityid.PhaseOutcomeSummary+` pos JOIN `+entityid.Job+` j ON j.id=pos.job_id WHERE pos.active AND j.workspace_id=$1 AND j.origin_id=$2 ORDER BY pos.job_id LIMIT 24`, workspaceID, subscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	var jobIDs []string
	jobTemplates := map[string]string{}
	for rows.Next() {
		var jid, tid string
		if err := rows.Scan(&jid, &tid); err != nil {
			t.Fatal(err)
		}
		jobIDs = append(jobIDs, jid)
		jobTemplates[jid] = tid
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if len(jobIDs) < 24 {
		t.Skipf("workspace has only %d eligible jobs; need 24", len(jobIDs))
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspaceID})
	phase := NewPostgresPhaseOutcomeSummaryRepository(postgresCore.NewWorkspaceAwareOperations(db), "phase_outcome_summary").(*PostgresPhaseOutcomeSummaryRepository)
	template := NewPostgresJobTemplatePhaseRepository(postgresCore.NewWorkspaceAwareOperations(db), "job_template_phase").(*PostgresJobTemplatePhaseRepository)
	for _, n := range []int{1, 12, 24} {
		t.Run(fmt.Sprintf("%d-jobs", n), func(t *testing.T) {
			templateIDs := make([]string, 0, n)
			seenTemplate := map[string]bool{}
			for _, jid := range jobIDs[:n] {
				tid := jobTemplates[jid]
				if !seenTemplate[tid] {
					seenTemplate[tid] = true
					templateIDs = append(templateIDs, tid)
				}
			}
			statements.Store(0)
			for _, jid := range jobIDs[:n] {
				if _, err := phase.ListByJob(ctx, &phaseoutcomesummarypb.ListPhaseOutcomeSummarysByJobRequest{JobId: jid}); err != nil {
					t.Fatal(err)
				}
			}
			beforePhase := statements.Load()
			statements.Store(0)
			if _, err := phase.ListByJobs(ctx, jobIDs[:n]); err != nil {
				t.Fatal(err)
			}
			afterPhase := statements.Load()
			statements.Store(0)
			for _, tid := range templateIDs {
				if _, err := template.ListByJobTemplate(ctx, &jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: tid}); err != nil {
					t.Fatal(err)
				}
			}
			beforeTemplate := statements.Load()
			statements.Store(0)
			if _, err := template.ListByTemplates(ctx, templateIDs); err != nil {
				t.Fatal(err)
			}
			afterTemplate := statements.Load()
			if beforePhase != int64(n) || afterPhase != 1 || beforeTemplate != int64(len(templateIDs)) || afterTemplate != 1 {
				t.Fatalf("driver statements phase %d→%d template %d→%d", beforePhase, afterPhase, beforeTemplate, afterTemplate)
			}
			t.Logf("driver SQL statements for one subscription, %d jobs, %d templates: phase %d→%d; template %d→%d; combined %d→%d", n, len(templateIDs), beforePhase, afterPhase, beforeTemplate, afterTemplate, beforePhase+beforeTemplate, afterPhase+afterTemplate)
		})
	}
}
