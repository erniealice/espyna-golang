//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	expreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	payagingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"
	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	revreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
	suppstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
)

// TableConfig holds table names for the ledger reporting adapter.
// Unlike entity repositories that use a single table, reporting queries
// span multiple tables, so the adapter needs its own config.
type TableConfig struct {
	Revenue              string
	RevenueLineItem      string
	InventoryTransaction string
	InventoryItem        string
	Product              string
	Location             string
	RevenueCategory      string
	Expenditure          string
	ExpenditureLineItem  string // expenditure_line_item table
	ExpenditureCategory  string // expenditure_category table
	Supplier             string // supplier table
	ProductCollection    string // product_collection table (product ↔ product line join) — deprecated, kept for backward compat
	Collection           string // collection table (product line / product_line) — deprecated, kept for backward compat
	Line                 string // line table (product taxonomy — product.line_id FK target)
	LocationArea         string // location_area table
	SupplierCategory     string // supplier_category table
	TreasuryDisbursement string // treasury_disbursement table
	DisbursementMethod   string // disbursement_method table
	Client               string // client table
	ClientCategory       string // client_category table
	Category             string // category table (parent categories)
	TreasuryCollection   string // treasury_collection table
	CollectionMethod     string // collection_method table
	PaymentTerm          string // payment_term table
}

// LedgerReportingAdapter implements ports.LedgerReportingService using PostgreSQL.
// Unlike entity repositories that use the registry pattern,
// the ledger reporting adapter is instantiated directly because it
// spans multiple tables and doesn't follow the single-entity CRUD model.
//
// Current implementation: queries raw revenue + inventory tables directly.
// Future: when Journal Entries are implemented, this adapter will be
// replaced with one that queries journal entries by account type.
type LedgerReportingAdapter struct {
	db          *sql.DB
	tableConfig TableConfig
}

// NewLedgerReportingAdapter creates a new ledger reporting adapter.
func NewLedgerReportingAdapter(db *sql.DB, config TableConfig) *LedgerReportingAdapter {
	return &LedgerReportingAdapter{db: db, tableConfig: config}
}

