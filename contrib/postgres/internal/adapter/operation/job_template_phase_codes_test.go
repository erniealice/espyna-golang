//go:build postgresql

package operation

import (
	"strings"
	"testing"
)

func TestPhaseCodesByScheduleSQLScopeAndGrain(t *testing.T) {
	for _, fragment := range []string{
		"pp.price_schedule_id = ps.id AND pp.workspace_id = $1 AND pp.active",
		"s.price_plan_id = pp.id AND s.workspace_id = $1 AND s.active",
		"j.origin_id = s.id AND j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'",
		"j.workspace_id = $1 AND j.active",
		"jt.workspace_id = $1 AND jt.active",
		"jtp.job_template_id = jt.id AND jtp.active",
		"ps.id = $2 AND ps.workspace_id = $1 AND ps.active",
		"SELECT DISTINCT jtp.code, jtp.name, jt.id AS template_id",
		"array_agg(name ORDER BY name)",
		"count(DISTINCT template_id)",
	} {
		if !strings.Contains(phaseCodesByScheduleSQL, fragment) {
			t.Errorf("missing phase-code SQL invariant %q", fragment)
		}
	}
}
