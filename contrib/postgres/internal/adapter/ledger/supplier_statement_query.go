//go:build postgresql

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

func buildSupplierStatementQuery(tc TableConfig, req *stmtpb.SupplierStatementRequest, workspaceID string) (string, []any) {
	// $1 = supplier_id (required)
	// $2 = start_date (optional)
	// $3 = end_date (optional)
	// $4 = workspace_id (optional)
	args := []any{
		req.GetSupplierId(),
		nilIfEmpty(req.GetStartDate()),
		nilIfEmpty(req.GetEndDate()),
		nilIfEmpty(workspaceID),
	}

	// UNION ALL: expenditures (bills) + disbursements (payments)
	// Order by date, then bills before payments on same date
	query := fmt.Sprintf(`
SELECT transaction_date, transaction_type, reference, description,
       billed_amount, paid_amount, source_id, status
FROM (
    SELECT
        e.expenditure_date::text AS transaction_date,
        'bill' AS transaction_type,
        COALESCE(e.reference_number, '') AS reference,
        COALESCE(e.name, '') AS description,
        e.total_amount::bigint AS billed_amount,
        0::bigint AS paid_amount,
        e.id AS source_id,
        COALESCE(e.status, '') AS status
    FROM %s e
    WHERE e.active = true
      AND e.status NOT IN ('cancelled', 'draft')
      AND e.supplier_id = $1
      AND ($2::timestamptz IS NULL OR e.expenditure_date::timestamptz >= $2::timestamptz)
      AND ($3::timestamptz IS NULL OR e.expenditure_date::timestamptz <= $3::timestamptz)
      AND ($4::text IS NULL OR e.workspace_id = $4)

    UNION ALL

    SELECT
        TO_CHAR(TO_TIMESTAMP(d.payment_date / 1000.0), 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS transaction_date,
        'payment' AS transaction_type,
        COALESCE(d.reference_number, '') AS reference,
        COALESCE(d.name, '') AS description,
        0::bigint AS billed_amount,
        d.amount::bigint AS paid_amount,
        d.id AS source_id,
        COALESCE(d.status, '') AS status
    FROM %s d
    JOIN %s e ON e.id = d.expenditure_id
    WHERE d.active = true
      AND d.status IN ('paid', 'completed')
      AND e.supplier_id = $1
      AND ($2::timestamptz IS NULL OR TO_TIMESTAMP(d.payment_date / 1000.0) >= $2::timestamptz)
      AND ($3::timestamptz IS NULL OR TO_TIMESTAMP(d.payment_date / 1000.0) <= $3::timestamptz)
      AND ($4::text IS NULL OR e.workspace_id = $4)
) combined
ORDER BY transaction_date ASC,
         CASE transaction_type WHEN 'bill' THEN 0 ELSE 1 END ASC`,
		tc.Expenditure,
		tc.TreasuryDisbursement,
		tc.Expenditure,
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
func buildSupplierNameQuery(tc TableConfig) string {
	return fmt.Sprintf(`SELECT COALESCE(name, '') FROM %s WHERE id = $1`, tc.Supplier)
}
