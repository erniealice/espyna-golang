//go:build mysql

package ledger

import (
	"database/sql"
	"fmt"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
)

// buildClientStatementQuery constructs the UNION ALL + window function SQL
// for a client statement with running balance, and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries (?).
//
// Dialect differences from postgres gold standard:
//   - $N → ? (same left-to-right order; 5 distinct values × repeated use)
//   - $N::text IS NULL → ? IS NULL
//   - ::bigint / ::date casts removed
//   - interval '1 day' → INTERVAL 1 DAY
//   - active = true → active = 1
//   - CONCAT replaces || for string concatenation in buildClientNameQuery
//   - SUM(...) OVER (ORDER BY ...) window function ✓ (MySQL 8.0+)
//
// Parameters (same order as postgres):
//
//	? = client_id (text)
//	? = start_date (text or NULL)
//	? = end_date (text or NULL)
//	? = currency (text or NULL)
//	? = workspace_id (text or NULL)
func buildClientStatementQuery(tc TableConfig, req *clientstmtpb.ClientStatementRequest, workspaceID string) (string, []any) {
	// Base args (5 values, each may be repeated in the expanded list below).
	clientID := req.GetClientId()
	startDate := nilIfEmpty(req.GetStartDate())
	endDate := nilIfEmpty(req.GetEndDate())
	currency := nilIfEmpty(req.GetCurrency())
	wsID := nilIfEmpty(workspaceID)

	query := fmt.Sprintf(`
WITH statement AS (
    SELECT
        r.revenue_date AS date,
        'invoice' AS type,
        r.reference_number,
        r.name AS description,
        r.total_amount AS billed,
        0 AS received,
        r.id AS entity_id,
        r.status
    FROM %s r
    WHERE r.client_id = ?
        AND r.status != 'cancelled'
        AND r.active = 1
        AND (? IS NULL OR r.revenue_date >= ?)
        AND (? IS NULL OR r.revenue_date <= ?)
        AND (? IS NULL OR r.currency = ?)
        AND (? IS NULL OR r.workspace_id = ?)

    UNION ALL

    SELECT
        tc.payment_date AS date,
        'collection' AS type,
        tc.reference_number,
        tc.name AS description,
        0 AS billed,
        tc.amount AS received,
        tc.id AS entity_id,
        tc.status
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    WHERE r.client_id = ?
        AND r.status != 'cancelled'
        AND r.active = 1
        AND (? IS NULL OR tc.payment_date >= ?)
        AND (? IS NULL OR tc.payment_date < DATE_ADD(DATE(?), INTERVAL 1 DAY))
        AND (? IS NULL OR tc.currency = ?)
        AND (? IS NULL OR r.workspace_id = ?)
)
SELECT
    date,
    type,
    COALESCE(reference_number, '') AS reference_number,
    COALESCE(description, '') AS description,
    billed,
    received,
    SUM(billed - received) OVER (ORDER BY date, CASE type WHEN 'invoice' THEN 0 WHEN 'collection' THEN 1 END) AS balance,
    entity_id,
    status
FROM statement
ORDER BY date, CASE type WHEN 'invoice' THEN 0 WHEN 'collection' THEN 1 END`,
		tc.Revenue,
		tc.TreasuryCollection,
		tc.Revenue,
	)

	// Expand args: each ? placeholder bound once.
	// Revenue block: client_id, start IS NULL, start, end IS NULL, end, currency IS NULL, currency, ws IS NULL, ws
	// Collection block: client_id, start IS NULL, start, end IS NULL, end, end (DATE_ADD), currency IS NULL, currency, ws IS NULL, ws
	args := []any{
		// revenue invoice leg
		clientID,
		startDate, startDate,
		endDate, endDate,
		currency, currency,
		wsID, wsID,
		// collection leg
		clientID,
		startDate, startDate,
		endDate, endDate, endDate, // end_date appears 3× for IS NULL, < DATE_ADD(DATE(?), ...)
		currency, currency,
		wsID, wsID,
	}

	return query, args
}

// scanClientStatementEntries scans SQL result rows into proto StatementEntry slices.
func scanClientStatementEntries(rows *sql.Rows) ([]*clientstmtpb.StatementEntry, error) {
	var entries []*clientstmtpb.StatementEntry
	for rows.Next() {
		var e clientstmtpb.StatementEntry
		if err := rows.Scan(
			&e.Date,
			&e.Type,
			&e.ReferenceNumber,
			&e.Description,
			&e.Billed,
			&e.Received,
			&e.Balance,
			&e.EntityId,
			&e.Status,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// buildClientStatementSummary computes totals from statement entries.
func buildClientStatementSummary(entries []*clientstmtpb.StatementEntry, req *clientstmtpb.ClientStatementRequest) *clientstmtpb.ClientStatementSummary {
	s := &clientstmtpb.ClientStatementSummary{}
	for _, e := range entries {
		s.TotalBilled += e.Billed
		s.TotalReceived += e.Received
		switch e.Type {
		case "invoice":
			s.InvoiceCount++
		case "collection":
			s.CollectionCount++
		}
	}
	s.OutstandingBalance = s.TotalBilled - s.TotalReceived
	if req.Currency != nil {
		s.Currency = req.GetCurrency()
	}
	return s
}

// buildClientNameQuery returns a query to fetch the client display name.
// MySQL dialect: || → CONCAT; $1 → ?
func buildClientNameQuery(tc TableConfig) string {
	return fmt.Sprintf(
		"SELECT COALESCE(name, CONCAT(first_name, ' ', last_name)) FROM %s WHERE id = ?",
		tc.Client,
	)
}
