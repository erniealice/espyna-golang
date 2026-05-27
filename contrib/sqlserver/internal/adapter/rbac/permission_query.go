//go:build sqlserver

package rbac

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// SQLServerPermissionQuery implements security.PermissionQuery using SQL Server
// RBAC tables.
//
// Binding-scoped grant resolution mirrors the postgres gold standard
// (permission_query.go). All grant-chain semantics, fail-closed posture, and
// DENY-wins logic are identical — only the SQL dialect differs:
//
//   - Placeholders: $N → @pN
//   - Identifier quote: "x" → [x]  (bracket quoting)
//   - Boolean literal: true/false → 1/0  (BIT columns)
//   - ILIKE → LIKE  (SQL Server default CI collation is case-insensitive)
//   - No row-value comparison (a, b) < (x, y) in T-SQL — not needed here
//     because keyset pagination is not used in the permission query.
//
// Grant-chain selection (one row, never a union) — same logic as postgres:
//
//  1. PRINCIPAL_TYPE_OPERATOR_OWNER (1) / OPERATOR_STAFF (2) →
//     workspace_user (id = bindingID) → workspace_user_role.role_id
//  2. PRINCIPAL_TYPE_CLIENT (3) →
//     client_portal_grant (id = bindingID).role_id
//  3. PRINCIPAL_TYPE_SUPPLIER (5) →
//     supplier_portal_grant (id = bindingID).role_id
//  4. PRINCIPAL_TYPE_CLIENT_DELEGATE (4) →
//     delegate (id = bindingID) JOIN delegate_client
//     (delegate_id = bindingID AND client_id = actingAsClientID).role_id
//  5. PRINCIPAL_TYPE_SUPPLIER_DELEGATE (6) →
//     delegate (id = bindingID) JOIN delegate_supplier
//     (delegate_id = bindingID AND supplier_id = actingAsSupplierID).role_id
//
// Fail-closed: exact `(0, "")` zero pair → legacy union; all other partial /
// malformed / delegate-missing-target hints → empty result set.
type SQLServerPermissionQuery struct {
	db *sql.DB
}

// NewSQLServerPermissionQuery creates the SQL Server-backed permission query service.
func NewSQLServerPermissionQuery(db *sql.DB) *SQLServerPermissionQuery {
	return &SQLServerPermissionQuery{db: db}
}

var _ security.PermissionQuery = (*SQLServerPermissionQuery)(nil)

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
// EXACT zero pair. SQL Server translation: true → 1 (BIT), [brackets] for
// identifiers, @p1/@p2 placeholders.
const userRolesUnionCTE = `
	WITH [user_roles] AS (
		-- 1. WorkspaceUser → workspace_user_role (operator owner / staff)
		SELECT [wur].[role_id]
		FROM [workspace_user] [wu]
		JOIN [workspace_user_role] [wur] ON [wur].[workspace_user_id] = [wu].[id]
		WHERE [wu].[user_id] = @p1
		  AND [wu].[workspace_id] = @p2
		  AND [wu].[active] = 1
		  AND [wur].[active] = 1
		  AND [wur].[role_id] IS NOT NULL

		UNION

		-- 2. ClientPortalGrant (CLIENT)
		SELECT [cpg].[role_id]
		FROM [client_portal_grant] [cpg]
		WHERE [cpg].[user_id] = @p1
		  AND [cpg].[workspace_id] = @p2
		  AND [cpg].[active] = 1
		  AND [cpg].[role_id] IS NOT NULL

		UNION

		-- 3. SupplierPortalGrant (SUPPLIER)
		SELECT [spg].[role_id]
		FROM [supplier_portal_grant] [spg]
		WHERE [spg].[user_id] = @p1
		  AND [spg].[workspace_id] = @p2
		  AND [spg].[active] = 1
		  AND [spg].[role_id] IS NOT NULL

		UNION

		-- 4. Delegate → DelegateClient (CLIENT_DELEGATE)
		SELECT [dc].[role_id]
		FROM [delegate] [d]
		JOIN [delegate_client] [dc] ON [dc].[delegate_id] = [d].[id]
		LEFT JOIN [client] [c] ON [c].[id] = [dc].[client_id] AND [c].[active] = 1
		WHERE [d].[user_id] = @p1
		  AND [d].[active] = 1
		  AND [dc].[active] = 1
		  AND COALESCE([dc].[workspace_id], [c].[workspace_id]) = @p2
		  AND [dc].[role_id] IS NOT NULL

		UNION

		-- 5. Delegate → DelegateSupplier (SUPPLIER_DELEGATE)
		SELECT [ds].[role_id]
		FROM [delegate] [d]
		JOIN [delegate_supplier] [ds] ON [ds].[delegate_id] = [d].[id]
		LEFT JOIN [supplier] [s] ON [s].[id] = [ds].[supplier_id] AND [s].[active] = 1
		WHERE [d].[user_id] = @p1
		  AND [d].[active] = 1
		  AND [ds].[active] = 1
		  AND COALESCE([ds].[workspace_id], [s].[workspace_id]) = @p2
		  AND [ds].[role_id] IS NOT NULL
	)
`

