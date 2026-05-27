//go:build mysql

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
// Dialect differences from postgres gold standard:
//   - $N → ? (same left-to-right order; 4 distinct values × repeated use)
//   - $N::timestamptz IS NULL → ? IS NULL
//   - e.expenditure_date::timestamptz → e.expenditure_date (MySQL datetime direct compare)
//   - e.expenditure_date::text → DATE_FORMAT(e.expenditure_date, '%Y-%m-%dT%H:%i:%sZ')
//   - TO_CHAR(TO_TIMESTAMP(d.payment_date / 1000.0), ...) → DATE_FORMAT(FROM_UNIXTIME(d.payment_date / 1000), ...)
//   - TO_TIMESTAMP(d.payment_date / 1000.0) → FROM_UNIXTIME(d.payment_date / 1000)
//   - ::bigint / ::text casts removed
//   - active = true → active = 1
func buildSupplierStatementQuery(tc TableConfig, req *stmtpb.SupplierStatementRequest, workspaceID string) (string, []any) {
	// Base args (4 values).
	supplierID := req.GetSupplierId()
	startDate := nilIfEmpty(req.GetStartDate())
	endDate := nilIfEmpty(req.GetEndDate())
	wsID := nilIfEmpty(workspaceID)

	query := fmt.Sprintf(`
SELECT transaction_date, transaction_type, reference, description,
       billed_amount, paid_amount, source_id, status
FROM (
    SELECT
        DATE_FORMAT(e.expenditure_date, '%%Y-%%m-%%dT%%H:%%i:%%sZ') AS transaction_date,
        'bill' AS transaction_type,
        COALESCE(e.reference_number, '') AS reference,
        COALESCE(e.name, '') AS description,
        e.total_amount AS billed_amount,
        0 AS paid_amount,
        e.id AS source_id,
        COALESCE(e.status, '') AS status
    FROM %s e
    WHERE e.active = 1
      AND e.status NOT IN ('cancelled', 'draft')
      AND e.supplier_id = ?
      AND (? IS NULL OR e.expenditure_date >= ?)
      AND (? IS NULL OR e.expenditure_date <= ?)
      AND (? IS NULL OR e.workspace_id = ?)

    UNION ALL

    SELECT
        DATE_FORMAT(FROM_UNIXTIME(d.payment_date / 1000), '%%Y-%%m-%%dT%%H:%%i:%%sZ') AS transaction_date,
        'payment' AS transaction_type,
        COALESCE(d.reference_number, '') AS reference,
        COALESCE(d.name, '') AS description,
        0 AS billed_amount,
        d.amount AS paid_amount,
        d.id AS source_id,
        COALESCE(d.status, '') AS status
    FROM %s d
    JOIN %s e ON e.id = d.expenditure_id
    WHERE d.active = 1
      AND d.status IN ('paid', 'completed')
      AND e.supplier_id = ?
      AND (? IS NULL OR FROM_UNIXTIME(d.payment_date / 1000) >= ?)
      AND (? IS NULL OR FROM_UNIXTIME(d.payment_date / 1000) <= ?)
      AND (? IS NULL OR e.workspace_id = ?)
) combined
ORDER BY transaction_date ASC,
         CASE transaction_type WHEN 'bill' THEN 0 ELSE 1 END ASC`,
		tc.Expenditure,
		tc.TreasuryDisbursement,
		tc.Expenditure,
	)

	// Expand each arg for its ? placeholders.
	// Expenditure leg: supplier_id, start IS NULL, start, end IS NULL, end, ws IS NULL, ws
	// Disbursement leg: supplier_id, start IS NULL, start, end IS NULL, end, ws IS NULL, ws
	args := []any{
		// expenditure leg
		supplierID,
		startDate, startDate,
		endDate, endDate,
		wsID, wsID,
		// disbursement leg
		supplierID,
		startDate, startDate,
		endDate, endDate,
		wsID, wsID,
	}

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
// Dialect: $1 → ?
func buildSupplierNameQuery(tc TableConfig) string {
	return fmt.Sprintf("SELECT COALESCE(name, '') FROM %s WHERE id = ?", tc.Supplier)
}
