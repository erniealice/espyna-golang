//go:build postgresql

package operation

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// GetPhaseApprovalGateRollup is the report-card render gate's group-grain input
// read (plan 20260729 Phase 2, locked design h2-synthesis.md): for ONE validated
// delivery group and the card's target template phases it returns, per template
// phase, the exact gate inputs (target_count / any_workflow_entered /
// all_published / has_data) plus the applied-group echo.
//
// Extraction of the loadApprovalRollups precedent (same file family): the group
// predicate is THE shared groupNarrowPredicate — never re-authored, and never
// the looser outcome-matrix cells predicate (`client_id IN (...)`, which drops
// the subscription_id = j.origin_id delivery pin and would re-open the
// foreign-year partial-roster hazard). It is applied in SQL BEFORE aggregation,
// and neither statement carries a LIMIT/OFFSET, so the H-1 "narrow after the
// page" hazard is absent by construction.
//
// Fail-closed input contract — every "cannot apply" state is an ERROR, never an
// empty success (the deliberate inversion of GetOutcomeMatrix's empty-success
// degrades, and the same posture as narrowPhasesToGroup): nil request, empty
// subscription_group_id, empty job_template_phase_ids, and a missing/empty
// trusted workspace identity all refuse before any query runs. A requested
// template phase with zero group members is simply ABSENT from the response —
// the consumer's coverage check turns absence into its own fail-closed outcome.

// gateAnyWorkflowEnteredSQLExpr is the SQL twin of the consumer-side
// "workflow entered" predicate (fayna phaseWorkflowEntered): a phase has
// entered the approval workflow when its status advanced beyond
// IN_PROGRESS/UNSPECIFIED, OR any of the four approval audit stamps is set
// (submit/verify/publish/return — a returned sheet can be back at IN_PROGRESS
// with only returned_by to show for it). The stamps are COALESCE-guarded
// because the protojson storage bridge persists unset scalars as SQL NULL.
// approval_status itself is schema-NOT NULL (staged migration 20260719) so the
// NOT IN needs no NULL guard; the UNSPECIFIED token is listed anyway — it is
// the proto zero value and must never count as entered.
//
// Kept as a named constant so the Go↔SQL parity test evaluates EXACTLY the
// expression the shipped statement embeds.
const gateAnyWorkflowEnteredSQLExpr = `jp.approval_status NOT IN ('PHASE_APPROVAL_STATUS_IN_PROGRESS',
                                    'PHASE_APPROVAL_STATUS_UNSPECIFIED')
         OR COALESCE(jp.submitted_by,'') <> ''
         OR COALESCE(jp.verified_by ,'') <> ''
         OR COALESCE(jp.published_by,'') <> ''
         OR COALESCE(jp.returned_by ,'') <> ''`

// gateRollupStatusSQL builds Statement A — the per-template-phase status
// aggregate over the group-narrowed sheet. The predicate arrives pre-built
// (emitted ONCE by the caller from groupNarrowPredicate(groupID, 3, 2)) and is
// hosted here exactly once. Binds: $1 = template-phase id array, $2 = trusted
// workspace (also bound to sgm_g.workspace_id through the predicate's wsArgN),
// $3 = the group id appended by the predicate. No LIMIT/OFFSET; no ORDER BY —
// the consumer keys rollups by id, order is irrelevant.
func gateRollupStatusSQL(narrow string) string {
	return `
SELECT jp.template_phase_id,
       COUNT(*) AS target_count,
       BOOL_OR(
         ` + gateAnyWorkflowEnteredSQLExpr + `
       ) AS any_workflow_entered,
       BOOL_AND(jp.approval_status = 'PHASE_APPROVAL_STATUS_PUBLISHED') AS all_published
FROM ` + entityid.JobPhase + ` jp
JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
WHERE jp.template_phase_id = ANY($1)
  AND j.workspace_id = $2
  AND jp.active = true
  AND jp.template_phase_id IS NOT NULL` + narrow + `
GROUP BY jp.template_phase_id`
}

