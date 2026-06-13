//go:build postgresql

package entity

import (
	"context"
	"fmt"

	homedash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/home"
)

// ListUsersByRoleID returns workspace users assigned to the given role.
// Joins workspace_user_role -> workspace_user -> user for display fields.
//
// Workspace isolation: filters on wu.workspace_id = $2. workspace_user_role
// has no direct workspace_id column — scoping is via the parent
// workspace_user join.
func (r *PostgresWorkspaceUserRoleRepository) ListUsersByRoleID(ctx context.Context, workspaceID, roleID string) ([]homedash.UserByRole, error) {
	query := fmt.Sprintf(`
		SELECT wur.id, wu.id, wu.user_id,
		       COALESCE(u.first_name || ' ' || u.last_name, u.email_address) as user_name,
		       COALESCE(u.email_address, '') as email,
		       COALESCE(TO_CHAR(wur.date_created, 'Mon DD, YYYY'), '') as date_assigned
		FROM %s wur
		JOIN workspace_user wu ON wur.workspace_user_id = wu.id
		JOIN "user" u ON wu.user_id = u.id
		WHERE wur.role_id = $1 AND wur.active = true AND wu.workspace_id = $2
		ORDER BY u.first_name, u.last_name
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, roleID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list users by role: %w", err)
	}
	defer rows.Close()

	var users []homedash.UserByRole
	for rows.Next() {
		var u homedash.UserByRole
		if err := rows.Scan(&u.WorkspaceUserRoleID, &u.WorkspaceUserID, &u.UserID, &u.UserName, &u.Email, &u.DateAssigned); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
