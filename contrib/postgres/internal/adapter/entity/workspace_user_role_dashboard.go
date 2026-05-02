//go:build postgresql

package entity

import (
	"context"
	"fmt"
	"time"

	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// RecentAssignmentRow is one row of the "recent role changes" widget on
// the admin dashboard. It includes denormalized workspace_user_id +
// role_id for the linked record, plus user_email + role_name to keep the
// view layer free of further round trips.
type RecentAssignmentRow struct {
	ID              string
	WorkspaceUserID string
	UserEmail       string
	UserFullName    string
	RoleID          string
	RoleName        string
	DateCreated     *time.Time
}

// RecentAssignments returns the most recent (active) role assignments
// across the workspace, joined to workspace_user, user, and role for
// display-friendly fields.
//
// Workspace isolation: workspace_user_role has no direct workspace_id
// column, so we filter via the joined workspace_user.workspace_id.
func (r *PostgresWorkspaceUserRoleRepository) RecentAssignments(ctx context.Context, workspaceID string, limit int32) ([]*workspaceuserrolepb.WorkspaceUserRole, error) {
	if limit <= 0 {
		limit = 5
	}

	query := fmt.Sprintf(`
		SELECT
			wur.id,
			wur.workspace_user_id,
			wur.role_id,
			wur.active,
			wur.date_created,
			wur.date_modified
		FROM %s wur
		JOIN workspace_user wu ON wu.id = wur.workspace_user_id
		WHERE ($1::text IS NULL OR $1::text = '' OR wu.workspace_id = $1)
		  AND wur.active = true
		ORDER BY wur.date_created DESC NULLS LAST
		LIMIT $2
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent assignments: %w", err)
	}
	defer rows.Close()

	var out []*workspaceuserrolepb.WorkspaceUserRole
	for rows.Next() {
		var (
			id              string
			workspaceUserID string
			roleID          string
			active          bool
			dateCreated     *time.Time
			dateModified    *time.Time
		)
		if err := rows.Scan(&id, &workspaceUserID, &roleID, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("failed to scan assignment row: %w", err)
		}

		assignment := &workspaceuserrolepb.WorkspaceUserRole{
			Id:              id,
			WorkspaceUserId: workspaceUserID,
			RoleId:          roleID,
			Active:          active,
		}
		if dateCreated != nil && !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			assignment.DateCreated = &ts
			s := dateCreated.Format(time.RFC3339)
			assignment.DateCreatedString = &s
		}
		if dateModified != nil && !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			assignment.DateModified = &ts
			s := dateModified.Format(time.RFC3339)
			assignment.DateModifiedString = &s
		}
		out = append(out, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignment rows: %w", err)
	}
	return out, nil
}

// CountSinceDays returns the number of (active) workspace_user_role rows
// created within the last N days for the workspace. Used for the "Recent
// Role Changes (7d)" stat on the admin dashboard.
//
// Workspace isolation: filters wu.workspace_id = $1.
func (r *PostgresWorkspaceUserRoleRepository) CountSinceDays(ctx context.Context, workspaceID string, days int32) (int64, error) {
	if days <= 0 {
		days = 7
	}
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s wur
		JOIN workspace_user wu ON wu.id = wur.workspace_user_id
		WHERE ($1::text IS NULL OR $1::text = '' OR wu.workspace_id = $1)
		  AND wur.active = true
		  AND wur.date_created >= NOW() - ($2 || ' days')::interval
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, workspaceID, fmt.Sprintf("%d", days))
	var n int64
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to count recent assignments: %w", err)
	}
	return n, nil
}
