//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"fmt"

	admindash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/admin"
)

// Count returns the number of roles in the workspace.
//
// Workspace isolation: WHERE workspace_id = $1. Roles are workspace-scoped —
// each workspace has its own set of role definitions.
func (r *PostgresRoleRepository) Count(ctx context.Context, workspaceID string) (int64, error) {
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
		return 0, fmt.Errorf("failed to count roles: %w", err)
	}
	return n, nil
}

// RolePermissionCount is one row of the "roles by permission count" table
// widget on the admin dashboard.
//
// **Aliased to the service-layer row type** so the postgres adapter
// directly satisfies [admindash.RoleDashboardRepository]. Go's interface
// satisfaction requires the *exact* named return type — without this alias
// the adapter's `TopByPermissionCount` would return its own local
// `entity.RolePermissionCount`, silently failing the type assertion in
// `internal/composition/core/initializers/service.go` and producing a nil
// `dashboardDeps.AdminRole` at runtime. See Q-SDM-DASHBOARD-COMPILE-ASSERTIONS
// (LOCKED) and the §8 "Lesson learned" caveat in
// `docs/wiki/articles/hexagonal-rules.md`.
type RolePermissionCount = admindash.RolePermissionCount

// TopByPermissionCount returns the top-N roles ordered by their assigned
// permission count (descending).
//
// Workspace isolation: WHERE r.workspace_id = $1.
//
// The role_permission table is workspace-scoped via its associated role.
func (r *PostgresRoleRepository) TopByPermissionCount(ctx context.Context, workspaceID string, limit int32) ([]RolePermissionCount, error) {
	if limit <= 0 {
		limit = 5
	}
	query := fmt.Sprintf(`
		SELECT
			r.id,
			r.name,
			COALESCE(rp_count.cnt, 0) AS permission_count
		FROM %s r
		LEFT JOIN (
			SELECT role_id, COUNT(*) AS cnt
			FROM role_permission
			WHERE active = true
			GROUP BY role_id
		) rp_count ON rp_count.role_id = r.id
		WHERE ($1::text IS NULL OR $1::text = '' OR r.workspace_id = $1)
		ORDER BY permission_count DESC, r.name ASC
		LIMIT $2
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top roles by permission count: %w", err)
	}
	defer rows.Close()

	var out []RolePermissionCount
	for rows.Next() {
		var rec RolePermissionCount
		if err := rows.Scan(&rec.RoleID, &rec.RoleName, &rec.PermissionCount); err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role rows: %w", err)
	}
	return out, nil
}