// GetGrossProfitReport executes a CTE-based SQL query to compute gross profit
// grouped by the requested dimension (product, location, category, or period).
func (a *LedgerReportingAdapter) GetGrossProfitReport(
	ctx context.Context,
	req *reportpb.GrossProfitReportRequest,
) (*reportpb.GrossProfitReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildGrossProfitQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineItems []*reportpb.GrossProfitLineItem
	for rows.Next() {
		var item reportpb.GrossProfitLineItem
		var groupID sql.NullString
		if err := rows.Scan(
			&item.GroupKey,
			&groupID,
			&item.TotalRevenue,
			&item.TotalDiscount,
			&item.NetRevenue,
			&item.CostOfGoodsSold,
			&item.GrossProfit,
			&item.GrossProfitMargin,
			&item.UnitsSold,
			&item.TransactionCount,
		); err != nil {
			return nil, err
		}
		if groupID.Valid {
			item.GroupId = &groupID.String
		}
		lineItems = append(lineItems, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary := computeSummary(lineItems, req)

	return &reportpb.GrossProfitReportResponse{
		LineItems: lineItems,
		Summary:   summary,
		Success:   true,
	}, nil
}

// GetRevenueReport executes a two-dimensional pivot SQL query to compute revenue
// grouped by two orthogonal dimensions (row_dimension × primary_dimension).
func (a *LedgerReportingAdapter) GetRevenueReport(
	ctx context.Context,
	req *revreportpb.RevenueReportRequest,
) (*revreportpb.RevenueReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildRevenueReportQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flat []flatRow
	for rows.Next() {
		var fr flatRow
		var rowID, colID sql.NullString
		if err := rows.Scan(
			&fr.RowKey,
			&rowID,
			&fr.ColKey,
			&colID,
			&fr.TotalRevenue,
			&fr.TransactionCount,
			&fr.TotalQuantity,
		); err != nil {
			return nil, err
		}
		if rowID.Valid {
			fr.RowID = rowID.String
		}
		if colID.Valid {
			fr.ColID = colID.String
		}
		flat = append(flat, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pivotFlatRows(flat, req), nil
}

// GetExpenditureReport executes a two-dimensional pivot SQL query to compute expenditures
// grouped by two orthogonal dimensions (row_dimension × primary_dimension).
func (a *LedgerReportingAdapter) GetExpenditureReport(
	ctx context.Context,
	req *expreportpb.ExpenditureReportRequest,
) (*expreportpb.ExpenditureReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildExpenditureReportQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flat []expenditureFlatRow
	for rows.Next() {
		var fr expenditureFlatRow
		var rowID, colID sql.NullString
		if err := rows.Scan(
			&fr.RowKey,
			&rowID,
			&fr.ColKey,
			&colID,
			&fr.TotalExpenditure,
			&fr.TransactionCount,
			&fr.TotalQuantity,
		); err != nil {
			return nil, err
		}
		if rowID.Valid {
			fr.RowID = rowID.String
		}
		if colID.Valid {
			fr.ColID = colID.String
		}
		flat = append(flat, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pivotFlatExpenditureRows(flat, req), nil
}

// GetDisbursementReport executes a two-dimensional pivot SQL query to compute disbursements
// grouped by two orthogonal dimensions (row_dimension x primary_dimension).
// Only paid/completed disbursements are included.
func (a *LedgerReportingAdapter) GetDisbursementReport(
	ctx context.Context,
	req *disbreportpb.DisbursementReportRequest,
) (*disbreportpb.DisbursementReportResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildDisbursementReportQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flat []disbursementFlatRow
	for rows.Next() {
		var fr disbursementFlatRow
		var rowID, colID sql.NullString
		if err := rows.Scan(
			&fr.RowKey,
			&rowID,
			&fr.ColKey,
			&colID,
			&fr.TotalDisbursement,
			&fr.TransactionCount,
			&fr.TotalQuantity,
		); err != nil {
			return nil, err
		}
		if rowID.Valid {
			fr.RowID = rowID.String
		}
		if colID.Valid {
			fr.ColID = colID.String
		}
		flat = append(flat, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pivotFlatDisbursementRows(flat, req), nil
}

// GetReceivablesAgingReport executes a CTE-based SQL query to compute outstanding
// receivables bucketed into 5 aging bands (current, 1-30, 31-60, 61-90, >90 days).
// Uses payment_date <= as_of_date (not status) for point-in-time accuracy.
func (a *LedgerReportingAdapter) GetReceivablesAgingReport(ctx context.Context, req *agingpb.ReceivablesAgingRequest) (*agingpb.ReceivablesAgingResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildReceivablesAgingQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agingRows, summaryBuckets, err := scanAgingRows(rows)
	if err != nil {
		return nil, err
	}

	summary := buildAgingSummary(agingRows, summaryBuckets, req)

	return &agingpb.ReceivablesAgingResponse{
		BucketLabels: []string{"Current", "1-30 Days", "31-60 Days", "61-90 Days", "Over 90 Days"},
		Rows:         agingRows,
		Summary:      summary,
		Success:      true,
	}, nil
}

// GetPayablesAgingReport executes a CTE-based SQL query to compute outstanding
// payables bucketed into 5 aging bands (current, 1-30, 31-60, 61-90, >90 days).
// Uses payment_date <= as_of_date (not status) for point-in-time accuracy.
func (a *LedgerReportingAdapter) GetPayablesAgingReport(ctx context.Context, req *payagingpb.PayablesAgingRequest) (*payagingpb.PayablesAgingResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildPayablesAgingQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agingRows, summaryBuckets, err := scanPayablesAgingRows(rows)
	if err != nil {
		return nil, err
	}

	summary := buildPayablesAgingSummary(agingRows, summaryBuckets, req)

	return &payagingpb.PayablesAgingResponse{
		BucketLabels: []string{"Current", "1-30 Days", "31-60 Days", "61-90 Days", "Over 90 Days"},
		Rows:         agingRows,
		Summary:      summary,
		Success:      true,
	}, nil
}

// GetCollectionSummaryReport executes a two-dimensional pivot SQL query to compute
// collection totals grouped by two orthogonal dimensions (row_dimension x primary_dimension).
// Filters by payment_date range for period selection.
func (a *LedgerReportingAdapter) GetCollectionSummaryReport(ctx context.Context, req *collsumpb.CollectionSummaryRequest) (*collsumpb.CollectionSummaryResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildCollectionSummaryQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flat []collectionFlatRow
	for rows.Next() {
		var fr collectionFlatRow
		var rowID, colID sql.NullString
		if err := rows.Scan(
			&fr.RowKey,
			&rowID,
			&fr.ColKey,
			&colID,
			&fr.TotalCollected,
			&fr.TransactionCount,
		); err != nil {
			return nil, err
		}
		if rowID.Valid {
			fr.RowID = rowID.String
		}
		if colID.Valid {
			fr.ColID = colID.String
		}
		flat = append(flat, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pivotFlatCollectionRows(flat, req), nil
}

// GetClientStatement executes a UNION ALL + window function query to produce
// a client statement with running balance across invoices and collections.
func (a *LedgerReportingAdapter) GetClientStatement(
	ctx context.Context,
	req *clientstmtpb.ClientStatementRequest,
) (*clientstmtpb.ClientStatementResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildClientStatementQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries, err := scanClientStatementEntries(rows)
	if err != nil {
		return nil, err
	}

	summary := buildClientStatementSummary(entries, req)

	// Fetch client name via separate query.
	nameQuery := buildClientNameQuery(a.tableConfig)
	var clientName sql.NullString
	_ = a.db.QueryRowContext(ctx, nameQuery, req.GetClientId()).Scan(&clientName)
	if clientName.Valid {
		summary.ClientName = clientName.String
	}

	return &clientstmtpb.ClientStatementResponse{
		Entries: entries,
		Summary: summary,
	}, nil
}

// GetSupplierStatement executes a UNION ALL query to produce a chronological
// supplier statement. Running balance is computed in Go after fetching rows.
func (a *LedgerReportingAdapter) GetSupplierStatement(
	ctx context.Context,
	req *suppstmtpb.SupplierStatementRequest,
) (*suppstmtpb.SupplierStatementResponse, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, args := buildSupplierStatementQuery(a.tableConfig, req, workspaceID)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raw []statementRow
	for rows.Next() {
		var r statementRow
		if err := rows.Scan(
			&r.TransactionDate, &r.TransactionType, &r.Reference,
			&r.Description, &r.BilledAmount, &r.PaidAmount,
			&r.SourceID, &r.Status,
		); err != nil {
			return nil, err
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	entries := buildStatementEntries(raw)
	summary := buildStatementSummary(entries, req)

	// Fetch supplier name via separate query.
	nameQuery := buildSupplierNameQuery(a.tableConfig)
	var supplierName sql.NullString
	_ = a.db.QueryRowContext(ctx, nameQuery, req.GetSupplierId()).Scan(&supplierName)
	if supplierName.Valid {
		summary.SupplierName = supplierName.String
	}

	return &suppstmtpb.SupplierStatementResponse{
		Entries: entries,
		Summary: summary,
		Success: true,
	}, nil
}

// GetSupplierBalances returns a map of supplier_id → outstanding centavo balance
// for all suppliers that have a non-zero outstanding balance.
// Suppliers with zero net balance are omitted (use 0 as the default for absent keys).
func (a *LedgerReportingAdapter) GetSupplierBalances(ctx context.Context) (map[string]int64, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, _ := buildSupplierBalancesQuery(a.tableConfig)
	rows, err := a.db.QueryContext(ctx, query, nilIfEmpty(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var supplierID string
		var outstanding int64
		if err := rows.Scan(&supplierID, &outstanding); err != nil {
			return nil, err
		}
		result[supplierID] = outstanding
	}
	return result, rows.Err()
}

// GetClientBalances returns a map of client_id → outstanding centavo balance
// for all clients that have a non-zero outstanding balance.
// Clients with zero net balance are omitted (use 0 as the default for absent keys).
func (a *LedgerReportingAdapter) GetClientBalances(ctx context.Context) (map[string]int64, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query, _ := buildClientBalancesQuery(a.tableConfig)
	rows, err := a.db.QueryContext(ctx, query, nilIfEmpty(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var clientID string
		var outstanding int64
		if err := rows.Scan(&clientID, &outstanding); err != nil {
			return nil, err
		}
		result[clientID] = outstanding
	}
	return result, rows.Err()
}

// ListRevenue returns revenue records, optionally filtered by date range.
func (a *LedgerReportingAdapter) ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	query := `SELECT id, reference_number, status, total_amount, currency,
		COALESCE(customer_first_name, '') || ' ' || COALESCE(customer_last_name, '') AS customer_name,
		COALESCE(notes, '') AS notes,
		COALESCE(location_name, '') AS location_name,
		created_at
		FROM ` + a.tableConfig.Revenue

	var args []any
	var conditions []string
	paramIdx := 1

	if start != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", paramIdx))
		args = append(args, *start)
		paramIdx++
	}
	if end != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", paramIdx))
		args = append(args, *end)
		paramIdx++
	}
	if workspaceID != "" {
		conditions = append(conditions, fmt.Sprintf("workspace_id = $%d", paramIdx))
		args = append(args, workspaceID)
		paramIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// ListExpenses returns expenditure records of type "expense", optionally filtered by date range.
func (a *LedgerReportingAdapter) ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	table := a.tableConfig.Expenditure
	if table == "" {
		return nil, nil
	}
	supplierTable := a.tableConfig.Supplier
	if supplierTable == "" {
		supplierTable = "supplier"
	}
	// expenditure has no vendor_name / category columns; join supplier and the
	// expenditure_category_id FK to recover those denormalized values for the
	// reporting consumer (which still expects vendor_name + category aliases).
	query := `SELECT e.id, e.reference_number,
		COALESCE(s.name, '') AS vendor_name,
		COALESCE(e.expenditure_category_id, '') AS category,
		e.status, e.total_amount, e.currency,
		COALESCE(e.expenditure_date_string, '') AS expenditure_date,
		COALESCE(e.notes, '') AS notes
		FROM ` + table + ` e
		LEFT JOIN ` + supplierTable + ` s ON s.id = e.supplier_id
		WHERE e.expenditure_type = 'expense'`

	var args []any
	paramIdx := 1

	if start != nil {
		query += fmt.Sprintf(" AND e.date_created >= $%d", paramIdx)
		args = append(args, *start)
		paramIdx++
	}
	if end != nil {
		query += fmt.Sprintf(" AND e.date_created <= $%d", paramIdx)
		args = append(args, *end)
		paramIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND e.workspace_id = $%d", paramIdx)
		args = append(args, workspaceID)
		paramIdx++
	}

	query += " ORDER BY e.date_created DESC LIMIT 200"

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// scanToMaps converts sql.Rows into a slice of column→value maps.
func scanToMaps(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = values[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// computeSummary aggregates line item totals into a report summary.
func computeSummary(items []*reportpb.GrossProfitLineItem, req *reportpb.GrossProfitReportRequest) *reportpb.GrossProfitSummary {
	s := &reportpb.GrossProfitSummary{}
	for _, item := range items {
		s.TotalRevenue += item.TotalRevenue
		s.TotalDiscount += item.TotalDiscount
		s.NetRevenue += item.NetRevenue
		s.TotalCogs += item.CostOfGoodsSold
		s.TotalGrossProfit += item.GrossProfit
		s.TotalUnitsSold += item.UnitsSold
		s.TotalTransactions += item.TransactionCount
	}
	if s.NetRevenue > 0 {
		s.OverallMargin = float64(s.TotalGrossProfit) / float64(s.NetRevenue) * 100
	}
	if req.StartDate != nil {
		sd := req.GetStartDate()
		s.StartDate = &sd
	}
	if req.EndDate != nil {
		ed := req.GetEndDate()
		s.EndDate = &ed
	}
	if req.Currency != nil {
		s.Currency = req.GetCurrency()
	}
	return s
}
