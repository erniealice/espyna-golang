//go:build postgresql

// Package principalscope provides GENERIC, cross-vertical row-scoping for the
// active session principal. It confines an adapter's operational reads to the
// rows the current principal kind is entitled to — independent of any vertical
// (education/teacher, health/practitioner, field-service/technician,
// professional-services/consultant all share the same staff→job→client delivery
// graph). It is the per-principal analogue of the per-workspace tenant scope in
// adapter/core (WorkspaceAwareOperations), and mirrors the delegate IDOR junction
// scope in adapter/entity/delegate.go.
//
// The scope value is always taken from the SESSION identity (identity.FromContext),
// never from a request parameter, so it cannot be spoofed. Every helper is
// fail-closed: a staff principal with no resolved id yields an always-false
// predicate (zero rows), and a non-staff / no-identity caller leaves the query
// unchanged (preserving existing workspace/client scoping).
package principalscope

import (
	"context"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// PrincipalTypeStaff is the integer value of the esqyma
// domain.entity.v1.PrincipalType member PRINCIPAL_TYPE_STAFF. When the session's
// active binding carries this kind, the principal_id IS the acting staff.id and
// operational reads must be confined to that staff member's own rows.
const PrincipalTypeStaff int32 = 7

// originTypeSubscription is the text token the job table stores for the esqyma
// domain.operation.v1.OriginType member ORIGIN_TYPE_SUBSCRIPTION (jobs persist
// the full protojson enum name). The subscription_seat tier below matches it
// exactly so the seat→job join only reaches subscription-originated jobs.
const originTypeSubscription = "ORIGIN_TYPE_SUBSCRIPTION"

// StaffRowScope reports whether the active session principal is a STAFF principal
// and, if so, the staff.id its operational reads must be confined to.
//
// The scope value is identity.PrincipalID — the acting staff.id from the SESSION
// binding (stamped by the session middleware), NEVER a request param. applies==false
// for every non-staff principal (operator/client/supplier/delegate, a pre-selection
// session with no resolved binding, or a service-to-service / CLI context with no
// identity at all) — those callers leave their query unchanged.
//
// FAIL-CLOSED: a STAFF principal with an empty PrincipalID (a malformed session)
// returns staffID=="" with applies==true; the *Clause helpers turn that into an
// always-false predicate, so such a session sees zero rows.
func StaffRowScope(ctx context.Context) (staffID string, applies bool) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil {
		return "", false
	}
	if id.PrincipalType != PrincipalTypeStaff {
		return "", false
	}
	return id.PrincipalID, true
}

// workspaceScope returns the SESSION identity's WorkspaceID (empty when there is
// no identity). It is the SAME source StaffRowScope reads its principal from, so
// the workspace bound woven into the reachable-set subqueries below cannot be
// spoofed by a request parameter. An empty WorkspaceID binds the workspace column
// to the empty string (matches no row) — a malformed staff session then sees zero
// rows, consistent with the fail-closed contract.
func workspaceScope(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok && id != nil {
		return id.WorkspaceID
	}
	return ""
}

// reachableClientUnion is the graph-derived set of client.id values the acting
// staff.id ($staffP) serves within one workspace ($wsP): the client_id of any job
// the staff is assigned to (job_task.assigned_to) OR has recorded/reviewed an
// outcome for (task_outcome.recorded_by|reviewed_by), joined up through
// job_phase to job.client_id, OR whose origin subscription carries the staff's
// active seat matched to the job's deliverable (subscription_seat.staff_id, the
// assignment source of truth: the seat's product_plan.product_id must equal the
// job template's output_product_id; a template with no output_product_id matches
// no seat — inner-join, fail-closed), OR whose origin subscription's cohort the
// staff is the class-edge servicer for (subscription_group_product_plan_staff,
// "sgpps" — the UI-maintained "who services this cohort's offering" source of
// truth, matched to the job's deliverable via product_plan.product_id ==
// job.output_product_id). The workspace bound (belt-and-suspenders) confines the
// graph walk to the session workspace even if an outer query's workspace
// predicate is bypassed; the seat branch binds BOTH sides (job AND seat carry
// their own workspace_id) because job.origin_id is plain text, not an FK. Table
// names come from registry/entityid (no literals).
func reachableClientUnion(staffP, wsP int) string {
	s := fmt.Sprintf("$%d", staffP)
	w := fmt.Sprintf("$%d", wsP)
	return "SELECT j.client_id FROM " + entityid.Job + " j" +
		" JOIN " + entityid.JobPhase + " jp ON jp.job_id = j.id" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" WHERE jt.assigned_to = " + s + " AND j.workspace_id = " + w +
		" UNION " +
		"SELECT j2.client_id FROM " + entityid.Job + " j2" +
		" JOIN " + entityid.JobPhase + " jp2 ON jp2.job_id = j2.id" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" WHERE (t.recorded_by = " + s + " OR t.reviewed_by = " + s + ") AND j2.workspace_id = " + w +
		" UNION " +
		"SELECT j3.client_id FROM " + entityid.Job + " j3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = j3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = j3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = " + s + " AND ss.status = 'active' AND ss.active = true" +
		" AND j3.origin_type = '" + originTypeSubscription + "'" +
		" AND j3.workspace_id = " + w + " AND ss.workspace_id = " + w +
		" UNION " +
		// Class-edge tier (OPTIMIZED, 4 tables): the acting staff is the class-edge
		// servicer (sgpps) for the job's subject. NO job_template hop —
		// job.output_product_id is populated by spawn, so the subject matches on the
		// job directly; NO subscription_group / subscription hops —
		// subscription_group_member carries BOTH subscription_group_id AND
		// subscription_id. Drives from the staff's OWN sgpps edges (idx_..._staff_id,
		// a tiny starting set) and is fail-closed identically to the tiers above: an
		// empty staff/workspace bind matches no sgpps row (both filters land on the
		// edge), so it yields zero rows and cannot be widened by a request param.
		"SELECT jce.client_id FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = " + s + " AND e.active AND e.workspace_id = " + w
}

