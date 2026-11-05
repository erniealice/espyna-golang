//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// outcomeInClassPrimaryScopeSQL is the main-teacher-override EXISTS fragment for
// one outcome (design-v2 rule 4): the outcome's job task sits in an active phase of
// a job in the workspace ($wsArgN) whose class the staff ($facetArgN) holds a
// PRIMARY class edge on. The edge walk is classPrimaryEdgeOwnedSQL, never copied.
func outcomeInClassPrimaryScopeSQL(jobTaskIDExpr string, facetArgN, wsArgN int) string {
	return fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM `+"job_task"+` cjt
			JOIN job_phase jp ON jp.id = cjt.job_phase_id AND jp.active
			JOIN job j ON j.id = jp.job_id AND j.workspace_id = $%[1]d
			WHERE cjt.id = %[2]s
			  AND `, wsArgN, jobTaskIDExpr) + classPrimaryEdgeOwnedSQL(facetArgN, wsArgN) + `
		)`
}

// inClassOverrideScope reports whether the ClassOverride decision admits reading an
// outcome the acting staff does not own: a primary class edge on its class. Fails
// closed on a denied decision, missing ids or any error.
func (r *PostgresTaskOutcomeRepository) inClassOverrideScope(ctx context.Context, jobTaskID, staffID string) bool {
	d, _ := reviewedit.DecisionFromContext(ctx)
	if !d.ClassOverride || staffID == "" || jobTaskID == "" {
		return false
	}
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return false
	}
	exec := r.executor(ctx)
	if exec == nil {
		return false
	}
	var in bool
	if err := exec.QueryRowContext(ctx,
		`SELECT `+outcomeInClassPrimaryScopeSQL("$1", 2, 3), jobTaskID, staffID, idn.WorkspaceID).Scan(&in); err != nil {
		log.Printf("WARN: task_outcome class override scope check failed (denied): %v", err)
		return false
	}
	return in
}
