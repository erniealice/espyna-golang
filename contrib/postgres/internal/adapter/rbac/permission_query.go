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
// RBAC tables. Chain traversed:
//
//	workspace_user → workspace_user_role → role_permission → permission
//
// DENY-wins: a permission appears in the result only if there is at least
// one ALLOW grant AND zero DENY grants across the user's active roles in
// the given workspace.
type PostgresPermissionQuery struct {
	db *sql.DB
}

// NewPostgresPermissionQuery creates the PG-backed permission query service.
func NewPostgresPermissionQuery(db *sql.DB) *PostgresPermissionQuery {
	return &PostgresPermissionQuery{db: db}
}

var _ security.PermissionQuery = (*PostgresPermissionQuery)(nil)

// GetUserPermissionCodes returns all effective ALLOW codes for a user in a
// workspace, with DENY-wins applied. Empty slice when no permissions.
func (q *PostgresPermissionQuery) GetUserPermissionCodes(ctx context.Context, userID, workspaceID string) ([]string, error) {
	const stmt = `
		SELECT DISTINCT p.permission_code
		FROM permission p
		JOIN role_permission rp ON rp.permission_id = p.id
		JOIN workspace_user_role wur ON wur.role_id = rp.role_id
		JOIN workspace_user wu ON wu.id = wur.workspace_user_id
		WHERE wu.user_id = $1
		  AND wu.workspace_id = $2
		  AND rp.permission_type = 'PERMISSION_TYPE_ALLOW'
		  AND p.active = true
		  AND rp.active = true
		  AND wur.active = true
		  AND wu.active = true
		  AND p.permission_code NOT IN (
		      SELECT p2.permission_code
		      FROM permission p2
		      JOIN role_permission rp2 ON rp2.permission_id = p2.id
		      JOIN workspace_user_role wur2 ON wur2.role_id = rp2.role_id
		      JOIN workspace_user wu2 ON wu2.id = wur2.workspace_user_id
		      WHERE wu2.user_id = $1
		        AND wu2.workspace_id = $2
		        AND rp2.permission_type = 'PERMISSION_TYPE_DENY'
		        AND p2.active = true
		        AND rp2.active = true
		        AND wur2.active = true
		        AND wu2.active = true
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
