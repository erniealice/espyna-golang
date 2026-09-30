//go:build postgresql

package ledger

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
)

// Application-leg tests (20260927-usage-and-pass-through-charges, C4, AC-VERT-01). They run on the
// leasing_usage1 clone inside ONE rolled-back transaction (FK triggers disabled, superuser only;
// skipped otherwise). Nothing is left behind.

var legTableConfig = TableConfig{
	Revenue: "revenue", RevenueLineItem: "revenue_line_item", Location: "location", LocationArea: "location_area",
	Client: "client", ClientCategory: "client_category", Category: "category",
	TreasuryCollection: "treasury_collection", CollectionMethod: "collection_method",
	CollectionApplication: "collection_application", RecoveryDocument: "recovery_document",
}

const (
	legWsA = "s1leg-ws-a"
	legWsB = "s1leg-ws-b"
)

func legTx(t *testing.T, fn func(tx *sql.Tx)) {
	t.Helper()
	db := scopetest.New(t, "collection_application").DB // TEST_DATABASE_URL; unset/unreachable FAILS (C16), never skips
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatalf("cannot disable FK triggers (the test role needs superuser): %v", err)
	}
	fn(tx)
}

func legExec(t *testing.T, tx *sql.Tx, q string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(q, args...); err != nil {
		t.Fatalf("seed: %v\n%s", err, q)
	}
}

