//go:build postgresql

package ledger

import (
	"context"
	"fmt"

	equitydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/equity"
)

// SumContributedTotal returns the total positive contribution balance across all
// active equity accounts, in centavos. Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_equity_account_workspace_active
//	  ON equity_account(workspace_id, active);
func (r *PostgresEquityAccountRepository) SumContributedTotal(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(ea.balance), 0)::bigint
		FROM equity_account ea
		WHERE ea.active = true
		  AND ea.balance > 0
		  AND ($1::text IS NULL OR $1::text = '' OR ea.workspace_id = $1)`

	var total int64
	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&total)
	if err != nil {
		return 0, nil //nolint:nilerr // Graceful degradation when balance column is absent
	}
	return total, nil
}

// CountActive returns the number of active equity accounts. Workspace-scoped.
func (r *PostgresEquityAccountRepository) CountActive(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COUNT(*)::bigint
		FROM equity_account ea
		WHERE ea.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR ea.workspace_id = $1)`

	var n int64
	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&n)
	if err != nil {
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

// EquityAccountSlice is a tiny row shape used by the equity dashboard for the
// "top contributors" widget — keeps the dashboard query cheap and avoids
// dragging in the full proto for purely read-only display.
//
// **Aliased to the service-layer row type** so the postgres adapter
// directly satisfies [equitydash.EquityAccountDashboardRepository]. Go's
// interface satisfaction requires the *exact* named return type — without
// this alias the adapter's `TopContributors` would return its own local
// `ledger.EquityAccountSlice`, silently failing the type assertion in
// `internal/composition/core/initializers/service.go` and producing a nil
// `dashboardDeps.EquityAccount` at runtime. See Q-SDM-DASHBOARD-COMPILE-ASSERTIONS
// (LOCKED) and the §8 "Lesson learned" caveat in
// `docs/wiki/articles/hexagonal-rules.md`.
type EquityAccountSlice = equitydash.EquityAccountSlice

// TopContributors returns the active equity accounts with the highest positive
// balances. Workspace-scoped. Returned in centavos.
func (r *PostgresEquityAccountRepository) TopContributors(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]EquityAccountSlice, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			ea.id,
			ea.name,
			COALESCE(ea.owner_name, '') AS owner_name,
			ea.account_type,
			ea.balance
		FROM equity_account ea
		WHERE ea.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR ea.workspace_id = $1)
		ORDER BY ea.balance DESC NULLS LAST
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make([]EquityAccountSlice, 0, limit)
	for rows.Next() {
		var row EquityAccountSlice
		if scanErr := rows.Scan(&row.ID, &row.Name, &row.OwnerName, &row.AccountType, &row.Balance); scanErr != nil {
			return nil, fmt.Errorf("failed to scan top equity account row: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top equity account rows: %w", err)
	}
	return out, nil
}
