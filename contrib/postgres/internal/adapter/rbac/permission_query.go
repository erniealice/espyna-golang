//go:build postgresql

package rbac

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// PostgresPermissionQuery implements security.PermissionQuery using PostgreSQL
// RBAC tables.
//
// Binding-scoped grant resolution (A2 / WKR-P0-2 — 2026-05-24):
// The query traverses ONE of the five grant chains, selected by the
// (bindingKind, bindingID) hint sourced from the session row. This closes
// the silent privilege-elevation hole where a user holding multiple
// bindings in the same workspace (e.g. CLIENT + OPERATOR_STAFF) would
// silently receive the UNION of permissions across all bindings.
//
// Delegate target scoping (A2-followup / codex A2-P0-1 — 2026-05-24):
// For CLIENT_DELEGATE and SUPPLIER_DELEGATE the parent delegate row is
// only the user-anchoring join — the actual role grant lives on the
// per-target delegate_client / delegate_supplier rows. The CTE now
// additionally filters by acting_as_client_id / acting_as_supplier_id so
// a delegate with N>1 targets only sees the permissions of the row they
// are currently acting through (matching the lock boundary at
// apps/service-admin/internal/composition/principal_switch.go's
// lockTargetBinding for delegate kinds).
//
// Grant-chain selection (one row, never a union):
//
//  1. PRINCIPAL_TYPE_OPERATOR_OWNER (1) / OPERATOR_STAFF (2) →
//     workspace_user (id = bindingID) → workspace_user_role.role_id
//  2. PRINCIPAL_TYPE_CLIENT (3) →
//     client_portal_grant (id = bindingID).role_id
//  3. PRINCIPAL_TYPE_SUPPLIER (5) →
//     supplier_portal_grant (id = bindingID).role_id
//  4. PRINCIPAL_TYPE_CLIENT_DELEGATE (4) →
//     delegate (id = bindingID) JOIN delegate_client
//       (delegate_id = bindingID AND client_id = actingAsClientID).role_id
//  5. PRINCIPAL_TYPE_SUPPLIER_DELEGATE (6) →
//     delegate (id = bindingID) JOIN delegate_supplier
//       (delegate_id = bindingID AND supplier_id = actingAsSupplierID).role_id
//
// All chains still join role_permission → permission and apply the same
// DENY-wins predicate that the union variant did.
//
// Fail-closed posture (codex A2-P1-1 fix — 2026-05-24): the legacy
// union-fallback path is reserved for the EXACT zero pair
// `(bindingKind=0, bindingID="")`. Any other combination — partial hints
// (CLIENT, ""), out-of-range bindingKinds, kind set with empty id, id
// set with UNSPECIFIED kind, or a delegate kind with no acting-as id —
// returns an empty permission set. Production callers MUST always supply
// a complete binding hint; only legacy bootstrap and test paths exercise
// the union path.
//
// DENY-wins: a permission appears in the result only if there is at
// least one ALLOW grant AND zero DENY grants across the user's active
// roles in the selected binding within the given workspace.
type PostgresPermissionQuery struct {
	db *sql.DB
}

// NewPostgresPermissionQuery creates the PG-backed permission query service.
func NewPostgresPermissionQuery(db *sql.DB) *PostgresPermissionQuery {
	return &PostgresPermissionQuery{db: db}
}

var _ security.PermissionQuery = (*PostgresPermissionQuery)(nil)

// PrincipalType integer values mirror domain.entity.v1.PrincipalType.
// Re-declared locally so the adapter doesn't drag the proto package into
// the contrib module's import set (the wire integer is the only thing the
// adapter needs).
const (
	principalTypeUnspecified      int32 = 0
	principalTypeOperatorOwner    int32 = 1
	principalTypeOperatorStaff    int32 = 2
	principalTypeClient           int32 = 3
	principalTypeClientDelegate   int32 = 4
	principalTypeSupplier         int32 = 5
	principalTypeSupplierDelegate int32 = 6
)

