//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"testing"

	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// TestIntegrationApprovalRollupReadOnlyParity exercises the source-owned CTE
// against a configured PostgreSQL lane and compares every returned phase with
// independent scalar observations. The session is forced read-only and uses one
// connection so the guard and timeouts apply to every probe.
func TestIntegrationApprovalRollupReadOnlyParity(t *testing.T) {
	db := openGroupNarrowDB(t)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for _, setting := range []string{
		"SET default_transaction_read_only = on",
		"SET statement_timeout = '15s'",
		"SET lock_timeout = '1s'",
	} {
		if _, err := db.Exec(setting); err != nil {
			t.Fatalf("set read-only integration guard %q: %v", setting, err)
		}
	}

	ctx := context.Background()
	const fixtureSQL = `
		SELECT j.job_template_id, j.workspace_id
		FROM job_phase jp
		JOIN job j ON j.id = jp.job_id
		WHERE jp.active = true
		  AND jp.template_phase_id IS NOT NULL
		  AND j.job_template_id IS NOT NULL
		  AND j.job_template_id <> ''
		  AND j.workspace_id IS NOT NULL
		  AND j.workspace_id <> ''
		GROUP BY j.job_template_id, j.workspace_id
		ORDER BY COUNT(*) DESC, j.job_template_id
		LIMIT 1`
	var templateID, workspaceID string
	if err := db.QueryRowContext(ctx, fixtureSQL).Scan(&templateID, &workspaceID); err != nil {
		t.Skipf("no active template-backed phase fixture: %v", err)
	}

	a := &PostgresOutcomeMatrixQuery{db: db}
	rollups, err := a.loadApprovalRollups(ctx, templateID, workspaceID, "")
	if err != nil {
		t.Fatalf("load one-read approval roll-up: %v", err)
	}
	if len(rollups) == 0 {
		t.Fatal("fixture has phases but one-read approval roll-up returned none")
	}
	assertApprovalRollupParity(t, ctx, db, templateID, workspaceID, "", rollups)

	// Exercise the real $3 group binding and subscription-origin membership
	// predicate. Status/data/blanks narrow to the group; hard freeze must remain
	// the ungrouped template-grain observation.
	const groupFixtureSQL = `
		SELECT j.job_template_id, j.workspace_id, sgm.subscription_group_id
		FROM job_phase jp
		JOIN job j ON j.id = jp.job_id
		JOIN subscription_group_member sgm
		  ON sgm.client_id = j.client_id
		 AND sgm.subscription_id = j.origin_id
		 AND sgm.workspace_id = j.workspace_id
		 AND sgm.active = true
		WHERE jp.active = true
		  AND jp.template_phase_id IS NOT NULL
		  AND j.job_template_id IS NOT NULL
		  AND j.job_template_id <> ''
		  AND j.workspace_id IS NOT NULL
		  AND j.workspace_id <> ''
		GROUP BY j.job_template_id, j.workspace_id, sgm.subscription_group_id
		ORDER BY COUNT(*) DESC, j.job_template_id, sgm.subscription_group_id
		LIMIT 1`
	var groupTemplateID, groupWorkspaceID, groupID string
	if err := db.QueryRowContext(ctx, groupFixtureSQL).Scan(&groupTemplateID, &groupWorkspaceID, &groupID); err != nil {
		t.Fatalf("resolve group-scoped approval fixture: %v", err)
	}
	groupRollups, err := a.loadApprovalRollups(ctx, groupTemplateID, groupWorkspaceID, groupID)
	if err != nil {
		t.Fatalf("load group-scoped one-read approval roll-up: %v", err)
	}
	if len(groupRollups) == 0 {
		t.Fatal("group fixture has phases but group-scoped approval roll-up returned none")
	}
	assertApprovalRollupParity(t, ctx, db, groupTemplateID, groupWorkspaceID, groupID, groupRollups)

	explainRows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+buildApprovalRollupSQL(""), templateID, workspaceID)
	if err != nil {
		t.Fatalf("EXPLAIN one-read approval roll-up: %v", err)
	}
	defer explainRows.Close()
	var explainLines int
	for explainRows.Next() {
		var line string
		if err := explainRows.Scan(&line); err != nil {
			t.Fatalf("scan EXPLAIN row: %v", err)
		}
		explainLines++
		if testing.Verbose() {
			t.Log(line)
		}
	}
	if err := explainRows.Err(); err != nil {
		t.Fatalf("iterate EXPLAIN rows: %v", err)
	}
	if explainLines == 0 {
		t.Fatal("EXPLAIN returned no plan lines")
	}
	t.Logf("verified %d ungrouped and %d group-scoped phase roll-ups for template %.8s…; EXPLAIN emitted %d lines", len(rollups), len(groupRollups), templateID, explainLines)
}

