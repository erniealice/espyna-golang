//go:build postgresql

package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// protoGradingToMap marshals a proto message to a JSON-shaped map[string]any for the
// generic dbOps write path (used by the education-grading R5 adapters).
func protoGradingToMap(msg proto.Message) (map[string]any, error) {
	jsonData, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	return data, nil
}

// principalTypeStaff is the integer value of the esqyma
// domain.entity.v1.PrincipalType enum member PRINCIPAL_TYPE_STAFF. When the
// session's active binding carries this kind, the principal_id IS the acting
// staff.id and the operational reads below must be confined to that staff
// member's own rows (Phase 4 row-scope; mirrors the delegate IDOR junction
// scope in contrib/postgres/internal/adapter/entity/delegate.go).
const principalTypeStaff int32 = 7

// staffRowScope reports whether the active session principal is a STAFF
// principal and, if so, the staff.id its operational reads must be confined to.
//
// The scope value is identity.PrincipalID — the acting staff.id taken from the
// SESSION binding (stamped by the session middleware), NEVER a request param.
// applies==false for every non-staff principal (operator/client/supplier/
// delegate, a pre-selection session with no resolved binding, or a
// service-to-service / CLI context with no identity at all) — those callers
// leave their query unchanged, preserving all existing workspace/client scoping.
//
// FAIL-CLOSED: a STAFF principal with an empty PrincipalID (a malformed session)
// returns staffID=="" with applies==true; staffScopeClause turns that into an
// always-false predicate, so such a session sees zero rows rather than falling
// through to an unscoped read.
func staffRowScope(ctx context.Context) (staffID string, applies bool) {
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil {
		return "", false
	}
	if id.PrincipalType != principalTypeStaff {
		return "", false
	}
	return id.PrincipalID, true
}

// staffScopeClause returns a SQL predicate fragment that confines a read to the
// active STAFF principal's own rows on the given staff.id column, together with
// the positional bind args to append to the query in order.
//
//   - non-staff principal  → ("", nil): the caller's query is unchanged.
//   - staff, empty staff.id → (" AND 1=0", nil): fail-closed, zero rows.
//   - staff, real staff.id  → (" AND <col> = $<nextParam>", []any{staffID}).
//
// col is always a hardcoded, adapter-controlled column expression (never
// request-derived), so there is no injection surface. nextParam is the 1-based
// index of the NEXT positional placeholder (i.e. existing-arg-count + 1); the
// caller appends the returned args after its existing args so the indexes line up.
func staffScopeClause(ctx context.Context, col string, nextParam int) (clause string, args []any) {
	staffID, applies := staffRowScope(ctx)
	if !applies {
		return "", nil
	}
	if staffID == "" {
		return " AND 1=0", nil
	}
	return fmt.Sprintf(" AND %s = $%d", col, nextParam), []any{staffID}
}

// staffScopeClauseAny is staffScopeClause for rows that carry the staff.id on
// more than one authorship axis (e.g. task_outcome, which a STAFF principal may
// own as either recorded_by OR reviewed_by). The single session staff.id is
// bound once and reused across every column via the shared $<nextParam>
// placeholder; the predicate matches when ANY column equals it. Same
// non-staff-unchanged / empty-id-fail-closed contract as staffScopeClause. cols
// are all hardcoded, adapter-controlled column expressions (no injection).
func staffScopeClauseAny(ctx context.Context, cols []string, nextParam int) (clause string, args []any) {
	staffID, applies := staffRowScope(ctx)
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