// reachableJobUnion is the graph-derived set of job.id values the acting staff.id
// ($staffP) is assigned to, has recorded/reviewed an outcome for, holds an
// active subscription_seat on the job's origin subscription matched to the job's
// deliverable (the seat's product_plan.product_id must equal the job template's
// output_product_id; a template with no output_product_id matches no seat —
// inner-join, fail-closed), OR is the class-edge servicer (sgpps) for the job's
// subject on the job's origin cohort (matched to the job's deliverable via
// product_plan.product_id == job.output_product_id), within one workspace ($wsP).
// job_phase/job_task carry no workspace_id, so the workspace bound is enforced by
// joining job (aliased jw/jw2) on the phase's job_id; the seat branch binds BOTH
// sides (job AND seat carry their own workspace_id) because job.origin_id is plain
// text, not an FK. Table names come from registry/entityid (no literals).
func reachableJobUnion(staffP, wsP int) string {
	s := fmt.Sprintf("$%d", staffP)
	w := fmt.Sprintf("$%d", wsP)
	return "SELECT jp.job_id FROM " + entityid.JobPhase + " jp" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" JOIN " + entityid.Job + " jw ON jw.id = jp.job_id AND jw.workspace_id = " + w +
		" WHERE jt.assigned_to = " + s +
		" UNION " +
		"SELECT jp2.job_id FROM " + entityid.JobPhase + " jp2" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" JOIN " + entityid.Job + " jw2 ON jw2.id = jp2.job_id AND jw2.workspace_id = " + w +
		" WHERE t.recorded_by = " + s + " OR t.reviewed_by = " + s +
		" UNION " +
		"SELECT jw3.id FROM " + entityid.Job + " jw3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = jw3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = jw3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = " + s + " AND ss.status = 'active' AND ss.active = true" +
		" AND jw3.origin_type = '" + originTypeSubscription + "'" +
		" AND jw3.workspace_id = " + w + " AND ss.workspace_id = " + w +
		" UNION " +
		// Class-edge tier (OPTIMIZED, 4 tables): the acting staff is the class-edge
		// servicer (sgpps) for the job's subject. NO job_template hop —
		// job.output_product_id is populated by spawn, so the subject matches on the
		// job directly; NO subscription_group / subscription hops —
		// subscription_group_member carries BOTH subscription_group_id AND
		// subscription_id. Drives from the staff's OWN sgpps edges (idx_..._staff_id,
		// a tiny starting set) and is fail-closed identically to the tiers above: an
		// empty staff/workspace bind matches no sgpps row (both filters land on the
		// edge), so it yields zero rows and cannot be widened by a request param.
		"SELECT jce.id FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = " + s + " AND e.active AND e.workspace_id = " + w
}

// StaffScopeClause returns a SQL predicate fragment that confines a read to the
// active STAFF principal's own rows on the given staff.id column, with the
// positional bind args to append in order.
//
//   - non-staff principal  → ("", nil): the caller's query is unchanged.
//   - staff, empty staff.id → (" AND 1=0", nil): fail-closed, zero rows.
//   - staff, real staff.id  → (" AND <col> = $<nextParam>", []any{staffID}).
//
// col is always a hardcoded, adapter-controlled column expression (never
// request-derived), so there is no injection surface. nextParam is the 1-based
// index of the NEXT positional placeholder (existing-arg-count + 1); the caller
// appends the returned args after its existing args so the indexes line up.
func StaffScopeClause(ctx context.Context, col string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	return fmt.Sprintf(" AND %s = $%d", col, nextParam), []any{staffID}
}

