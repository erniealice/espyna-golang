//go:build mysql

package rbac

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MySQLPermissionQuery implements security.PermissionQuery using MySQL 8.0+
// RBAC tables.
//
// Dialect translation from postgres gold standard (permission_query.go):
//   - Placeholders: $1,$2,... → ? (positional, re-sequenced)
//   - Identifier quoting: "ident" → `ident` (backticks)
//   - Boolean literal: true → 1 (MySQL TINYINT(1))
//   - No postgres-specific syntax otherwise — CTEs and JOIN shapes are identical
//
// Binding-scoped grant resolution (A2 / WKR-P0-2 — 2026-05-24):
// Identical semantics to the postgres implementation. See that file for the
// full design rationale. The five grant chains are the same; only the
// placeholder token and quoting style differ.
//
// DENY-wins posture is preserved: a permission appears in the result only if
// there is at least one ALLOW grant AND zero DENY grants across the user's
// active roles in the selected binding within the given workspace.
type MySQLPermissionQuery struct {
	db *sql.DB
}

// NewMySQLPermissionQuery creates the MySQL-backed permission query service.
func NewMySQLPermissionQuery(db *sql.DB) *MySQLPermissionQuery {
	return &MySQLPermissionQuery{db: db}
}

var _ security.PermissionQuery = (*MySQLPermissionQuery)(nil)

// PrincipalType integer values mirror domain.entity.v1.PrincipalType.
// Re-declared locally so the adapter doesn't drag the proto package into
// the contrib module's import set.
const (
	principalTypeUnspecified      int32 = 0
	principalTypeOperatorOwner    int32 = 1
	principalTypeOperatorStaff    int32 = 2
	principalTypeClient           int32 = 3
	principalTypeClientDelegate   int32 = 4
	principalTypeSupplier         int32 = 5
	principalTypeSupplierDelegate int32 = 6
)

// userRolesUnionCTE is the legacy UNION-across-all-bindings CTE retained for
// the backwards-compatibility fall-back when (bindingKind, bindingID) is the
// EXACT zero pair. Production callers post-A2 always set the hint.
//
// Dialect: $1,$2 → ? (two positional args: userID, workspaceID).
// Boolean literal: active = true → active = 1 (MySQL TINYINT(1)).
const userRolesUnionCTE = `
	WITH user_roles AS (
		-- 1. WorkspaceUser → workspace_user_role (operator owner / staff)
		SELECT wur.role_id
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id
		WHERE wu.user_id = ?
		  AND wu.workspace_id = ?
		  AND wu.active = 1
		  AND wur.active = 1
		  AND wur.role_id IS NOT NULL

		UNION

		-- 2. ClientPortalGrant (CLIENT)
		SELECT cpg.role_id
		FROM client_portal_grant cpg
		WHERE cpg.user_id = ?
		  AND cpg.workspace_id = ?
		  AND cpg.active = 1
		  AND cpg.role_id IS NOT NULL

		UNION

		-- 3. SupplierPortalGrant (SUPPLIER)
		SELECT spg.role_id
		FROM supplier_portal_grant spg
		WHERE spg.user_id = ?
		  AND spg.workspace_id = ?
		  AND spg.active = 1
		  AND spg.role_id IS NOT NULL

		UNION

		-- 4. Delegate → DelegateClient (CLIENT_DELEGATE)
		SELECT dc.role_id
		FROM delegate d
		JOIN delegate_client dc ON dc.delegate_id = d.id
		LEFT JOIN client c ON c.id = dc.client_id AND c.active = 1
		WHERE d.user_id = ?
		  AND d.active = 1
		  AND dc.active = 1
		  AND COALESCE(dc.workspace_id, c.workspace_id) = ?
		  AND dc.role_id IS NOT NULL

		UNION

		-- 5. Delegate → DelegateSupplier (SUPPLIER_DELEGATE)
		SELECT ds.role_id
		FROM delegate d
		JOIN delegate_supplier ds ON ds.delegate_id = d.id
		LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = 1
		WHERE d.user_id = ?
		  AND d.active = 1
		  AND ds.active = 1
		  AND COALESCE(ds.workspace_id, s.workspace_id) = ?
		  AND ds.role_id IS NOT NULL
	)
`

