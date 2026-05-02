//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"fmt"
)

// Count returns the total number of permissions in the system.
//
// Permission is a system-level entity (no workspace_id column) — confirmed
// by neighboring code in permission.go which has no workspace filter on any
// query. Per Phase 4b of the dashboards plan, this aggregate is intentionally
// global; the dashboard surfaces "all configured permissions" rather than a
// per-workspace projection.
func (r *PostgresPermissionRepository) Count(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, r.tableName)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query)
	var n int64
	if err := row.Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to count permissions: %w", err)
	}
	return n, nil
}