// StaffScopeClauseAny is StaffScopeClause for rows that carry the staff.id on more
// than one authorship axis (e.g. task_outcome, owned as recorded_by OR reviewed_by).
// The single session staff.id is bound once and reused across every column via the
// shared $<nextParam> placeholder; the predicate matches when ANY column equals it.
// Same non-staff-unchanged / empty-id-fail-closed contract. cols are all hardcoded,
// adapter-controlled column expressions (no injection).
func StaffScopeClauseAny(ctx context.Context, cols []string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s = $%d", c, nextParam)
	}
	return " AND (" + strings.Join(parts, " OR ") + ")", []any{staffID}
}

// StaffReachableClientClause confines a CLIENT query (the client row aliased as
// clientAlias, e.g. "c") to the clients the active STAFF principal serves — the
// client_id of any job the staff is assigned to, has recorded/reviewed an outcome
// for, or holds an active deliverable-matched subscription_seat on the origin
// subscription of, via the GENERIC operational-delivery graph
// (job_task.assigned_to / task_outcome.recorded_by|reviewed_by → job_phase →
// job.client_id, plus subscription_seat.staff_id → job.origin_id with the
// plan-product↔template-output match). Same 3-state contract as
// StaffScopeClause; emits an IN(<graph subquery>) because clients carry no
// staff column. The reviewed_by axis keeps parity with the task_outcome adapter.
//
// Two positional args are appended in order: $nextParam = the session staff.id,
// $(nextParam+1) = the session workspace.id (belt-and-suspenders workspace bound
// inside the graph subquery). Callers MUST advance their placeholder counter by
// len(args) (== 2), not a hardcoded +1.
func StaffReachableClientClause(ctx context.Context, clientAlias string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	sub := reachableClientUnion(nextParam, nextParam+1)
	return fmt.Sprintf(" AND %s.id IN (%s)", clientAlias, sub), []any{staffID, workspaceScope(ctx)}
}

// StaffReachableJobClause confines a JOB query (the job row aliased as jobAlias,
// e.g. "j") to the jobs the active STAFF principal is assigned to, has
// recorded/reviewed an outcome for, or holds an active deliverable-matched
// subscription_seat on the origin subscription of. Same 3-state contract;
// graph-derived through job_phase/job_task/task_outcome/subscription_seat
// (seat tier matched via job_template/product_plan).
//
// Two positional args are appended in order: $nextParam = the session staff.id,
// $(nextParam+1) = the session workspace.id. Callers MUST advance their
// placeholder counter by len(args) (== 2), not a hardcoded +1.
func StaffReachableJobClause(ctx context.Context, jobAlias string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	sub := reachableJobUnion(nextParam, nextParam+1)
	return fmt.Sprintf(" AND %s.id IN (%s)", jobAlias, sub), []any{staffID, workspaceScope(ctx)}
}

// StaffReachableClientExistsSQL returns a query that yields a single bool: whether
// the given staff.id can reach the given client.id via the delivery graph, an
// active deliverable-matched subscription_seat on the client's job's origin
// subscription, OR an active class edge (sgpps) servicing the client's cohort
// offering. It is
// for the generic dbOps by-id Read paths (ReadClient / item views) that have no
// WHERE seam — run it (after StaffRowScope reports a staff principal) and treat a
// false / no-row result as not-found, so an out-of-scope client is indistinguishable
// from a missing one. Args order: $1 = staffID, $2 = clientID, $3 = workspaceID.
//
// The 4th (class-edge) branch MIRRORS reachableClientUnion's 4th branch exactly so
// the by-id EXISTS check and the List row-set seam stay consistent — a teacher
// reachable ONLY via a class edge (the AY-2627 zero-seat shape) must not get a
// false not-found when opening the row by id (CF-2). Same 4-table join, same
// member-sourced subscription↔job match, same deliverable match; fail-closed on an
// empty staff/workspace bind (both land on the sgpps edge).
func StaffReachableClientExistsSQL() string {
	return "SELECT EXISTS(" +
		"SELECT 1 FROM " + entityid.Job + " j" +
		" JOIN " + entityid.JobPhase + " jp ON jp.job_id = j.id" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" WHERE jt.assigned_to = $1 AND j.client_id = $2 AND j.workspace_id = $3" +
		" UNION " +
		"SELECT 1 FROM " + entityid.Job + " j2" +
		" JOIN " + entityid.JobPhase + " jp2 ON jp2.job_id = j2.id" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" WHERE (t.recorded_by = $1 OR t.reviewed_by = $1) AND j2.client_id = $2 AND j2.workspace_id = $3" +
		" UNION " +
		"SELECT 1 FROM " + entityid.Job + " j3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = j3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = j3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = $1 AND ss.status = 'active' AND ss.active = true" +
		" AND j3.origin_type = '" + originTypeSubscription + "'" +
		" AND j3.client_id = $2 AND j3.workspace_id = $3 AND ss.workspace_id = $3" +
		" UNION " +
		// Class-edge tier (mirrors reachableClientUnion's 4th branch): the staff is
		// the sgpps servicer for the client's cohort offering, matched to the job's
		// deliverable via product_plan.product_id == job.output_product_id. Both binds
		// land on the edge ($1 staff, $3 workspace), so an empty bind yields no row.
		"SELECT 1 FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = $1 AND e.active AND e.workspace_id = $3 AND jce.client_id = $2" +
		")"
}

