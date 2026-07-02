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

	"github.com/erniealice/espyna-golang/shared/identity"
)

// PrincipalTypeStaff is the integer value of the esqyma
// domain.entity.v1.PrincipalType member PRINCIPAL_TYPE_STAFF. When the session's
// active binding carries this kind, the principal_id IS the acting staff.id and
// operational reads must be confined to that staff member's own rows.
const PrincipalTypeStaff int32 = 7

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
// client_id of any job the staff is assigned to OR has recorded/reviewed an outcome
// for, via the GENERIC operational-delivery graph (job_task.assigned_to /
// task_outcome.recorded_by|reviewed_by → job_phase → job.client_id). Same 3-state
// contract as StaffScopeClause; emits an IN(<graph subquery>) because clients carry
// no staff column. The reviewed_by axis keeps parity with the task_outcome adapter.
func StaffReachableClientClause(ctx context.Context, clientAlias string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	sub := fmt.Sprintf(`(
		SELECT j.client_id FROM job j
			JOIN job_phase jp ON jp.job_id = j.id
			JOIN job_task jt ON jt.job_phase_id = jp.id
		WHERE jt.assigned_to = $%d
		UNION
		SELECT j2.client_id FROM job j2
			JOIN job_phase jp2 ON jp2.job_id = j2.id
			JOIN job_task jt2 ON jt2.job_phase_id = jp2.id
			JOIN task_outcome t ON t.job_task_id = jt2.id
		WHERE t.recorded_by = $%d OR t.reviewed_by = $%d
	)`, nextParam, nextParam, nextParam)
	return fmt.Sprintf(" AND %s.id IN %s", clientAlias, sub), []any{staffID}
}

// StaffReachableJobClause confines a JOB query (the job row aliased as jobAlias,
// e.g. "j") to the jobs the active STAFF principal is assigned to OR has
// recorded/reviewed an outcome for. Same 3-state contract; graph-derived through
// job_phase/job_task/task_outcome.
func StaffReachableJobClause(ctx context.Context, jobAlias string, nextParam int) (clause string, args []any) {
	staffID, applies := StaffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	sub := fmt.Sprintf(`(
		SELECT jp.job_id FROM job_phase jp
			JOIN job_task jt ON jt.job_phase_id = jp.id
		WHERE jt.assigned_to = $%d
		UNION
		SELECT jp2.job_id FROM job_phase jp2
			JOIN job_task jt2 ON jt2.job_phase_id = jp2.id
			JOIN task_outcome t ON t.job_task_id = jt2.id
		WHERE t.recorded_by = $%d OR t.reviewed_by = $%d
	)`, nextParam, nextParam, nextParam)
	return fmt.Sprintf(" AND %s.id IN %s", jobAlias, sub), []any{staffID}
}

// StaffReachableClientExistsSQL returns a query that yields a single bool: whether
// the given staff.id can reach the given client.id via the delivery graph. It is
// for the generic dbOps by-id Read paths (ReadClient / item views) that have no
// WHERE seam — run it (after StaffRowScope reports a staff principal) and treat a
// false / no-row result as not-found, so an out-of-scope client is indistinguishable
// from a missing one. Args order: $1 = staffID, $2 = clientID.
func StaffReachableClientExistsSQL() string {
	return `SELECT EXISTS(
		SELECT 1 FROM job j
			JOIN job_phase jp ON jp.job_id = j.id
			JOIN job_task jt ON jt.job_phase_id = jp.id
		WHERE jt.assigned_to = $1 AND j.client_id = $2
		UNION
		SELECT 1 FROM job j2
			JOIN job_phase jp2 ON jp2.job_id = j2.id
			JOIN job_task jt2 ON jt2.job_phase_id = jp2.id
			JOIN task_outcome t ON t.job_task_id = jt2.id
		WHERE (t.recorded_by = $1 OR t.reviewed_by = $1) AND j2.client_id = $2
	)`
}

// StaffReachableJobExistsSQL is StaffReachableClientExistsSQL for a job.id (the
// generic dbOps by-id ReadJob path). Args order: $1 = staffID, $2 = jobID.
func StaffReachableJobExistsSQL() string {
	return `SELECT EXISTS(
		SELECT 1 FROM job_phase jp
			JOIN job_task jt ON jt.job_phase_id = jp.id
		WHERE jt.assigned_to = $1 AND jp.job_id = $2
		UNION
		SELECT 1 FROM job_phase jp2
			JOIN job_task jt2 ON jt2.job_phase_id = jp2.id
			JOIN task_outcome t ON t.job_task_id = jt2.id
		WHERE (t.recorded_by = $1 OR t.reviewed_by = $1) AND jp2.job_id = $2
	)`
}