// permissionSelect is the ALLOW-with-DENY-wins predicate appended after the
// per-binding user_roles CTE block.
//
// Dialect: no postgres-specific syntax here; PERMISSION_TYPE_ALLOW /
// PERMISSION_TYPE_DENY are string literals, not pg enums.
const permissionSelect = `
	SELECT DISTINCT p.permission_code
	FROM permission p
	JOIN role_permission rp ON rp.permission_id = p.id
	JOIN user_roles ur ON ur.role_id = rp.role_id
	WHERE rp.permission_type = 'PERMISSION_TYPE_ALLOW'
	  AND p.active = 1
	  AND rp.active = 1
	  AND p.permission_code NOT IN (
	      SELECT p2.permission_code
	      FROM permission p2
	      JOIN role_permission rp2 ON rp2.permission_id = p2.id
	      JOIN user_roles ur2 ON ur2.role_id = rp2.role_id
	      WHERE rp2.permission_type = 'PERMISSION_TYPE_DENY'
	        AND p2.active = 1
	        AND rp2.active = 1
	  )
`

// Per-binding user_roles CTEs — each is a SINGLE grant-chain SELECT.
// Parameter convention: ? = userID, ? = workspaceID, ? = bindingID
// (and ? = actingAsClientID / actingAsSupplierID for delegate kinds).
// MySQL uses positional ? so each CTE expects args in the order they appear.

const userRolesOperatorCTE = `
	WITH user_roles AS (
		SELECT wur.role_id
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id
		WHERE wu.id = ?
		  AND wu.user_id = ?
		  AND wu.workspace_id = ?
		  AND wu.active = 1
		  AND wur.active = 1
		  AND wur.role_id IS NOT NULL
	)
`

const userRolesClientCTE = `
	WITH user_roles AS (
		SELECT cpg.role_id
		FROM client_portal_grant cpg
		WHERE cpg.id = ?
		  AND cpg.user_id = ?
		  AND cpg.workspace_id = ?
		  AND cpg.active = 1
		  AND cpg.role_id IS NOT NULL
	)
`

const userRolesSupplierCTE = `
	WITH user_roles AS (
		SELECT spg.role_id
		FROM supplier_portal_grant spg
		WHERE spg.id = ?
		  AND spg.user_id = ?
		  AND spg.workspace_id = ?
		  AND spg.active = 1
		  AND spg.role_id IS NOT NULL
	)
`

// userRolesClientDelegateCTE — the role grant is on delegate_client, NOT on
// delegate. Args: bindingID, userID, workspaceID, actingAsClientID.
const userRolesClientDelegateCTE = `
	WITH user_roles AS (
		SELECT dc.role_id
		FROM delegate d
		JOIN delegate_client dc ON dc.delegate_id = d.id
		LEFT JOIN client c ON c.id = dc.client_id AND c.active = 1
		WHERE d.id = ?
		  AND dc.client_id = ?
		  AND d.user_id = ?
		  AND d.active = 1
		  AND dc.active = 1
		  AND COALESCE(dc.workspace_id, c.workspace_id) = ?
		  AND dc.role_id IS NOT NULL
	)
`

// userRolesSupplierDelegateCTE — symmetric to the client_delegate CTE.
// Args: bindingID, supplierID, userID, workspaceID.
const userRolesSupplierDelegateCTE = `
	WITH user_roles AS (
		SELECT ds.role_id
		FROM delegate d
		JOIN delegate_supplier ds ON ds.delegate_id = d.id
		LEFT JOIN supplier s ON s.id = ds.supplier_id AND s.active = 1
		WHERE d.id = ?
		  AND ds.supplier_id = ?
		  AND d.user_id = ?
		  AND d.active = 1
		  AND ds.active = 1
		  AND COALESCE(ds.workspace_id, s.workspace_id) = ?
		  AND ds.role_id IS NOT NULL
	)
`