// legRows runs a query and renders every row as a single string, sorted stable by the query itself.
func legRows(t *testing.T, tx *sql.Tx, q string, args []any) []string {
	t.Helper()
	rows, err := tx.Query(q, args...)
	if err != nil {
		t.Fatalf("query: %v\n%s", err, q)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			parts[i] = fmt.Sprint(v)
		}
		out = append(out, strings.Join(parts, "|"))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// seedLegacy inserts workspace A data that exists WITHOUT any application: a client, a revenue with
// a legacy receipt.
func seedLegacy(t *testing.T, tx *sql.Tx) {
	for _, ws := range []string{legWsA, legWsB} {
		legExec(t, tx, `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, ws)
	}
	legExec(t, tx, `INSERT INTO client (id, name, workspace_id, active) VALUES ('leg-c1', 'Leg Client', $1, true)`, legWsA)
	legExec(t, tx, `INSERT INTO revenue (id, client_id, workspace_id, name, reference_number, total_amount, currency, status, active, revenue_date, due_date)
		VALUES ('leg-rev1', 'leg-c1', $1, 'R1', 'INV-1', 1000, 'PHP', 'complete', true, '2026-08-01', '2026-08-15')`, legWsA)
	legExec(t, tx, `INSERT INTO treasury_collection (id, revenue_id, workspace_id, name, amount, currency, status, payment_date, active, collection_type)
		VALUES ('leg-tc1', 'leg-rev1', $1, 'Legacy receipt', 200, 'PHP', 'completed', '2026-08-05', true, 'sale')`, legWsA)
}

// legViews returns the table config for the report queries. Since the report-type-drift fix (RD,
// 20260930) the queries run directly on the live column types (revenue.due_date/revenue_date
// timestamptz, treasury_collection.payment_date text), so no temporary views are needed.
func legViews(t *testing.T, tx *sql.Tx) TableConfig {
	t.Helper()
	return legTableConfig
}

func balancesArgs(ws string) []any {
	if ws == "" {
		return []any{nil}
	}
	return []any{ws}
}

func agingRequest(asOf string) *agingpb.ReceivablesAgingRequest {
	return &agingpb.ReceivablesAgingRequest{RowDimension: "client", AsOfDate: &asOf}
}

func statementLegs(t *testing.T, q string) []string {
	t.Helper()
	i := strings.Index(q, "WITH statement AS (")
	j := strings.Index(q, "\n)\nSELECT")
	if i < 0 || j < 0 {
		t.Fatalf("unexpected statement query shape")
	}
	return strings.Split(q[i+len("WITH statement AS ("):j], "\n    UNION ALL\n")
}

// AC-VERT-01: with no collection_application rows each of the four queries returns exactly what the
// pre-S1 query returns (frozen baselines in legacy_report_queries_test.go), over seeded legacy data.
// The statement is compared functionally and structurally (pre-S1 legs byte-identical in the new query).
func TestReportQueriesUnchangedWithoutApplications(t *testing.T) {
	legTx(t, func(tx *sql.Tx) {
		var n int
		if err := tx.QueryRow(`SELECT count(*) FROM collection_application`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("clone holds %d application rows; the baseline comparison needs an empty collection_application table (clean up committed test data)", n)
		}
		seedLegacy(t, tx)
		vc := legViews(t, tx)
		compare := func(name string, cur, old string, curArgs, oldArgs []any) {
			got, want := legRows(t, tx, cur, curArgs), legRows(t, tx, old, oldArgs)
			if len(want) == 0 {
				t.Errorf("%s: baseline returned no rows; the comparison proves nothing", name)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: result differs without applications\n got: %v\nwant: %v", name, got, want)
			}
		}
		for _, ws := range []string{legWsA, ""} {
			cur, _ := buildClientBalancesQuery(legTableConfig)
			old, _ := legacyBuildClientBalancesQuery(legTableConfig)
			compare("balances/"+ws, cur, old, balancesArgs(ws), balancesArgs(ws))
		}
		for _, asOf := range []string{"2026-12-31", "2026-08-20"} {
			cur, ca := buildReceivablesAgingQuery(vc, agingRequest(asOf), legWsA)
			old, oa := legacyBuildReceivablesAgingQuery(vc, agingRequest(asOf), legWsA)
			compare("aging/"+asOf, cur, old, ca, oa)
		}
		for _, dims := range [][2]string{{"monthly", "client"}, {"yearly", "collection_method"}, {"client", "collection_type"}} {
			req := &collsumpb.CollectionSummaryRequest{PrimaryDimension: dims[0], RowDimension: dims[1]}
			cur, ca := buildCollectionSummaryQuery(vc, req, legWsA)
			old, oa := legacyBuildCollectionSummaryQuery(vc, req, legWsA)
			compare("summary/"+dims[0]+"x"+dims[1], cur, old, ca, oa)
		}
		// statement: full functional comparison (the pre-S1 legs are also byte-identical inside the new query)
		sreq := &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1"}
		cur, ca := buildClientStatementQuery(legTableConfig, sreq, legWsA)
		old, oa := legacyBuildClientStatementQuery(legTableConfig, sreq, legWsA)
		compare("statement", cur, old, ca, oa)
		curLegs, oldLegs := statementLegs(t, cur), statementLegs(t, old)
		if len(oldLegs) != 2 || len(curLegs) != 4 || strings.TrimSpace(curLegs[0]) != strings.TrimSpace(oldLegs[0]) || strings.TrimSpace(curLegs[1]) != strings.TrimSpace(oldLegs[1]) {
			t.Errorf("statement: pre-S1 legs are not preserved verbatim (%d -> %d legs)", len(oldLegs), len(curLegs))
		}
	})
}

// The application leg adds exactly the applied amounts, strictly within the workspace.
func TestApplicationLegAdjustsReports(t *testing.T) {
	legTx(t, func(tx *sql.Tx) {
		seedLegacy(t, tx)
		vc := legViews(t, tx)
		// applied_at 2026-09-10 (ms)
		const appliedAt = 1789000000000
		legExec(t, tx, `INSERT INTO treasury_collection (id, workspace_id, client_id, name, amount, currency, status, payment_date, active, collection_type)
			VALUES ('leg-tc2', $1, 'leg-c1', 'Receipt', 500, 'PHP', 'completed', '2026-09-10', true, 'receipt')`, legWsA)
		app := func(id, ws, status string, amount int64) {
			legExec(t, tx, `INSERT INTO collection_application (id, workspace_id, treasury_collection_id, client_id, target_kind, revenue_id, application_kind, amount, currency, applied_at, status, active)
				VALUES ($1, $2, 'leg-tc2', 'leg-c1', 'APPLICATION_TARGET_KIND_REVENUE', 'leg-rev1', 'APPLICATION_KIND_CASH', $3, 'PHP', $4, $5, true)`, id, ws, amount, appliedAt, status)
		}
		app("leg-a1", legWsA, "APPLICATION_STATUS_APPLIED", 300)
		app("leg-a2", legWsA, "APPLICATION_STATUS_REVERSED", 100) // reversed: never counts
		app("leg-a3", legWsB, "APPLICATION_STATUS_APPLIED", 400)  // foreign workspace: never counts for A
		legExec(t, tx, `INSERT INTO recovery_document (id, workspace_id, document_series_id, sequence_number, document_number, document_type, corrects_document_id, client_id, issue_date, due_date, total_amount, currency, status, issuance_key, active)
			VALUES ('leg-d1', $1, 'leg-ser', 1, 'SOA-1', 'RECOVERY_DOCUMENT_TYPE_STATEMENT', NULL, 'leg-c1', '2026-09-01', '2026-09-30', 700, 'PHP', 'RECOVERY_DOCUMENT_STATUS_ISSUED', 'k1', true),
			       ('leg-d2', $1, 'leg-ser', 2, 'CN-1', 'RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE', 'leg-d1', 'leg-c1', '2026-09-05', NULL, -50, 'PHP', 'RECOVERY_DOCUMENT_STATUS_ISSUED', 'k2', true)`, legWsA)

		// balances: 1000 - 200 (legacy) - 300 (applied) = 500; workspace B sees none of A's revenue.
		q, _ := buildClientBalancesQuery(legTableConfig)
		if got := legRows(t, tx, q, balancesArgs(legWsA)); !reflect.DeepEqual(got, []string{"leg-c1|500"}) {
			t.Errorf("balances A = %v, want [leg-c1|500]", got)
		}
		if got := legRows(t, tx, q, balancesArgs(legWsB)); len(got) != 0 {
			t.Errorf("balances B = %v, want none (foreign application must not leak)", got)
		}

		// aging as-of 2026-12-31: 1000 - 200 - 300 = 500 outstanding, one invoice.
		agingReq := &agingpb.ReceivablesAgingRequest{RowDimension: "client"}
		asOf := "2026-12-31"
		agingReq.AsOfDate = &asOf
		q, args := buildReceivablesAgingQuery(vc, agingReq, legWsA)
		rows := legRows(t, tx, q, args)
		if len(rows) != 1 || !strings.HasSuffix(rows[0], "|500|1") {
			t.Errorf("aging A = %v, want one row totalling 500 over 1 invoice", rows)
		}
		// as-of BEFORE the application date ignores it: 800 outstanding
		early := "2026-09-01"
		agingReq.AsOfDate = &early
		q, args = buildReceivablesAgingQuery(vc, agingReq, legWsA)
		if rows := legRows(t, tx, q, args); len(rows) != 1 || !strings.HasSuffix(rows[0], "|800|1") {
			t.Errorf("aging A as-of before application = %v, want 800", rows)
		}

		// statement: application received row (300), recovery documents billed (700, -50); no foreign or reversed rows.
		q, args = buildClientStatementQuery(legTableConfig, &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1"}, legWsA)
		var app300, doc700, cn50, apps int
		for _, r := range legRows(t, tx, q, args) {
			f := strings.Split(r, "|")
			switch {
			case f[1] == "application":
				apps++
				if f[5] == "300" {
					app300++
				}
			case f[1] == "recovery_document" && f[4] == "700":
				doc700++
			case f[1] == "recovery_document" && f[4] == "-50":
				cn50++
			}
		}
		if apps != 1 || app300 != 1 || doc700 != 1 || cn50 != 1 {
			t.Errorf("statement legs: applications=%d (300:%d) doc700=%d cn50=%d", apps, app300, doc700, cn50)
		}
		// no workspace parameter => the new standalone legs contribute nothing (strict)
		q, args = buildClientStatementQuery(legTableConfig, &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1"}, "")
		for _, r := range legRows(t, tx, q, args) {
			if strings.Contains(r, "|application|") || strings.Contains(r, "|recovery_document|") {
				t.Errorf("wildcard statement leaked a new leg row: %s", r)
			}
		}

		// summary: legacy 200 + the receipt 500 (revenue_id NULL) = 700 for A; the wildcard query drops the receipt leg.
		sumReq := &collsumpb.CollectionSummaryRequest{PrimaryDimension: "yearly", RowDimension: "client"}
		q, args = buildCollectionSummaryQuery(vc, sumReq, legWsA)
		var total int64
		for _, r := range legRows(t, tx, q, args) {
			f := strings.Split(r, "|")
			var v int64
			fmt.Sscan(f[4], &v)
			total += v
		}
		if total != 700 {
			t.Errorf("collection summary A = %d, want 700 (200 legacy + 500 receipt)", total)
		}
		q, args = buildCollectionSummaryQuery(vc, sumReq, "")
		total = 0
		for _, r := range legRows(t, tx, q, args) {
			f := strings.Split(r, "|")
			var v int64
			fmt.Sscan(f[4], &v)
			total += v
		}
		if total != 200 {
			t.Errorf("wildcard collection summary = %d, want 200 (receipt leg is strict)", total)
		}
	})
}

// A1 m1: the applied-to-revenue subtotal adds no `$n IS NULL OR` workspace wildcard; it is scoped by
// its caller's join to the parent revenue row.
func TestAppliedToRevenueSubqueryAddsNoWorkspaceWildcard(t *testing.T) {
	q := appliedToRevenueSubquery(legTableConfig.CollectionApplication, "")
	if strings.Contains(q, "IS NULL OR") || strings.Contains(q, "$") {
		t.Fatalf("the subtotal must carry no workspace parameter or wildcard:\n%s", q)
	}
	if !strings.Contains(q, "FROM "+legTableConfig.CollectionApplication+" ca") {
		t.Fatalf("the table name must come from TableConfig:\n%s", q)
	}
	for name, q := range map[string]string{"balances": func() string { s, _ := buildClientBalancesQuery(legTableConfig); return s }()} {
		if !strings.Contains(q, "applied.workspace_id = r.workspace_id") {
			t.Errorf("%s: the subtotal must be joined on the parent revenue's workspace", name)
		}
	}
}

// A1 m7: the statement's application rows join their referenced receipt, invoice and document on
// the application's workspace too, so a reference that ever points across workspaces renders
// nothing from the foreign workspace.
func TestStatementApplicationJoinsNeverRenderAForeignReference(t *testing.T) {
	legTx(t, func(tx *sql.Tx) {
		seedLegacy(t, tx)
		legExec(t, tx, `INSERT INTO treasury_collection (id, workspace_id, client_id, name, amount, currency, status, payment_date, active, collection_type, reference_number)
			VALUES ('leg-tcB', $1, 'leg-c1', 'Foreign receipt', 500, 'PHP', 'completed', '2026-09-10', true, 'receipt', 'FOREIGN-RCPT')`, legWsB)
		legExec(t, tx, `INSERT INTO revenue (id, client_id, workspace_id, name, reference_number, total_amount, currency, status, active, revenue_date, due_date)
			VALUES ('leg-revB', 'leg-c1', $1, 'RB', 'FOREIGN-INV', 1000, 'PHP', 'complete', true, '2026-08-01', '2026-08-15')`, legWsB)
		legExec(t, tx, `INSERT INTO collection_application (id, workspace_id, treasury_collection_id, client_id, target_kind, revenue_id, application_kind, amount, currency, applied_at, status, active)
			VALUES ('leg-aX', $1, 'leg-tcB', 'leg-c1', 'APPLICATION_TARGET_KIND_REVENUE', 'leg-revB', 'APPLICATION_KIND_CASH', 300, 'PHP', 1789000000000, 'APPLICATION_STATUS_APPLIED', true)`, legWsA)
		q, args := buildClientStatementQuery(legTableConfig, &clientstmtpb.ClientStatementRequest{ClientId: "leg-c1"}, legWsA)
		seen := 0
		for _, r := range legRows(t, tx, q, args) {
			if strings.Contains(r, "FOREIGN-") {
				t.Errorf("statement rendered a foreign-workspace reference: %s", r)
			}
			if strings.Contains(r, "|application|") {
				seen++
			}
		}
		if seen != 1 {
			t.Errorf("want the one own-workspace application row, got %d", seen)
		}
	})
}