// userRolesUnionCTE is the legacy UNION-across-all-bindings CTE retained
// for the backwards-compatibility fall-back when (bindingKind, bindingID)
// is the EXACT zero pair (codex A2-P1-1 fix: only the exact zero pair
// hits this path; partial / malformed hints fail closed). Production
// callers post-A2 always set the hint and take the binding-scoped CTEs
// below instead.
const userRolesUnionCTE = `
	WITH user_roles AS (
		-- 1. WorkspaceUser → workspace_user_role (operator owner / staff)
		SELECT wur.role_id
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id
		WHERE wu.user_id = $1
		  AND wu.workspace_id = $2
		  AND wu.active = true
		  AND wur.active = true
		  AND wur.role_id IS NOT NULL

		UNION

		-- 2. ClientPortalGrant (CLIENT)
		SELECT cpg.role_id
		FROM client_portal_grant cpg
		WHERE cpg.user_id = $1
		  AND cpg.workspace_id = $2
		  AND cpg.active = true
		  AND cpg.role_id IS NOT NULL

		UNION

		-- 3. SupplierPortalGrant (SUPPLIER)
		SELECT spg.role_id
		FROM supplier_portal_grant spg
		WHERE spg.user_id = $1
		  AND spg.workspace_id = $2
		  AND spg.active = true
		  AND spg.role_id IS NOT NULL

		UNION

		-- 4. Delegate → DelegateClient (CLIENT_DELEGATE)
		SELECT dc.role_id
		FROM delegate d
		JOIN delegate_client dc ON dc.delegate_id = d.id
		LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
		WHERE d.user_id = $1
		  AND d.active = true
		  AND dc.active = true
		  AND COALESCE(dc.workspace_id, c.workspace_id) = $2
		  AND dc.role_id IS NOT NULL

		UNION

		-- 5. Delegate → DelegateSupplier (SUPPLIER_DELEGATE)
		SELECT ds.role_id
		FROM delegate d
		JOIN delegate_supplier ds ON ds.delegate_id = d.id
		LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
		WHERE d.user_id = $1
		  AND d.active = true
		  AND ds.active = true
		  AND COALESCE(ds.workspace_id, s.workspace_id) = $2
		  AND ds.role_id IS NOT NULL
	)
`

// permissionSelect is the ALLOW-with-DENY-wins predicate that every
// binding chain feeds. It's appended after the per-binding user_roles
// CTE block.
const permissionSelect = `
	SELECT DISTINCT p.permission_code
	FROM permission p
	JOIN role_permission rp ON rp.permission_id = p.id
	JOIN user_roles ur ON ur.role_id = rp.role_id
	WHERE rp.permission_type = 'PERMISSION_TYPE_ALLOW'
	  AND p.active = true
	  AND rp.active = true
	  AND p.permission_code NOT IN (
	      SELECT p2.permission_code
	      FROM permission p2
	      JOIN role_permission rp2 ON rp2.permission_id = p2.id
	      JOIN user_roles ur2 ON ur2.role_id = rp2.role_id
	      WHERE rp2.permission_type = 'PERMISSION_TYPE_DENY'
	        AND p2.active = true
	        AND rp2.active = true
	  )
`

// Per-binding user_roles CTEs — each is a SINGLE grant-chain SELECT with
// no UNIONs, so the user only sees the permissions of the binding the
// session row currently identifies. Parameter convention: $1 = userID,
// $2 = workspaceID, $3 = bindingID, $4 = actingAsClientID /
// actingAsSupplierID (delegate kinds only).

const userRolesOperatorCTE = `
	WITH user_roles AS (
		SELECT wur.role_id
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id
		WHERE wu.id = $3
		  AND wu.user_id = $1
		  AND wu.workspace_id = $2
		  AND wu.active = true
		  AND wur.active = true
		  AND wur.role_id IS NOT NULL
	)
`

const userRolesClientCTE = `
	WITH user_roles AS (
		SELECT cpg.role_id
		FROM client_portal_grant cpg
		WHERE cpg.id = $3
		  AND cpg.user_id = $1
		  AND cpg.workspace_id = $2
		  AND cpg.active = true
		  AND cpg.role_id IS NOT NULL
	)
`

