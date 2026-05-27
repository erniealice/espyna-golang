//go:build postgresql

package ledger

import (
	"context"

	"github.com/erniealice/espyna-golang/consumer"
	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
)

// GetCashBookReport executes a UNION ALL query over revenue and expenditure tables
// to produce a simple chronological cash book showing receipts and disbursements.
// amounts are stored in pesos in the DB and multiplied by 100 to produce centavos.
func (a *LedgerReportingAdapter) GetCashBookReport(
	ctx context.Context,
	req *reportpb.CashBookReportRequest,
) (*reportpb.CashBookReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	limit := int32(200)
	if req.Limit != nil && req.GetLimit() > 0 {
		limit = req.GetLimit()
	}

	query := `
		SELECT tx_date, description, reference, tx_type, amount
		FROM (
			SELECT
				TO_CHAR(date_created, 'YYYY-MM-DD') AS tx_date,
				COALESCE(NULLIF(TRIM(name), ''), 'Collection') AS description,
				COALESCE(NULLIF(reference_number, ''), '-') AS reference,
				'Receipt' AS tx_type,
				-- TODO(Q-CENTAVO-INFLATION): verify fycha display caller before changing *100
				(total_amount * 100)::bigint AS amount
			FROM ` + a.tableConfig.Revenue + `
			WHERE status NOT IN ('cancelled', 'draft')
			AND ($1::text IS NULL OR workspace_id = $1)

			UNION ALL

			SELECT
				TO_CHAR(expenditure_date, 'YYYY-MM-DD') AS tx_date,
				COALESCE(NULLIF(name, ''), 'Payment') AS description,
				COALESCE(NULLIF(reference_number, ''), '-') AS reference,
				CASE WHEN expenditure_type = 'purchase' THEN 'Purchase' ELSE 'Expense' END AS tx_type,
				-- TODO(Q-CENTAVO-INFLATION): verify fycha display caller before changing *100
				(total_amount * 100)::bigint AS amount
			FROM ` + a.tableConfig.Expenditure + `
			WHERE status NOT IN ('cancelled', 'draft')
			AND ($1::text IS NULL OR workspace_id = $1)
		) combined
		ORDER BY tx_date DESC, reference
		LIMIT $2
	`

	rows, err := a.db.QueryContext(ctx, query, nilIfEmpty(workspaceID), limit)
	if err != nil {
		return &reportpb.CashBookReportResponse{Success: false}, err
	}
	defer rows.Close()

	var data []*reportpb.CashBookReportRow
	for rows.Next() {
		var row reportpb.CashBookReportRow
		if err := rows.Scan(&row.TxDate, &row.Description, &row.Reference, &row.TxType, &row.Amount); err != nil {
			return nil, err
		}
		data = append(data, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &reportpb.CashBookReportResponse{
		Data:    data,
		Success: true,
	}, nil
}

// GetSimplePayablesAgingReport executes a CTE-based SQL query to compute outstanding
// payables per supplier bucketed into 5 aging bands. This is the simplified view-level
// version; the full parameterized version is GetPayablesAgingReport (payables_aging proto).
// Amounts are stored in pesos in the DB and multiplied by 100 to produce centavos.
func (a *LedgerReportingAdapter) GetSimplePayablesAgingReport(
	ctx context.Context,
	req *reportpb.PayablesAgingReportRequest,
) (*reportpb.PayablesAgingReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `
		WITH outstanding AS (
			SELECT
				e.id,
				COALESCE(NULLIF(TRIM(s.company_name), ''), NULLIF(TRIM(e.name), ''), 'Unknown') AS supplier_name,
				e.total_amount - COALESCE(paid.total_paid, 0) AS outstanding_amount,
				CURRENT_DATE - COALESCE(e.due_date, e.expenditure_date)::date AS days_overdue
			FROM ` + a.tableConfig.Expenditure + ` e
			LEFT JOIN ` + a.tableConfig.Supplier + ` s ON s.id = e.supplier_id
			LEFT JOIN (
				SELECT d.expenditure_id, SUM(d.amount) AS total_paid
				FROM ` + a.tableConfig.TreasuryDisbursement + ` d
				WHERE d.active = true AND d.status IN ('paid', 'completed')
				GROUP BY d.expenditure_id
			) paid ON paid.expenditure_id = e.id
			WHERE e.active = true
			  AND e.expenditure_type IN ('purchase', 'expense')
			  AND e.status NOT IN ('paid', 'cancelled')
			  AND e.total_amount - COALESCE(paid.total_paid, 0) > 0
			  AND ($1::text IS NULL OR e.workspace_id = $1)
		)
		SELECT
			supplier_name,
			-- TODO(Q-CENTAVO-INFLATION): verify fycha display caller before changing *100
			(COALESCE(SUM(outstanding_amount) FILTER (WHERE days_overdue <= 0), 0) * 100)::bigint AS current_amt,
			(COALESCE(SUM(outstanding_amount) FILTER (WHERE days_overdue BETWEEN 1 AND 30), 0) * 100)::bigint AS days_30,
			(COALESCE(SUM(outstanding_amount) FILTER (WHERE days_overdue BETWEEN 31 AND 60), 0) * 100)::bigint AS days_60,
			(COALESCE(SUM(outstanding_amount) FILTER (WHERE days_overdue BETWEEN 61 AND 90), 0) * 100)::bigint AS days_90,
			(COALESCE(SUM(outstanding_amount) FILTER (WHERE days_overdue > 90), 0) * 100)::bigint AS over_90,
			(COALESCE(SUM(outstanding_amount), 0) * 100)::bigint AS total
		FROM outstanding
		GROUP BY supplier_name
		ORDER BY total DESC
	`

	rows, err := a.db.QueryContext(ctx, query, nilIfEmpty(workspaceID))
	if err != nil {
		return &reportpb.PayablesAgingReportResponse{Success: false}, err
	}
	defer rows.Close()

	var data []*reportpb.PayablesAgingReportRow
	for rows.Next() {
		var row reportpb.PayablesAgingReportRow
		if err := rows.Scan(
			&row.SupplierName,
			&row.Current,
			&row.Days_30,
			&row.Days_60,
			&row.Days_90,
			&row.Over_90,
			&row.Total,
		); err != nil {
			return nil, err
		}
		data = append(data, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &reportpb.PayablesAgingReportResponse{
		Data:    data,
		Success: true,
	}, nil
}