// gateRollupHasDataSQL builds Statement B — the has_data set probe (the exact
// loadApprovalRollups (B) shape re-keyed to template_phase_id = ANY($1)): a
// template phase is present iff ANY active task_outcome exists under any active
// job_task of any group-narrowed sheet member. Same binds as Statement A; the
// predicate is hosted exactly once; no LIMIT/OFFSET.
func gateRollupHasDataSQL(narrow string) string {
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

// gateAgg is one Statement-A row.
type gateAgg struct {
	templatePhaseID    string
	targetCount        int32
	anyWorkflowEntered bool
	allPublished       bool
}

// assembleGateRollups builds the response rollups from the two statement
// results. AppliedSubscriptionGroupId is the REQUEST's group id verbatim — the
// echo is the applied-narrow proof, and it is only ever stamped onto rows
// produced by a query that bound $3 = groupID. Pure function so the echo
// contract is pinned without a database.
func assembleGateRollups(groupID string, aggs []gateAgg, hasData map[string]bool) []*matrixpb.PhaseApprovalGateRollup {
	out := make([]*matrixpb.PhaseApprovalGateRollup, 0, len(aggs))
	for _, g := range aggs {
		out = append(out, &matrixpb.PhaseApprovalGateRollup{
			JobTemplatePhaseId:         g.templatePhaseID,
			AppliedSubscriptionGroupId: groupID,
			TargetCount:                g.targetCount,
			AnyWorkflowEntered:         g.anyWorkflowEntered,
			AllPublished:               g.allPublished,
			HasData:                    hasData[g.templatePhaseID],
		})
	}
	return out
}

// GetPhaseApprovalGateRollup implements the generated OutcomeMatrixServiceServer
// method. It rides PostgresOutcomeMatrixQuery on purpose: the struct's factory
// registration (outcome_matrix_query.go init) is postgres-only and fail-closed
// by absence, so a provider that does not implement this read cannot silently
// inherit anything — the composition port is simply nil there.
func (a *PostgresOutcomeMatrixQuery) GetPhaseApprovalGateRollup(
	ctx context.Context,
	req *matrixpb.GetPhaseApprovalGateRollupRequest,
) (*matrixpb.GetPhaseApprovalGateRollupResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("phase_approval_gate: request is required (fail closed)")
	}
	groupID := req.GetSubscriptionGroupId()
	if groupID == "" {
		return nil, fmt.Errorf("phase_approval_gate: subscription_group_id is required — an empty group must never mean an unnarrowed read (fail closed)")
	}
	tpIDs := req.GetJobTemplatePhaseIds()
	if len(tpIDs) == 0 {
		return nil, fmt.Errorf("phase_approval_gate: job_template_phase_ids must be non-empty (fail closed)")
	}
	// identity.FromContext, not identity.Must: a missing identity must refuse,
	// not panic — and unlike GetOutcomeMatrix it must refuse with an ERROR,
	// because an empty success here would read as "no sheets" to a
	// document-integrity consumer.
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return nil, fmt.Errorf("phase_approval_gate: a trusted workspace identity is required — refusing an unscoped read (fail closed)")
	}
	workspaceID := id.WorkspaceID

	// Emit the shared predicate ONCE; both statements host that single string.
	narrow, narrowArgs := groupNarrowPredicate(groupID, 3, 2)
	if narrow == "" {
		// Unreachable (groupID is non-empty above) and kept unreachable on
		// purpose: running the residual unnarrowed aggregate would be the exact
		// fail-open this read exists to make unrepresentable.
		return nil, fmt.Errorf("phase_approval_gate: group narrow predicate declined to emit — refusing the unnarrowed residual (fail closed)")
	}
	args := append([]any{pq.Array(tpIDs), workspaceID}, narrowArgs...)

	// Statement A — status aggregate per template phase.
	rows, err := a.db.QueryContext(ctx, gateRollupStatusSQL(narrow), args...)
	if err != nil {
		return nil, fmt.Errorf("phase_approval_gate: status aggregate query: %w", err)
	}
	defer rows.Close()
	var aggs []gateAgg
	for rows.Next() {
		var g gateAgg
		if err := rows.Scan(&g.templatePhaseID, &g.targetCount, &g.anyWorkflowEntered, &g.allPublished); err != nil {
			return nil, fmt.Errorf("phase_approval_gate: scan status aggregate: %w", err)
		}
		aggs = append(aggs, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("phase_approval_gate: status aggregate rows: %w", err)
	}

	// Statement B — has_data set probe (skipped only when A matched nothing:
	// there is no row an absent probe could wrongly mark).
	hasData := map[string]bool{}
	if len(aggs) > 0 {
		if err := a.scanPhaseIDSet(ctx, gateRollupHasDataSQL(narrow), hasData, args...); err != nil {
			return nil, err
		}
	}

	return &matrixpb.GetPhaseApprovalGateRollupResponse{
		Rollups: assembleGateRollups(groupID, aggs, hasData),
		Success: true,
	}, nil
}
