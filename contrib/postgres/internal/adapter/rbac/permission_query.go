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
// Grant-chain selection (one row, never a union):
//
//  1. PRINCIPAL_TYPE_OPERATOR_OWNER (1) / OPERATOR_STAFF (2) →
//     workspace_user (id = bindingID) → workspace_user_role.role_id
//  2. PRINCIPAL_TYPE_CLIENT (3) →
//     client_portal_grant (id = bindingID).role_id
//  3. PRINCIPAL_TYPE_SUPPLIER (5) →
//     supplier_portal_grant (id = bindingID).role_id
//  4. PRINCIPAL_TYPE_CLIENT_DELEGATE (4) →
//     delegate (id = bindingID) → delegate_client.role_id
//  5. PRINCIPAL_TYPE_SUPPLIER_DELEGATE (6) →
//     delegate (id = bindingID) → delegate_supplier.role_id
//
// All chains still join role_permission → permission and apply the same
// DENY-wins predicate that the union variant did.
//
// Backwards-compatibility fall-back: when (bindingKind, bindingID) is
// (0, "") — the proto zero values — the adapter preserves the historical
// union-across-all-bindings behaviour. Production callers MUST supply
// both; only the workspace_user-only and degraded test paths should
// land here, and they'll receive the same (looser) permissions they
// always did. The use case's request_required guard ensures we never
// reach this path on a real RBAC lookup with a nil request.
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
// is the zero pair. Production callers post-A2 always set the hint and
// take the binding-scoped CTEs below instead.
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
// $2 = workspaceID, $3 = bindingID.

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

const userRolesClientDelegateCTE = `
	WITH user_roles AS (
		SELECT dc.role_id
		FROM delegate d
		JOIN delegate_client dc ON dc.delegate_id = d.id
		LEFT JOIN client c ON c.id = dc.client_id AND c.active = true
		WHERE d.id = $3
		  AND d.user_id = $1
		  AND d.active = true
		  AND dc.active = true
		  AND COALESCE(dc.workspace_id, c.workspace_id) = $2
		  AND dc.role_id IS NOT NULL
	)
`

const userRolesSupplierDelegateCTE = `
	WITH user_roles AS (
		SELECT ds.role_id
		FROM delegate d
		JOIN delegate_supplier ds ON ds.delegate_id = d.id
		LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = true
		WHERE d.id = $3
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
// When (bindingKind, bindingID) is non-zero/non-empty, only the matching
// grant chain contributes role IDs — closing the silent-elevation hole
// that the legacy UNION variant allowed. When both are zero values the
// adapter degrades to the legacy UNION behaviour for backwards
// compatibility (see PostgresPermissionQuery doc).
func (q *PostgresPermissionQuery) GetUserPermissionCodes(
	ctx context.Context,
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
) ([]string, error) {
	var (
		stmt string
		args []any
	)
	if bindingKind != principalTypeUnspecified && bindingID != "" {
		cte, ok := bindingCTEForKind(bindingKind)
		if !ok {
			// Unknown kind ⇒ no roles can match. Return empty (rather
			// than degrading to the union) so an out-of-range hint
			// fails closed.
			return []string{}, nil
		}
		stmt = cte + permissionSelect
		args = []any{userID, workspaceID, bindingID}
	} else {
		// Backwards-compatibility fall-back — legacy union across all
		// bindings (matches pre-A2 behaviour).
		stmt = userRolesUnionCTE + permissionSelect
		args = []any{userID, workspaceID}
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

// bindingCTEForKind selects the per-binding user_roles CTE for the given
// PrincipalType integer. Returns ("", false) for UNSPECIFIED or
// out-of-range values so the caller can fail closed.
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
