//go:build postgresql

package operation

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/registry/entityid"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
)

// ============================================================================
// Approval scope — plan 20260924-approval-role-workflow D3/D4.
//
// An approval transition is allowed only when verb ∧ scope ∧ state hold. The
// verb is the RBAC gate in the use case; the state is the uniform-source check
// in each transition; this file is the SCOPE axis for STAFF (kind 7) sessions:
//
//   - workspace scope: the caller holds approval_scope:workspace (resolved fresh
//     in-tx by the use case, carried in approvalctx.ScopeDecision) → any sheet;
//   - reviewer edge: every job in the sheet belongs to a product plan offering
//     the acting staff REVIEWS (product_plan_staff.role = 'reviewer', active).
//
// Operator sessions keep the pre-2026-09-24 behaviour (RBAC-only verify /
// publish / return) — owner decision D-RUN-3. Submit keeps its own D7 ownership
// gate (class edge or explicit assignment), widened by the reviewer edge.
// ============================================================================

// reviewerEdgeOwnedSQL is the reviewer-edge EXISTS fragment, correlated on the
// outer aliases j (job) and jp (job_phase). It mirrors classEdgeOwnedSQL's walk
// (member(client, active, workspace) → CURRENT section → class (active,
// workspace) → product_plan.product_id = j.output_product_id) but ends at the
// OFFERING-level product_plan_staff reviewer row instead of a per-class sgpps
// edge, so one reviewer row covers every section taking that offering. A legacy
// job (output_product_id NULL) never matches (fail closed).
func reviewerEdgeOwnedSQL(facetArgN, wsArgN int) string {
	return fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM `+entityid.SubscriptionGroupMember+` rm
			JOIN `+entityid.SubscriptionGroup+` rsg
			       ON rsg.id = rm.subscription_group_id AND rsg.status = 'current'
			JOIN `+entityid.SubscriptionGroupProductPlan+` rc
			       ON rc.subscription_group_id = rm.subscription_group_id AND rc.active AND rc.workspace_id = $%[2]d
			JOIN `+entityid.ProductPlan+` rpp
			       ON rpp.id = rc.product_plan_id AND rpp.product_id = j.output_product_id
			JOIN `+entityid.ProductPlanStaff+` rr
			       ON rr.product_plan_id = rpp.id AND rr.active
			      AND rr.role = '`+principalscope.ProductPlanStaffRoleReviewer+`'
			      AND rr.staff_id = $%[1]d AND rr.workspace_id = $%[2]d
			WHERE rm.client_id = j.client_id AND rm.active AND rm.workspace_id = $%[2]d
		)`, facetArgN, wsArgN)
}

// sheetOutsideReviewerScopeSQL probes whether ANY active phase of the sheet lies
// outside the acting staff's reviewer scope. Args: $1 template, $2 template
// phase, $3 workspace, $4 staff facet; narrow is the optional group predicate
// (placeholders $5/$3).
func sheetOutsideReviewerScopeSQL(narrow string) string {
	return `
		SELECT EXISTS (
			SELECT 1
			FROM ` + entityid.JobPhase + ` jp
			JOIN ` + entityid.Job + ` j ON j.id = jp.job_id
			WHERE jp.template_phase_id = $2 AND j.job_template_id = $1 AND j.workspace_id = $3
			  AND jp.active = true` + narrow + `
			  AND NOT ` + reviewerEdgeOwnedSQL(4, 3) + `
		)`
}

// assertApprovalScope enforces the scope axis for verify / publish / return.
// verb names the transition in errors. Operator sessions (kinds 1/2) pass
// unchanged; a STAFF session passes with workspace scope, otherwise EVERY phase
// of the sheet must be inside the re-proven staff facet's reviewer scope; every
// other kind fails closed. Fail closed on a malformed staff session, a facet
// that cannot be re-proven, or a probe error.
func assertApprovalScope(ctx context.Context, exec sqlexec.DBExecutor, verb, templateID, phaseID, wsID, groupID string) error {
	if principalscope.IsOperatorSession(ctx) {
		return nil
	}
	staffID, isStaff := principalscope.ActingStaff(ctx)
	if !isStaff {
		// Unresolved (0) and portal (3-6) kinds never act on approvals — fail closed.
		return fmt.Errorf("job_phase %s: principal kind is neither operator nor staff — fail closed", verb)
	}
	if staffID == "" {
		// A malformed STAFF session never trusts a scope decision (review wave-2 #1).
		return fmt.Errorf("job_phase %s: staff session has no resolved principal id — fail closed", verb)
	}
	d, _ := approvalctx.ScopeDecisionFromContext(ctx)
	if d.WorkspaceScope {
		return nil
	}
	facet, err := resolveStaffFacet(ctx, exec, wsID)
	if err != nil {
		return fmt.Errorf("job_phase %s: %w", verb, err)
	}
	narrow, narrowArgs := groupNarrowPredicate(groupID, 5, 3)
	var outside bool
	if err := exec.QueryRowContext(ctx, sheetOutsideReviewerScopeSQL(narrow),
		append([]any{templateID, phaseID, wsID, facet}, narrowArgs...)...).Scan(&outside); err != nil {
		return fmt.Errorf("job_phase %s: reviewer-scope probe: %w", verb, err)
	}
	if outside {
		return fmt.Errorf("job_phase %s: sheet is outside the acting staff's approval scope (needs approval_scope:workspace or a reviewer edge on every job's offering) — fail closed", verb)
	}
	return nil
}

// assertNotSelfVerify enforces separation of duties on verify (plan D4): unless
// the caller holds job_phase:verify_own, a sheet any of whose locked phases was
// submitted by the acting user cannot be verified by them. Applies to every
// session kind (Superadmin holds verify_own as data — owner D-RUN-1).
func assertNotSelfVerify(ctx context.Context, exec sqlexec.DBExecutor, locked []lockedPhase, actorUserID string) error {
	d, _ := approvalctx.ScopeDecisionFromContext(ctx)
	if d.VerifyOwn {
		return nil
	}
	if actorUserID == "" {
		return fmt.Errorf("job_phase verify: no trusted actor for the separation-of-duties check — fail closed")
	}
	ids := make([]string, len(locked))
	for i, lp := range locked {
		ids[i] = lp.id
	}
	var own bool
	if err := exec.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+entityid.JobPhase+` WHERE id = ANY($1) AND submitted_by = $2)`,
		pq.Array(ids), actorUserID).Scan(&own); err != nil {
		return fmt.Errorf("job_phase verify: separation-of-duties probe: %w", err)
	}
	if own {
		return fmt.Errorf("job_phase verify: the acting user submitted this sheet and lacks job_phase:verify_own (separation of duties) — denied")
	}
	return nil
}

// publishSourceState returns the uniform source state a publish may start from:
// VERIFIED always; a uniformly FOR_REVIEW sheet only when the caller holds
// job_phase:publish_unverified (plan D4). Any other / mixed state is refused.
func publishSourceState(ctx context.Context, locked []lockedPhase) (string, error) {
	if err := requireUniformSource(locked, apVerified); err == nil {
		return apVerified, nil
	}
	d, _ := approvalctx.ScopeDecisionFromContext(ctx)
	if d.PublishUnverified {
		if err := requireUniformSource(locked, apForReview); err == nil {
			return apForReview, nil
		}
	}
	return "", requireUniformSource(locked, apVerified)
}
