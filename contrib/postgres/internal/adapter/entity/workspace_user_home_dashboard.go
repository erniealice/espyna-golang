//go:build postgresql

package entity

import (
	"context"
	"fmt"
	"log"
	"time"

	homedash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/home"
)

// HomeDashboardStats returns total, active, inactive workspace-user counts
// and total active roles for the workspace.
//
// Workspace isolation: all four subqueries filter on workspace_id = $1.
// The role count queries the `role` table directly — acceptable for a
// dashboard aggregate that naturally crosses entity boundaries (same pattern
// as UsersPerRole which joins workspace_user_role).
func (r *PostgresWorkspaceUserRepository) HomeDashboardStats(ctx context.Context, workspaceID string) (homedash.HomeDashboardStats, error) {
	query := `SELECT
		COALESCE((SELECT COUNT(*) FROM workspace_user WHERE workspace_id = $1), 0),
		COALESCE((SELECT COUNT(*) FROM workspace_user WHERE active = true AND workspace_id = $1), 0),
		COALESCE((SELECT COUNT(*) FROM workspace_user WHERE active = false AND workspace_id = $1), 0),
		COALESCE((SELECT COUNT(*) FROM role WHERE active = true AND workspace_id = $1), 0)`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, workspaceID)

	var stats homedash.HomeDashboardStats
	if err := row.Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.InactiveUsers, &stats.TotalRoles); err != nil {
		log.Printf("home dashboard stats query error: %v", err)
		return homedash.HomeDashboardStats{}, nil
	}
	return stats, nil
}

// HomeRecentActivity returns the most recent user/role activity rows,
// synthesized from workspace_user and role created/modified timestamps.
//
// Workspace isolation: both subqueries filter on workspace_id = $1.
func (r *PostgresWorkspaceUserRepository) HomeRecentActivity(ctx context.Context, workspaceID string, limit int32) ([]homedash.ActivityRow, error) {
	if limit <= 0 {
		limit = 5
	}
	query := fmt.Sprintf(`
		(SELECT 'user_created' as event_type, u.first_name || ' ' || u.last_name as name, wu.date_created as event_date
		 FROM workspace_user wu JOIN "user" u ON wu.user_id = u.id
		 WHERE wu.date_created IS NOT NULL AND wu.workspace_id = $1
		 ORDER BY wu.date_created DESC LIMIT 3)
		UNION ALL
		(SELECT 'role_modified' as event_type, r.name, r.date_modified as event_date
		 FROM role r
		 WHERE r.date_modified IS NOT NULL AND r.workspace_id = $1
		 ORDER BY r.date_modified DESC LIMIT 2)
		ORDER BY event_date DESC
		LIMIT %d
	`, limit)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
	if err != nil {
		log.Printf("home dashboard activity query error: %v", err)
		return nil, nil
	}
	defer rows.Close()

	var items []homedash.ActivityRow
	for rows.Next() {
		var eventType, name string
		var eventDate time.Time
		if err := rows.Scan(&eventType, &name, &eventDate); err != nil {
			continue
		}
		items = append(items, homedash.ActivityRow{
			EventType: eventType,
			Name:      name,
			EventDate: eventDate,
		})
	}
	return items, rows.Err()
}

// HomeUserCreationsPerMonth returns per-month workspace-user creation counts
// for the last N months.
//
// Workspace isolation: filters on workspace_id = $1.
func (r *PostgresWorkspaceUserRepository) HomeUserCreationsPerMonth(ctx context.Context, workspaceID string, months int32) ([]homedash.MonthCount, error) {
	if months <= 0 {
		months = 12
	}
	query := fmt.Sprintf(`
		SELECT TO_CHAR(date_trunc('month', wu.date_created), 'Mon') as month_label,
		       COUNT(*) as user_count
		FROM %s wu
		WHERE wu.date_created >= NOW() - INTERVAL '%d months'
		  AND wu.workspace_id = $1
		GROUP BY date_trunc('month', wu.date_created)
		ORDER BY date_trunc('month', wu.date_created)
	`, r.tableName, months)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID)
	if err != nil {
		log.Printf("home dashboard chart query error: %v", err)
		return nil, nil
	}
	defer rows.Close()

	var result []homedash.MonthCount
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			continue
		}
		result = append(result, homedash.MonthCount{Label: label, Count: count})
	}
	return result, rows.Err()
}
