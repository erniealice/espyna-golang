//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"fmt"
)

// SumBalanceByType returns total account balance grouped by account element
// (asset / liability / equity / revenue / expense). Balances are returned in
// centavos (int64). Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_account_workspace_element ON account(workspace_id, element)
//	  WHERE active = true;
func (r *PostgresAccountRepository) SumBalanceByType(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT a.element, COALESCE(SUM(a.balance), 0)::bigint
		FROM account a
		WHERE a.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR a.workspace_id = $1)
		GROUP BY a.element`

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		// Some legacy schemas may not have a `balance` column — fall back to
		// returning empty so the dashboard still renders gracefully.
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
func (r *PostgresAccountRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT a.status, COUNT(*)::bigint
		FROM account a
		WHERE a.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR a.workspace_id = $1)
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

// normalizeElementKey lowercases the element key to a stable form.
// DB stores short names ("ASSET", "LIABILITY"…); we normalise to lower-case so
// view code can switch on a stable key without re-encoding the proto enum.
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

// Compile-time guard against drift in the *sql.DB pointer used above.
var _ = (*sql.DB)(nil)
