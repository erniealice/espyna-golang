//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"fmt"
)

// CountByWorkspace returns the number of workspace_user rows in the
// workspace.
//
// Workspace isolation: WHERE workspace_id = $1. Confirmed by
// PostgresWorkspaceUserRepository's existing list/get queries which filter
// on workspace_id.
func (r *PostgresWorkspaceUserRepository) CountByWorkspace(ctx context.Context, workspaceID string) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE ($1::text IS NULL OR $1::text = '' OR workspace_id = $1)
	`, r.tableName)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, workspaceID)
	var n int64
	if err := row.Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to count workspace_user: %w", err)
	}
	return n, nil
}

// UsersPerRole returns the number of (active) workspace_user rows assigned
// to each role, joined via the workspace_user_role link table.
//
// Workspace isolation: filters wu.workspace_id = $1.
//
// Returned map: role_id → count.
func (r *PostgresWorkspaceUserRepository) UsersPerRole(ctx context.Context, workspaceID string) (map[string]int64, error) {
	query := fmt.Sprintf(`
		SELECT
			wur.role_id,
			COUNT(DISTINCT wu.id) AS user_count
		FROM %s wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id
		WHERE ($1::text IS NULL OR $1::text = '' OR wu.workspace_id = $1)
		  AND wur.active = true
		GROUP BY wur.role_id
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users per role: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var roleID string
		var n int64
		if err := rows.Scan(&roleID, &n); err != nil {
			return nil, fmt.Errorf("failed to scan users-per-role row: %w", err)
		}
		out[roleID] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users-per-role rows: %w", err)
	}
	return out, nil
}
