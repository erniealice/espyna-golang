//go:build postgresql

package ledger

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	payagingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"
	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
	stmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
)

// Report type-drift tests (RD, 20260930). Live schema (and the esqyma baseline migration): revenue.due_date /
// revenue_date / expenditure.due_date are timestamptz; treasury_collection / treasury_disbursement.payment_date
// are text (ISO date or datetime, possibly ''). The report SQL used to assume epoch-ms bigint / timestamptz and
// failed to parse. These tests run every affected query on the real column types inside a rolled-back
// transaction on leasing_usage1.

func driftTableConfig() TableConfig {
	tc := legTableConfig
	tc.Expenditure, tc.TreasuryDisbursement = "expenditure", "treasury_disbursement"
	tc.Supplier, tc.SupplierCategory, tc.ExpenditureCategory = "supplier", "supplier_category", "expenditure_category"
	return tc
}

func TestReceivablesReportsRunOnLiveColumnTypes(t *testing.T) {
	legTx(t, func(tx *sql.Tx) {
		seedLegacy(t, tx) // rev1: 1000 total, due 2026-08-15, receipt 200 paid 2026-08-05
		// A second, overdue invoice with a datetime-shaped and an empty payment_date.
		legExec(t, tx, `INSERT INTO revenue (id, client_id, workspace_id, name, reference_number, total_amount, currency, status, active, revenue_date, due_date)
			VALUES ('leg-rev2', 'leg-c1', $1, 'R2', 'INV-2', 400, 'PHP', 'complete', true, '2026-05-01', '2026-05-31')`, legWsA)
		legExec(t, tx, `INSERT INTO treasury_collection (id, revenue_id, workspace_id, name, amount, currency, status, payment_date, active, collection_type)
			VALUES ('leg-tc3', 'leg-rev2', $1, 'Datetime receipt', 100, 'PHP', 'completed', '2026-06-10T09:30:00Z', true, 'sale'),
			       ('leg-tc4', 'leg-rev2', $1, 'Blank date receipt', 50, 'PHP', 'completed', '', true, 'sale')`, legWsA)
		tc := legTableConfig

		// Aging as of 2026-09-15: rev1 balance 800 due 08-15 -> 31 days (31-60); rev2 balance 300
		// (blank-date receipt never counts) due 05-31 -> 107 days (over 90).
		asOf := "2026-09-15"
		q, args := buildReceivablesAgingQuery(tc, &agingpb.ReceivablesAgingRequest{RowDimension: "client", AsOfDate: &asOf}, legWsA)
		rows := legRows(t, tx, q, args)
		if len(rows) != 1 {
			t.Fatalf("aging rows = %v", rows)
		}
		// row_key|row_id|current|1-30|31-60|61-90|>90|total|count
		if !strings.HasSuffix(rows[0], "|0|0|800|0|300|1100|2") {
			t.Errorf("aging buckets = %s, want 31-60=800 and >90=300 (total 1100 over 2 invoices)", rows[0])
		}

		// Statement: dates are YYYY-MM-DD text across every leg; the end date is inclusive of its own day.
		q, args = buildClientStatementQuery(tc, &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1"}, legWsA)
		var dates []string
		for _, r := range legRows(t, tx, q, args) {
			dates = append(dates, strings.Split(r, "|")[0])
		}
		for _, d := range dates {
			if d != "" && len(d) != 10 {
				t.Errorf("statement date %q is not YYYY-MM-DD", d)
			}
		}
		end := "2026-08-01"
		q, args = buildClientStatementQuery(tc, &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1", EndDate: &end}, legWsA)
		var inv []string
		for _, r := range legRows(t, tx, q, args) {
			if f := strings.Split(r, "|"); f[1] == "invoice" {
				inv = append(inv, f[0])
			}
		}
		if !reflect.DeepEqual(inv, []string{"2026-05-01", "2026-08-01"}) {
			t.Errorf("statement end-date filter dropped the end day: invoices = %v", inv)
		}

		// Collection summary: yearly x client counts the datetime receipt and the legacy one, not the blank one.
		q, args = buildCollectionSummaryQuery(tc, &collsumpb.CollectionSummaryRequest{PrimaryDimension: "yearly", RowDimension: "client"}, legWsA)
		var total int64
		for _, r := range legRows(t, tx, q, args) {
			var v int64
			if _, err := fmt.Sscan(strings.Split(r, "|")[4], &v); err == nil {
				total += v
			}
		}
		if total != 350 { // 200 + 100 + 50 (blank payment_date still a collection, undated in the filter path)
			t.Errorf("collection summary total = %d, want 350", total)
		}
		// Date-filtered: only 2026-06 datetime receipt.
		s, e := "2026-06-01", "2026-06-30"
		q, args = buildCollectionSummaryQuery(tc, &collsumpb.CollectionSummaryRequest{PrimaryDimension: "monthly", RowDimension: "client", StartDate: &s, EndDate: &e}, legWsA)
		if rows := legRows(t, tx, q, args); len(rows) != 1 || !strings.Contains(rows[0], "|100|1") {
			t.Errorf("collection summary June = %v, want the single 100 receipt", rows)
		}
	})
}

func TestPayablesReportsRunOnLiveColumnTypes(t *testing.T) {
	legTx(t, func(tx *sql.Tx) {
		legExec(t, tx, `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, legWsA)
		legExec(t, tx, `INSERT INTO expenditure (id, workspace_id, supplier_id, vendor_id, name, total_amount, currency, status, active, expenditure_type, expenditure_date, due_date)
			VALUES ('leg-exp1', $1, 'leg-s1', 'leg-s1', 'E1', 900, 'PHP', 'complete', true, 'purchase', '2026-07-01', '2026-07-31')`, legWsA)
		legExec(t, tx, `INSERT INTO treasury_disbursement (id, workspace_id, expenditure_id, supplier_id, name, amount, currency, status, payment_date, active, disbursement_type)
			VALUES ('leg-td1', $1, 'leg-exp1', 'leg-s1', 'Pay1', 300, 'PHP', 'completed', '2026-08-05', true, 'payment')`, legWsA)
		tc := driftTableConfig()

		asOf := "2026-09-15"
		q, args := buildPayablesAgingQuery(tc, &payagingpb.PayablesAgingRequest{RowDimension: "supplier", AsOfDate: &asOf}, legWsA)
		rows := legRows(t, tx, q, args)
		// balance 600, due 07-31 -> 46 days overdue (31-60)
		if len(rows) != 1 || !strings.HasSuffix(rows[0], "|0|0|600|0|0|600|1") {
			t.Errorf("payables aging = %v, want one supplier with 600 in 31-60", rows)
		}

		q, args = buildDisbursementReportQuery(tc, &disbreportpb.DisbursementReportRequest{PrimaryDimension: "monthly", RowDimension: "supplier"}, legWsA)
		if rows := legRows(t, tx, q, args); len(rows) != 1 || !strings.Contains(rows[0], "|300|") {
			t.Errorf("disbursement report = %v, want one 300 cell", rows)
		}

		q, args = buildSupplierStatementQuery(tc, &stmtpb.SupplierStatementRequest{SupplierId: "leg-s1"}, legWsA)
		var types []string
		for _, r := range legRows(t, tx, q, args) {
			types = append(types, strings.Split(r, "|")[1])
		}
		if !reflect.DeepEqual(types, []string{"bill", "payment"}) {
			t.Errorf("supplier statement types = %v, want [bill payment]", types)
		}
	})
}