const userRolesSupplierCTE = `
	WITH user_roles AS (
		SELECT spg.role_id
		FROM supplier_portal_grant spg
		WHERE spg.id = $3
		  AND spg.user_id = $1
		  AND spg.workspace_id = $2
		  AND spg.active = true
		  AND spg.role_id IS NOT NULL
	)
`

// userRolesClientDelegateCTE — the role grant is on delegate_client, NOT
// on delegate. The parent delegate row only anchors user_id/active; the
// per-target dc.client_id = $4 filter is what restricts the lookup to
// the row the delegate is currently acting through (codex A2-P0-1).
const userRolesClientDelegateCTE = `
	WITH user_roles AS (
		SELECT dc.role_id
		FROM delegate d
		JOIN delegate_client dc ON dc.delegate_id = d.id
		LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
		WHERE d.id = $3
		  AND dc.client_id = $4
		  AND d.user_id = $1
		  AND d.active = true
		  AND dc.active = true
		  AND COALESCE(dc.workspace_id, c.workspace_id) = $2
		  AND dc.role_id IS NOT NULL
	)
`

// userRolesSupplierDelegateCTE — symmetric to the client_delegate CTE.
// ds.supplier_id = $4 filter restricts to the per-target row the
// delegate is currently acting through.
const userRolesSupplierDelegateCTE = `
	WITH user_roles AS (
		SELECT ds.role_id
		FROM delegate d
		JOIN delegate_supplier ds ON ds.delegate_id = d.id
		LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
		WHERE d.id = $3
		  AND ds.supplier_id = $4
		  AND d.user_id = $1
		  AND d.active = true
		  AND ds.active = true
		  AND COALESCE(ds.workspace_id, s.workspace_id) = $2
		  AND ds.role_id IS NOT NULL
	)
`

// GetUserPermissionCodes returns all effective ALLOW codes for a user in a
// workspace, with DENY-wins applied. Empty slice when no permissions.
//
// When (bindingKind, bindingID) is non-zero/non-empty AND any required
// acting-as id for delegate kinds is supplied, only the matching grant
// chain contributes role IDs — closing the silent-elevation holes that
// the legacy UNION variant and the pre-followup delegate CTE allowed.
//
// Fail-closed paths (codex A2-P1-1 + A2-P0-1):
//   - exact `(0, "")` → legacy union (backwards-compat).
//   - partial hint (e.g. `(CLIENT, "")`, `(UNSPECIFIED, "cpg-1")`) →
//     empty result.
//   - out-of-range bindingKind → empty result.
//   - delegate kind with empty acting_as id → empty result.
func (q *PostgresPermissionQuery) GetUserPermissionCodes(
	ctx context.Context,
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) ([]string, error) {
	stmt, args, ok := buildPermissionQuerySQL(
		userID, workspaceID, bindingKind, bindingID,
		actingAsClientID, actingAsSupplierID,
	)
	if !ok {
		// Fail-closed: malformed/partial/invalid hint or delegate kind
		// missing the per-target acting-as id. Return empty (never
		// degrade to union for non-legacy callers).
		return []string{}, nil
	}

	rows, err := q.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("permission_query: %w", err)
	}
	defer rows.Close()

	codes := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("permission_query scan: %w", err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("permission_query rows: %w", err)
	}
	return codes, nil
}

