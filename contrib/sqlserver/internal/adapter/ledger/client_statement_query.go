//go:build sqlserver

package ledger

import (
	"database/sql"
	"fmt"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
)

// buildClientStatementQuery constructs the UNION ALL + window function SQL
// for a client statement with running balance, and its parameter args.
//
// Parameters:
//
//	@p1 = client_id (text)
//	@p2 = start_date (text or NULL)
//	@p3 = end_date (text or NULL)
//	@p4 = currency (text or NULL)
//	@p5 = workspace_id (text or NULL)
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - r.total_amount::bigint → CAST(r.total_amount AS bigint).
//   - 0::bigint → CAST(0 AS bigint).
//   - revenue_date / payment_date comparisons: @pN IS NULL instead of $N::text IS NULL.
//   - ($3::date + interval '1 day') → DATEADD(day, 1, CAST(@p3 AS date)).
//   - SUM(billed - received) OVER (ORDER BY …) is supported in SQL Server 2017+.
//   - active = true → active = 1.
//   - buildClientNameQuery: COALESCE(name, first_name + ' ' + last_name).
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
        CAST(r.total_amount AS bigint) AS billed,
        CAST(0 AS bigint) AS received,
        r.id AS entity_id,
        r.status
    FROM %s r
    WHERE r.client_id = @p1
        AND r.status != 'cancelled'
        AND r.active = 1
        AND (@p2 IS NULL OR r.revenue_date >= @p2)
        AND (@p3 IS NULL OR r.revenue_date <= @p3)
        AND (@p4 IS NULL OR r.currency = @p4)
        AND (@p5 IS NULL OR r.workspace_id = @p5)

    UNION ALL

    SELECT
        tc.payment_date AS date,
        'collection' AS type,
        tc.reference_number,
        tc.name AS description,
        CAST(0 AS bigint) AS billed,
        CAST(tc.amount AS bigint) AS received,
        tc.id AS entity_id,
        tc.status
    FROM %s tc
    JOIN %s r ON r.id = tc.revenue_id
    WHERE r.client_id = @p1
        AND r.status != 'cancelled'
        AND r.active = 1
        AND (@p2 IS NULL OR tc.payment_date >= @p2)
        AND (@p3 IS NULL OR tc.payment_date < CONVERT(varchar, DATEADD(day, 1, CAST(@p3 AS date)), 23))
        AND (@p4 IS NULL OR tc.currency = @p4)
        AND (@p5 IS NULL OR r.workspace_id = @p5)
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

// buildClientNameQuery returns a query to fetch the client display name.
// SQL Server differences: COALESCE(name, first_name + ' ' + last_name) — uses + instead of ||.
// @p1 is the client_id parameter.
func buildClientNameQuery(tc TableConfig) string {
	return fmt.Sprintf(
		`SELECT COALESCE(name, first_name + ' ' + last_name) FROM %s WHERE id = @p1`,
		tc.Client,
	)
}