// permissionSelect is the ALLOW-with-DENY-wins predicate appended after any
// user_roles CTE. SQL Server translation: DISTINCT is supported, @p placeholders.
const permissionSelect = `
	SELECT DISTINCT [p].[permission_code]
	FROM [permission] [p]
	JOIN [role_permission] [rp] ON [rp].[permission_id] = [p].[id]
	JOIN [user_roles] [ur] ON [ur].[role_id] = [rp].[role_id]
	WHERE [rp].[permission_type] = 'PERMISSION_TYPE_ALLOW'
	  AND [p].[active] = 1
	  AND [rp].[active] = 1
	  AND [p].[permission_code] NOT IN (
	      SELECT [p2].[permission_code]
	      FROM [permission] [p2]
	      JOIN [role_permission] [rp2] ON [rp2].[permission_id] = [p2].[id]
	      JOIN [user_roles] [ur2] ON [ur2].[role_id] = [rp2].[role_id]
	      WHERE [rp2].[permission_type] = 'PERMISSION_TYPE_DENY'
	        AND [p2].[active] = 1
	        AND [rp2].[active] = 1
	  )
`

// Per-binding user_roles CTEs — SQL Server translation of the postgres gold
// standard. @p1 = userID, @p2 = workspaceID, @p3 = bindingID,
// @p4 = actingAsClientID / actingAsSupplierID (delegate kinds only).

const userRolesOperatorCTE = `
	WITH [user_roles] AS (
		SELECT [wur].[role_id]
		FROM [workspace_user] [wu]
		JOIN [workspace_user_role] [wur] ON [wur].[workspace_user_id] = [wu].[id]
		WHERE [wu].[id] = @p3
		  AND [wu].[user_id] = @p1
		  AND [wu].[workspace_id] = @p2
		  AND [wu].[active] = 1
		  AND [wur].[active] = 1
		  AND [wur].[role_id] IS NOT NULL
	)
`

const userRolesClientCTE = `
	WITH [user_roles] AS (
		SELECT [cpg].[role_id]
		FROM [client_portal_grant] [cpg]
		WHERE [cpg].[id] = @p3
		  AND [cpg].[user_id] = @p1
		  AND [cpg].[workspace_id] = @p2
		  AND [cpg].[active] = 1
		  AND [cpg].[role_id] IS NOT NULL
	)
`

const userRolesSupplierCTE = `
	WITH [user_roles] AS (
		SELECT [spg].[role_id]
		FROM [supplier_portal_grant] [spg]
		WHERE [spg].[id] = @p3
		  AND [spg].[user_id] = @p1
		  AND [spg].[workspace_id] = @p2
		  AND [spg].[active] = 1
		  AND [spg].[role_id] IS NOT NULL
	)
`

// userRolesClientDelegateCTE scopes to the delegate_client row the delegate is
// currently acting through (codex A2-P0-1 equivalent for SQL Server).
const userRolesClientDelegateCTE = `
	WITH [user_roles] AS (
		SELECT [dc].[role_id]
		FROM [delegate] [d]
		JOIN [delegate_client] [dc] ON [dc].[delegate_id] = [d].[id]
		LEFT JOIN [client] [c] ON [c].[id] = [dc].[client_id] AND [c].[active] = 1
		WHERE [d].[id] = @p3
		  AND [dc].[client_id] = @p4
		  AND [d].[user_id] = @p1
		  AND [d].[active] = 1
		  AND [dc].[active] = 1
		  AND COALESCE([dc].[workspace_id], [c].[workspace_id]) = @p2
		  AND [dc].[role_id] IS NOT NULL
	)
`

// userRolesSupplierDelegateCTE — symmetric to the client_delegate CTE.
const userRolesSupplierDelegateCTE = `
	WITH [user_roles] AS (
		SELECT [ds].[role_id]
		FROM [delegate] [d]
		JOIN [delegate_supplier] [ds] ON [ds].[delegate_id] = [d].[id]
		LEFT JOIN [supplier] [s] ON [s].[id] = [ds].[supplier_id] AND [s].[active] = 1
		WHERE [d].[id] = @p3
		  AND [ds].[supplier_id] = @p4
		  AND [d].[user_id] = @p1
		  AND [d].[active] = 1
		  AND [ds].[active] = 1
		  AND COALESCE([ds].[workspace_id], [s].[workspace_id]) = @p2
		  AND [ds].[role_id] IS NOT NULL
	)
`

// GetUserPermissionCodes returns all effective ALLOW codes for a user in a
// workspace, with DENY-wins applied. Empty slice when no permissions.
//
// Fail-closed semantics are identical to the postgres gold standard:
// partial/malformed hints and delegate kinds without acting-as IDs return
// an empty set without executing any SQL.
func (q *SQLServerPermissionQuery) GetUserPermissionCodes(
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

// buildPermissionQuerySQL is the pure SQL-builder helper — identical dispatch
// logic to the postgres gold standard; only the CTE/select constants differ.
// Returns (stmt, args, ok). ok=false means fail-closed (empty permission set).
func buildPermissionQuerySQL(
	userID, workspaceID string,
	bindingKind int32,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) (string, []any, bool) {
	// Legacy union path: exact zero quadruple only.
	if bindingKind == principalTypeUnspecified &&
		bindingID == "" &&
		actingAsClientID == "" &&
		actingAsSupplierID == "" {
		return userRolesUnionCTE + permissionSelect,
			[]any{userID, workspaceID},
			true
	}

	// Out-of-range bindingKind → fail closed.
	if !isKnownBindingKind(bindingKind) {
		return "", nil, false
	}

	// Partial hint (kind set but no id) → fail closed.
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
		return "", nil, false
	}
}

// isKnownBindingKind reports whether the integer value matches one of the six
// PrincipalType enumerants. UNSPECIFIED is NOT considered known here — the
// legacy union path is reached only via the exact-zero-pair check above.
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

func init() {
	registry.RegisterPermissionQueryFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewSQLServerPermissionQuery(sqlDB)
	})
}
