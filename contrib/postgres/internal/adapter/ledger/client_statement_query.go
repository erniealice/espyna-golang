//go:build postgresql

package ledger

import (
	"database/sql"
	"fmt"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
)

// buildClientStatementQuery constructs the UNION ALL + window function SQL
// for a client statement with running balance, and its parameter args.
// Table names come from TableConfig (developer-configured, safe for fmt.Sprintf).
// All user-provided values use parameterized queries ($1, $2, ...).
//
// Parameters:
//
//	$1 = client_id (text)
//	$2 = start_date (text or NULL)
//	$3 = end_date (text or NULL)
//	$4 = currency (text or NULL)
//	$5 = workspace_id (text or NULL)
func buildClientStatementQuery(tc TableConfig, req *clientstmtpb.ClientStatementRequest, workspaceID string) (string, []any) {
	args := []any{
		req.GetClientId(),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(req.GetCurrency()),
		nilIfEmpty(workspaceID),
	}

	query := fmt.Sprintf(`
WITH statement AS (
    SELECT
        r.revenue_date AS date,
        'invoice' AS type,
        r.reference_number,
        r.name AS description,
        r.total_amount::bigint AS billed,
        0::bigint AS received,
        r.id AS entity_id,
        r.status
    FROM %s r
    WHERE r.client_id = $1
        AND r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR r.revenue_date >= $2)
        AND ($3::text IS NULL OR r.revenue_date <= $3)
        AND ($4::text IS NULL OR r.currency = $4)
        AND ($5::text IS NULL OR r.workspace_id = $5)

    UNION ALL

    SELECT
        tc.payment_date AS date,
        'collection' AS type,
        tc.reference_number,
        tc.name AS description,
        0::bigint AS billed,
        tc.amount::bigint AS received,
        tc.id AS entity_id,
        tc.status
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    WHERE r.client_id = $1
        AND r.status != 'cancelled'
        AND r.active = true
        AND ($2::text IS NULL OR tc.payment_date >= $2::date)
        AND ($3::text IS NULL OR tc.payment_date < ($3::date + interval '1 day'))
        AND ($4::text IS NULL OR tc.currency = $4)
        AND ($5::text IS NULL OR r.workspace_id = $5)
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

// buildClientNameQuery returns a query and args to fetch the client display name.
func buildClientNameQuery(tc TableConfig) string {
	return fmt.Sprintf(
		`SELECT COALESCE(name, first_name || ' ' || last_name) FROM %s WHERE id = $1`,
		tc.Client,
	)
}
