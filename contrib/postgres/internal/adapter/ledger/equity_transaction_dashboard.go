//go:build postgresql

package ledger

import (
	"context"
	"fmt"
	"strings"
	"time"

	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// SumByTypeYTD groups SUM(amount) of equity transactions by transaction_type
// for the given calendar year. Workspace-scoped, centavos.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_equity_txn_workspace_type_date
//	  ON equity_transaction(workspace_id, transaction_type, transaction_date DESC);
func (r *PostgresEquityTransactionRepository) SumByTypeYTD(
	ctx context.Context,
	workspaceID string,
	year int,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	yearEnd := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	const query = `
		SELECT et.transaction_type, COALESCE(SUM(et.amount), 0)::bigint
		FROM equity_transaction et
		WHERE et.transaction_date >= $2
		  AND et.transaction_date < $3
		  AND ($1::text IS NULL OR $1::text = '' OR et.workspace_id = $1)
		GROUP BY et.transaction_type`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, yearStart, yearEnd)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 4)
	for rows.Next() {
		var (
			txnType string
			total   int64
		)
		if scanErr := rows.Scan(&txnType, &total); scanErr != nil {
			return nil, fmt.Errorf("failed to scan equity_transaction sum row: %w", scanErr)
		}
		out[normalizeEquityTxnTypeKey(txnType)] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating equity_transaction sum rows: %w", err)
	}
	return out, nil
}

// RecentTransactions returns the latest equity transactions, newest-first.
// Workspace-scoped.
func (r *PostgresEquityTransactionRepository) RecentTransactions(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*equitytransactionpb.EquityTransaction, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			et.id,
			et.equity_account_id,
			et.transaction_type,
			et.amount,
			et.transaction_date,
			COALESCE(et.description, '')
		FROM equity_transaction et
		WHERE ($1::text IS NULL OR $1::text = '' OR et.workspace_id = $1)
		ORDER BY et.transaction_date DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent equity transactions: %w", err)
	}
	defer rows.Close()

	var out []*equitytransactionpb.EquityTransaction
	for rows.Next() {
		var (
			id              string
			equityAccountID string
			txnType         string
			amount          int64
			txnDate         int64
			description     string
		)
		if scanErr := rows.Scan(&id, &equityAccountID, &txnType, &amount, &txnDate, &description); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent equity transaction row: %w", scanErr)
		}
		t := &equitytransactionpb.EquityTransaction{
			Id:              id,
			EquityAccountId: equityAccountID,
			TransactionType: parseEquityTransactionType(txnType),
			Amount:          amount,
			TransactionDate: txnDate,
		}
		if description != "" {
			d := description
			t.Description = &d
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent equity transaction rows: %w", err)
	}
	return out, nil
}

// parseEquityTransactionType maps a DB short name (e.g. "CONTRIBUTION") to the
// proto enum.
func parseEquityTransactionType(s string) equitytransactionpb.EquityTransactionType {
	switch strings.ToUpper(s) {
	case "CONTRIBUTION":
		return equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_CONTRIBUTION
	case "WITHDRAWAL":
		return equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_WITHDRAWAL
	case "DISTRIBUTION":
		return equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_DISTRIBUTION
	case "TRANSFER":
		return equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_TRANSFER
	default:
		return equitytransactionpb.EquityTransactionType_EQUITY_TRANSACTION_TYPE_UNSPECIFIED
	}
}

// normalizeEquityTxnTypeKey maps the DB short name to a stable lower-case key
// so view code can switch on a stable value.
func normalizeEquityTxnTypeKey(s string) string {
	return strings.ToLower(s)
}