// buildPermissionQuerySQL is the pure SQL-builder helper extracted so
// no-live-DB tests can assert the exact predicates each binding-hint
// shape produces (codex A2-P1-3). Returns (stmt, args, ok). ok=false
// means the caller MUST fail closed (return an empty permission set
// rather than executing any SQL).
//
// Dispatch rules:
//   - Exact legacy zero pair `(0, "")` → union CTE + 2 args.
//   - bindingKind ∈ {OPERATOR_OWNER, OPERATOR_STAFF, CLIENT, SUPPLIER}
//     AND bindingID != "" → per-kind scoped CTE + 3 args.
//   - bindingKind == CLIENT_DELEGATE AND bindingID != "" AND
//     actingAsClientID != "" → client-delegate scoped CTE + 4 args.
//   - bindingKind == SUPPLIER_DELEGATE AND bindingID != "" AND
//     actingAsSupplierID != "" → supplier-delegate scoped CTE + 4 args.
//   - All other shapes (partial hint, kind set with empty id, id set
//     with UNSPECIFIED kind, out-of-range kind, delegate kind with
//     empty acting-as id) → (nil, nil, false).
func buildPermissionQuerySQL(
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) (string, []any, bool) {
	// Reserve legacy union for EXACTLY the zero pair. Any other shape
	// must fail closed (codex A2-P1-1).
	if bindingKind == principalTypeUnspecified && bindingID == "" {
		return userRolesUnionCTE + permissionSelect,
			[]any{userID, workspaceID},
			true
	}

	// Out-of-range bindingKind → fail closed BEFORE switching on it.
	if !isKnownBindingKind(bindingKind) {
		return "", nil, false
	}

	// Partial hint (kind set but no id, or id set with UNSPECIFIED) →
	// fail closed. The UNSPECIFIED case is already screened above by
	// the zero-pair branch; this catches "kind != 0 && id == ''".
	if bindingID == "" {
		return "", nil, false
	}

	switch bindingKind {
	case principalTypeOperatorOwner, principalTypeOperatorStaff:
		return userRolesOperatorCTE + permissionSelect,
			[]any{userID, workspaceID, bindingID},
			true
	case principalTypeClient:
		return userRolesClientCTE + permissionSelect,
			[]any{userID, workspaceID, bindingID},
			true
	case principalTypeSupplier:
		return userRolesSupplierCTE + permissionSelect,
			[]any{userID, workspaceID, bindingID},
			true
	case principalTypeClientDelegate:
		if actingAsClientID == "" {
			// Delegate kind set but no per-target row → fail closed
			// (codex A2-P0-1: never union per-target grants).
			return "", nil, false
		}
		return userRolesClientDelegateCTE + permissionSelect,
			[]any{userID, workspaceID, bindingID, actingAsClientID},
			true
	case principalTypeSupplierDelegate:
		if actingAsSupplierID == "" {
			return "", nil, false
		}
		return userRolesSupplierDelegateCTE + permissionSelect,
			[]any{userID, workspaceID, bindingID, actingAsSupplierID},
			true
	default:
		// Defense-in-depth — isKnownBindingKind above already filtered
		// these; the switch covers every known case explicitly.
		return "", nil, false
	}
}

// isKnownBindingKind reports whether the integer value matches one of
// the six PrincipalType enumerants the adapter knows how to scope.
// UNSPECIFIED is NOT considered "known" here — the legacy union path is
// reached via the exact-zero-pair check in buildPermissionQuerySQL,
// not by falling through this guard.
func isKnownBindingKind(kind int32) bool {
	switch kind {
	case principalTypeOperatorOwner,
		principalTypeOperatorStaff,
		principalTypeClient,
		principalTypeClientDelegate,
		principalTypeSupplier,
		principalTypeSupplierDelegate:
		return true
	default:
		return false
	}
}

// bindingCTEForKind selects the per-binding user_roles CTE for the given
// PrincipalType integer. Returns ("", false) for UNSPECIFIED or
// out-of-range values so the caller can fail closed.
//
// Note: this exposes the per-kind CTE for inspection; the full SQL is
// assembled by buildPermissionQuerySQL which is the canonical entry
// point used by GetUserPermissionCodes. Kept exported (package-scope)
// for tests that pin the CTE-shape invariants.
func bindingCTEForKind(kind int32) (string, bool) {
	switch kind {
	case principalTypeOperatorOwner, principalTypeOperatorStaff:
		return userRolesOperatorCTE, true
	case principalTypeClient:
		return userRolesClientCTE, true
	case principalTypeSupplier:
		return userRolesSupplierCTE, true
	case principalTypeClientDelegate:
		return userRolesClientDelegateCTE, true
	case principalTypeSupplierDelegate:
		return userRolesSupplierDelegateCTE, true
	default:
		return "", false
	}
}

func init() {
	registry.RegisterPermissionQueryFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresPermissionQuery(sqlDB)
	})
}
