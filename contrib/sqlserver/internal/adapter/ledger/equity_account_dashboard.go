//go:build sqlserver

package ledger

import (
	"context"
	"fmt"

	equitydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/equity"
)

// EquityAccountSlice is aliased to the service-layer type so the adapter satisfies
// [equitydash.EquityAccountDashboardRepository] (identical pattern to the postgres gold standard).
type EquityAccountSlice = equitydash.EquityAccountSlice

// SumContributedTotal returns total positive equity account balance in centavos. Workspace-scoped.
//
// SQL Server differences: @p1; active = 1; CAST(… AS bigint) instead of ::bigint.
func (r *SQLServerEquityAccountRepository) SumContributedTotal(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT CAST(COALESCE(SUM(ea.balance), 0) AS bigint)
		FROM equity_account ea
		WHERE ea.active = 1
		  AND ea.balance > 0
		  AND (@p1 IS NULL OR @p1 = '' OR ea.workspace_id = @p1)`

	var total int64
	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&total)
	if err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// CountActive returns the number of active equity accounts. Workspace-scoped.
func (r *SQLServerEquityAccountRepository) CountActive(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT CAST(COUNT(*) AS bigint)
		FROM equity_account ea
		WHERE ea.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR ea.workspace_id = @p1)`

	var n int64
	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&n)
	if err != nil {
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

// TopContributors returns the active equity accounts with the highest balances.
// Workspace-scoped. SQL Server differences: @pN; OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY
// (requires ORDER BY; NULLS LAST emulated via CASE).
func (r *SQLServerEquityAccountRepository) TopContributors(
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
		WHERE ea.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR ea.workspace_id = @p1)
		ORDER BY
			CASE WHEN ea.balance IS NULL THEN 1 ELSE 0 END,
			ea.balance DESC
		OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY`

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