// GetUserPermissionCodes returns all effective ALLOW codes for a user in a
// workspace, with DENY-wins applied. Empty slice when no permissions.
//
// Fail-closed semantics are identical to the postgres gold standard — see
// that file's docstring for the full dispatch rules.
func (q *MySQLPermissionQuery) GetUserPermissionCodes(
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

// buildPermissionQuerySQL is the pure SQL-builder helper.
//
// MySQL arg ordering for per-binding CTEs differs from postgres because MySQL
// uses positional ? without numbering — the CTE const strings above list args
// in the order: (bindingID, userID, workspaceID) for operator/client/supplier,
// and (bindingID, actingAsID, userID, workspaceID) for delegate kinds.
//
// The legacy union CTE repeats (userID, workspaceID) five times — one pair per
// grant chain — so it takes 10 args total rather than 2.
//
// Returns (stmt, args, ok). ok=false means the caller MUST fail closed.
func buildPermissionQuerySQL(
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) (string, []any, bool) {
	// Legacy union: EXACT zero quadruple only.
	if bindingKind == principalTypeUnspecified &&
		bindingID == "" &&
		actingAsClientID == "" &&
		actingAsSupplierID == "" {
		// Union CTE repeats (userID, workspaceID) once per chain: 5 chains × 2 = 10 args.
		args := []any{
			userID, workspaceID, // chain 1: workspace_user
			userID, workspaceID, // chain 2: client_portal_grant
			userID, workspaceID, // chain 3: supplier_portal_grant
			userID, workspaceID, // chain 4: delegate_client
			userID, workspaceID, // chain 5: delegate_supplier
		}
		return userRolesUnionCTE + permissionSelect, args, true
	}

	// Out-of-range bindingKind → fail closed.
	if !isKnownBindingKind(bindingKind) {
		return "", nil, false
	}

	// Partial hint → fail closed.
	if bindingID == "" {
		return "", nil, false
	}

	switch bindingKind {
	case principalTypeOperatorOwner, principalTypeOperatorStaff:
		// CTE arg order: bindingID, userID, workspaceID
		return userRolesOperatorCTE + permissionSelect,
			[]any{bindingID, userID, workspaceID},
			true
	case principalTypeClient:
		// CTE arg order: bindingID, userID, workspaceID
		return userRolesClientCTE + permissionSelect,
			[]any{bindingID, userID, workspaceID},
			true
	case principalTypeSupplier:
		// CTE arg order: bindingID, userID, workspaceID
		return userRolesSupplierCTE + permissionSelect,
			[]any{bindingID, userID, workspaceID},
			true
	case principalTypeClientDelegate:
		if actingAsClientID == "" {
			return "", nil, false
		}
		// CTE arg order: bindingID, actingAsClientID, userID, workspaceID
		return userRolesClientDelegateCTE + permissionSelect,
			[]any{bindingID, actingAsClientID, userID, workspaceID},
			true
	case principalTypeSupplierDelegate:
		if actingAsSupplierID == "" {
			return "", nil, false
		}
		// CTE arg order: bindingID, actingAsSupplierID, userID, workspaceID
		return userRolesSupplierDelegateCTE + permissionSelect,
			[]any{bindingID, actingAsSupplierID, userID, workspaceID},
			true
	default:
		return "", nil, false
	}
}

// isKnownBindingKind reports whether the integer value matches one of the six
// PrincipalType enumerants. UNSPECIFIED is NOT considered "known" here — the
// legacy union path is reached via the exact-zero-pair check above.
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
// PrincipalType integer. Returns ("", false) for UNSPECIFIED or out-of-range
// values. Exported for tests that pin CTE-shape invariants.
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
		return NewMySQLPermissionQuery(sqlDB)
	})
}