// StaffReachableJobExistsSQL is StaffReachableClientExistsSQL for a job.id (the
// generic dbOps by-id ReadJob path). Args order: $1 = staffID, $2 = jobID,
// $3 = workspaceID. The 4th (class-edge) branch MIRRORS reachableJobUnion's 4th
// branch so a class-edge-only teacher (AY-2627 zero-seat shape) does not get a
// false not-found when opening a job by id (CF-2); fail-closed on empty
// staff/workspace binds (both land on the sgpps edge).
func StaffReachableJobExistsSQL() string {
	return "SELECT EXISTS(" +
		"SELECT 1 FROM " + entityid.JobPhase + " jp" +
		" JOIN " + entityid.JobTask + " jt ON jt.job_phase_id = jp.id" +
		" JOIN " + entityid.Job + " jw ON jw.id = jp.job_id AND jw.workspace_id = $3" +
		" WHERE jt.assigned_to = $1 AND jp.job_id = $2" +
		" UNION " +
		"SELECT 1 FROM " + entityid.JobPhase + " jp2" +
		" JOIN " + entityid.JobTask + " jt2 ON jt2.job_phase_id = jp2.id" +
		" JOIN " + entityid.TaskOutcome + " t ON t.job_task_id = jt2.id" +
		" JOIN " + entityid.Job + " jw2 ON jw2.id = jp2.job_id AND jw2.workspace_id = $3" +
		" WHERE (t.recorded_by = $1 OR t.reviewed_by = $1) AND jp2.job_id = $2" +
		" UNION " +
		"SELECT 1 FROM " + entityid.Job + " jw3" +
		" JOIN " + entityid.SubscriptionSeat + " ss ON ss.subscription_id = jw3.origin_id" +
		" JOIN " + entityid.JobTemplate + " tpl ON tpl.id = jw3.job_template_id" +
		" JOIN " + entityid.ProductPlan + " pl ON pl.id = ss.product_plan_id AND pl.product_id = tpl.output_product_id" +
		" WHERE ss.staff_id = $1 AND ss.status = 'active' AND ss.active = true" +
		" AND jw3.origin_type = '" + originTypeSubscription + "'" +
		" AND jw3.id = $2 AND jw3.workspace_id = $3 AND ss.workspace_id = $3" +
		" UNION " +
		// Class-edge tier (mirrors reachableJobUnion's 4th branch): the staff is the
		// sgpps servicer for the job's cohort offering, matched to the job's
		// deliverable via product_plan.product_id == job.output_product_id. Both binds
		// land on the edge ($1 staff, $3 workspace), so an empty bind yields no row.
		"SELECT 1 FROM " + entityid.SubscriptionGroupProductPlanStaff + " e" +
		" JOIN " + entityid.SubscriptionGroupMember + " m ON m.subscription_group_id = e.subscription_group_id AND m.active" +
		" JOIN " + entityid.ProductPlan + " pp ON pp.id = e.product_plan_id" +
		" JOIN " + entityid.Job + " jce ON jce.origin_id = m.subscription_id AND jce.output_product_id = pp.product_id" +
		" WHERE e.staff_id = $1 AND e.active AND e.workspace_id = $3 AND jce.id = $2" +
		")"
}

// StaffReachableClientIDsSQL returns a query yielding the DISTINCT set of
// client.id values the acting staff.id reaches via the delivery graph or an
// active deliverable-matched subscription_seat, confined to one workspace. It is the
// row-set seam for the GENERIC dbOps.List path (ListClients) that has no WHERE
// seam: fetch the set once, then drop any listed row whose id is absent
// (fail-closed). Args order: $1 = staffID, $2 = workspaceID.
func StaffReachableClientIDsSQL() string {
	return reachableClientUnion(1, 2)
}

// StaffReachableJobIDsSQL is StaffReachableClientIDsSQL for job.id — the row-set
// seam for the generic ListJobs path. Args order: $1 = staffID, $2 = workspaceID.
func StaffReachableJobIDsSQL() string {
	return reachableJobUnion(1, 2)
}
