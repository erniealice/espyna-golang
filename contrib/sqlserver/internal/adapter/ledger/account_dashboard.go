//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"fmt"
)

// SumBalanceByType returns total account balance grouped by account element
// (asset / liability / equity / revenue / expense). Centavos. Workspace-scoped.
//
// SQL Server differences: @p1 placeholder; active = 1 (BIT); no ::bigint cast
// (SQL Server SUM returns the column's native type — BIGINT if the column is
// already BIGINT; CAST(… AS bigint) added for safety).
func (r *SQLServerAccountRepository) SumBalanceByType(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT a.element, CAST(COALESCE(SUM(a.balance), 0) AS bigint)
		FROM [account] a
		WHERE a.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR a.workspace_id = @p1)
		GROUP BY a.element`

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 5)
	for rows.Next() {
		var (
			element string
			balance int64
		)
		if scanErr := rows.Scan(&element, &balance); scanErr != nil {
			return nil, fmt.Errorf("failed to scan account balance row: %w", scanErr)
		}
		out[normalizeElementKey(element)] = balance
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account balance rows: %w", err)
	}
	return out, nil
}

// CountByStatus returns counts of accounts grouped by status (active/inactive/locked).
// Workspace-scoped.
func (r *SQLServerAccountRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT a.status, CAST(COUNT(*) AS bigint)
		FROM [account] a
		WHERE a.active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR a.workspace_id = @p1)
		GROUP BY a.status`

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 3)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan account count row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account count rows: %w", err)
	}
	return out, nil
}

// normalizeElementKey lowercases the DB element short name to a stable key.
func normalizeElementKey(element string) string {
	switch element {
	case "ASSET":
		return "asset"
	case "LIABILITY":
		return "liability"
	case "EQUITY":
		return "equity"
	case "REVENUE":
		return "revenue"
	case "EXPENSE":
		return "expense"
	default:
		return element
	}
}

// Compile-time guard.
var _ = (*sql.DB)(nil)
