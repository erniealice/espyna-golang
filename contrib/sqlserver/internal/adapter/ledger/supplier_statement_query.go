//go:build sqlserver

package ledger

import (
	"fmt"

	stmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
)

type statementRow struct {
	TransactionDate string
	TransactionType string
	Reference       string
	Description     string
	BilledAmount    int64
	PaidAmount      int64
	SourceID        string
	Status          string
}

// buildSupplierStatementQuery constructs the UNION ALL SQL for a supplier statement.
//
// Parameters:
//
//	@p1 = supplier_id (required)
//	@p2 = start_date (optional)
//	@p3 = end_date (optional)
//	@p4 = workspace_id (optional)
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - e.expenditure_date::text → CAST(e.expenditure_date AS nvarchar(30)).
//   - TO_TIMESTAMP(d.payment_date / 1000.0) → DATEADD(ms, d.payment_date % 1000, DATEADD(s, d.payment_date / 1000, '19700101')).
//   - ($2::timestamptz IS NULL OR …) → (@p2 IS NULL OR …).
//   - ::bigint → CAST(… AS bigint).
func buildSupplierStatementQuery(tc TableConfig, req *stmtpb.SupplierStatementRequest, workspaceID string) (string, []any) {
	args := []any{
		req.GetSupplierId(),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(workspaceID),
	}

	// d.payment_date is stored as Unix milliseconds (int64).
	pmtDateExpr := "DATEADD(ms, d.payment_date % 1000, DATEADD(s, d.payment_date / 1000, '19700101'))"

	query := fmt.Sprintf(`
SELECT transaction_date, transaction_type, reference, description,
       billed_amount, paid_amount, source_id, status
FROM (
    SELECT
        CAST(e.expenditure_date AS nvarchar(30)) AS transaction_date,
        'bill' AS transaction_type,
        COALESCE(e.reference_number, '') AS reference,
        COALESCE(e.name, '') AS description,
        CAST(e.total_amount AS bigint) AS billed_amount,
        CAST(0 AS bigint) AS paid_amount,
        e.id AS source_id,
        COALESCE(e.status, '') AS status
    FROM %s e
    WHERE e.active = 1
      AND e.status NOT IN ('cancelled', 'draft')
      AND e.supplier_id = @p1
      AND (@p2 IS NULL OR e.expenditure_date >= @p2)
      AND (@p3 IS NULL OR e.expenditure_date <= @p3)
      AND (@p4 IS NULL OR e.workspace_id = @p4)

    UNION ALL

    SELECT
        CONVERT(varchar(30), %s, 126) AS transaction_date,
        'payment' AS transaction_type,
        COALESCE(d.reference_number, '') AS reference,
        COALESCE(d.name, '') AS description,
        CAST(0 AS bigint) AS billed_amount,
        CAST(d.amount AS bigint) AS paid_amount,
        d.id AS source_id,
        COALESCE(d.status, '') AS status
    FROM %s d
    JOIN %s e ON e.id = d.expenditure_id
    WHERE d.active = 1
      AND d.status IN ('paid', 'completed')
      AND e.supplier_id = @p1
      AND (@p2 IS NULL OR %s >= @p2)
      AND (@p3 IS NULL OR %s <= @p3)
      AND (@p4 IS NULL OR e.workspace_id = @p4)
) combined
ORDER BY transaction_date ASC,
         CASE transaction_type WHEN 'bill' THEN 0 ELSE 1 END ASC`,
		tc.Expenditure,
		pmtDateExpr,
		tc.TreasuryDisbursement,
		tc.Expenditure,
		pmtDateExpr, pmtDateExpr,
	)

	return query, args
}

// buildStatementEntries converts raw rows into proto entries with running balance computed in Go.
func buildStatementEntries(rows []statementRow) []*stmtpb.SupplierStatementEntry {
	entries := make([]*stmtpb.SupplierStatementEntry, 0, len(rows))
	var runningBalance int64
	for _, r := range rows {
		runningBalance += r.BilledAmount - r.PaidAmount
		entries = append(entries, &stmtpb.SupplierStatementEntry{
			Date:            r.TransactionDate,
			Type:            r.TransactionType,
			ReferenceNumber: r.Reference,
			Description:     r.Description,
			Billed:          r.BilledAmount,
			Paid:            r.PaidAmount,
			Balance:         runningBalance,
			EntityId:        r.SourceID,
			Status:          r.Status,
		})
	}
	return entries
}

func buildStatementSummary(entries []*stmtpb.SupplierStatementEntry, req *stmtpb.SupplierStatementRequest) *stmtpb.SupplierStatementSummary {
	var totalBilled, totalPaid int64
	var billCount, paymentCount int32
	for _, e := range entries {
		totalBilled += e.Billed
		totalPaid += e.Paid
		if e.Type == "bill" {
			billCount++
		} else {
			paymentCount++
		}
	}
	s := &stmtpb.SupplierStatementSummary{
		TotalBilled:        totalBilled,
		TotalPaid:          totalPaid,
		OutstandingBalance: totalBilled - totalPaid,
		BillCount:          billCount,
		PaymentCount:       paymentCount,
		Currency:           "PHP",
	}
	if req.StartDate != nil {
		sd := req.GetStartDate()
		s.StartDate = &sd
	}
	if req.EndDate != nil {
		ed := req.GetEndDate()
		s.EndDate = &ed
	}
	return s
}

// buildSupplierNameQuery returns a query to fetch the supplier display name.
// SQL Server: @p1 placeholder for supplier ID.
func buildSupplierNameQuery(tc TableConfig) string {
	return fmt.Sprintf(`SELECT COALESCE(name, '') FROM %s WHERE id = @p1`, tc.Supplier)
}
