//go:build postgresql

package operation

import "github.com/erniealice/espyna-golang/registry/entityid"

// Frozen pre-P5 has-data SQL. Its group and workspace predicates are the parity oracle.
func legacyGateRollupHasDataSQL(narrow string) string {
	return `
SELECT DISTINCT jp.template_phase_id
FROM ` + entityid.JobPhase + ` jp
JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
JOIN ` + entityid.JobTask + ` jt ON jt.job_phase_id = jp.id AND jt.active = true
JOIN ` + entityid.TaskOutcome + ` t ON t.job_task_id = jt.id AND t.active = true
WHERE jp.template_phase_id = ANY($1)
  AND j.workspace_id = $2
  AND jp.active = true` + narrow
}
