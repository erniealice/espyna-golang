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
// As of Phase P10b of docs/plan/20260521-workspace-keyed-routing/ (Q-WS-10
// → A — unify the permission catalog), the query traverses ALL FIVE grant
// chains and UNIONs the role IDs the user holds in the workspace, regardless
// of principal type:
//
//  1. workspace_user → workspace_user_role.role_id   (OPERATOR_OWNER / STAFF)
//  2. client_portal_grant.role_id                    (CLIENT)
//  3. supplier_portal_grant.role_id                  (SUPPLIER)
//  4. delegate → delegate_client.role_id             (CLIENT_DELEGATE)
//  5. delegate → delegate_supplier.role_id           (SUPPLIER_DELEGATE)
//
// Role IDs from any of these chains feed the same role_permission → permission
// join. The applicable_principal_types column on `permission` is the data-side
// counterpart that constrains which principal types may receive a given code
// (validation enforced at grant-assignment time — separate change). Filtering
// the returned codes by the caller's principal_type happens at the upcoming
// repository-ownership-filter step (deferred from this phase).
//
// DENY-wins: a permission appears in the result only if there is at least
// one ALLOW grant AND zero DENY grants across all the user's active roles in
// the given workspace.
type PostgresPermissionQuery struct {
	db *sql.DB
}

// NewPostgresPermissionQuery creates the PG-backed permission query service.
func NewPostgresPermissionQuery(db *sql.DB) *PostgresPermissionQuery {
	return &PostgresPermissionQuery{db: db}
}

var _ security.PermissionQuery = (*PostgresPermissionQuery)(nil)

// userRolesCTE collects every active role_id the user holds in the workspace
// across the five grant chains. Re-used twice in GetUserPermissionCodes (once
// for the ALLOW set, once for the DENY-set NOT-IN exclusion).
const userRolesCTE = `
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

// GetUserPermissionCodes returns all effective ALLOW codes for a user in a
// workspace, with DENY-wins applied. Empty slice when no permissions.
func (q *PostgresPermissionQuery) GetUserPermissionCodes(ctx context.Context, userID, workspaceID string) ([]string, error) {
	const stmt = userRolesCTE + `
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
	rows, err := q.db.QueryContext(ctx, stmt, userID, workspaceID)
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

func init() {
	registry.RegisterPermissionQueryFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresPermissionQuery(sqlDB)
	})
}