func assertApprovalRollupParity(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	templateID, workspaceID, groupID string,
	rollups []*matrixpb.PhaseApprovalRollup,
) {
	t.Helper()
	narrow, narrowArgs := groupNarrowPredicate(groupID, 4, 2)
	for _, got := range rollups {
		phaseID := got.GetJobTemplatePhaseId()
		statusSQL := `
			SELECT COUNT(*),
			       MIN(CASE jp.approval_status
			             WHEN 'PHASE_APPROVAL_STATUS_IN_PROGRESS' THEN 1
			             WHEN 'PHASE_APPROVAL_STATUS_FOR_REVIEW'  THEN 2
			             WHEN 'PHASE_APPROVAL_STATUS_VERIFIED'    THEN 3
			             WHEN 'PHASE_APPROVAL_STATUS_PUBLISHED'   THEN 4
			             ELSE 1 END),
			       COUNT(DISTINCT jp.approval_status)
			FROM job_phase jp
			JOIN job j ON j.id = jp.job_id
			WHERE j.job_template_id = $1
			  AND j.workspace_id = $2
			  AND jp.template_phase_id = $3
			  AND jp.active = true` + narrow
		args := append([]any{templateID, workspaceID, phaseID}, narrowArgs...)
		var targetCount int32
		var lowestRank, distinctStatuses int
		if err := db.QueryRowContext(ctx, statusSQL, args...).
			Scan(&targetCount, &lowestRank, &distinctStatuses); err != nil {
			t.Fatalf("independent status probe for %s: %v", phaseID, err)
		}

		hasDataSQL := `
			SELECT EXISTS (
			  SELECT 1
			  FROM job_phase jp
			  JOIN job j ON j.id = jp.job_id
			  JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active = true
			  JOIN task_outcome t ON t.job_task_id = jt.id AND t.active = true
			  WHERE j.job_template_id = $1
			    AND j.workspace_id = $2
			    AND jp.template_phase_id = $3
			    AND jp.active = true` + narrow + `
			)`
		var hasData bool
		if err := db.QueryRowContext(ctx, hasDataSQL, args...).Scan(&hasData); err != nil {
			t.Fatalf("independent data probe for %s: %v", phaseID, err)
		}

		blankSQL := `
			SELECT COUNT(*)
			FROM job_phase jp
			JOIN job j ON j.id = jp.job_id
			JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active = true
			JOIN template_task_criteria ttc
			  ON ttc.job_template_task_id = jt.template_task_id AND ttc.active = true
			JOIN outcome_criteria oc
			  ON oc.id = ttc.outcome_criteria_id AND oc.active = true
			WHERE j.job_template_id = $1
			  AND j.workspace_id = $2
			  AND jp.template_phase_id = $3
			  AND jp.active = true` + narrow + `
			  AND COALESCE(ttc.required_override, oc.required) = true
			  AND NOT EXISTS (
			    SELECT 1
			    FROM task_outcome t
			    WHERE t.job_task_id = jt.id
			      AND t.criteria_version_id = ttc.outcome_criteria_id
			      AND t.active = true
			  )`
		var blanks int32
		if err := db.QueryRowContext(ctx, blankSQL, args...).Scan(&blanks); err != nil {
			t.Fatalf("independent blank probe for %s: %v", phaseID, err)
		}

		frozen, err := sheetHardFrozen(ctx, db, templateID, phaseID, workspaceID, "")
		if err != nil {
			t.Fatalf("independent freeze probe for %s: %v", phaseID, err)
		}

		wantStatus := approvalRankToStatus(lowestRank)
		wantMixed := distinctStatuses > 1
		wantBlanks := int32(0)
		if wantStatus == jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS && !wantMixed {
			wantBlanks = blanks
		}
		if got.GetTargetCount() != targetCount ||
			got.GetStatus() != wantStatus ||
			got.GetMixed() != wantMixed ||
			got.GetHasData() != hasData ||
			got.GetBlankRequiredCount() != wantBlanks ||
			got.GetHardFrozen() != frozen {
			t.Errorf(
				"phase %s roll-up mismatch: got={count:%d status:%v mixed:%v data:%v blanks:%d frozen:%v} want={count:%d status:%v mixed:%v data:%v blanks:%d frozen:%v}",
				phaseID,
				got.GetTargetCount(), got.GetStatus(), got.GetMixed(), got.GetHasData(), got.GetBlankRequiredCount(), got.GetHardFrozen(),
				targetCount, wantStatus, wantMixed, hasData, wantBlanks, frozen,
			)
		}
	}
}
