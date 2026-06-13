//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
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
	ExpenditureLineItem  string
	ExpenditureCategory  string
	Supplier             string
	ProductCollection    string
	Collection           string
	Line                 string
	LocationArea         string
	SupplierCategory     string
	TreasuryDisbursement string
	DisbursementMethod   string
	Client               string
	ClientCategory       string
	Category             string
	TreasuryCollection   string
	CollectionMethod     string
	PaymentTerm          string
}

// SQLServerLedgerReportingAdapter implements LedgerReportingService using SQL Server.
// SQL Server differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …)
//   - Identifier quoting: [ident] (not "ident")
//   - ILIKE → LIKE (SQL Server default CI collation)
//   - LIMIT n OFFSET m → OFFSET m ROWS FETCH NEXT n ROWS ONLY (requires ORDER BY)
//   - SUM(x) FILTER (WHERE c) → SUM(CASE WHEN c THEN x END)
//   - postgres type-casts (::date, ::timestamptz) → CAST(… AS date/datetime2)
//   - TO_CHAR / date_trunc → FORMAT / DATETRUNC (SQL Server 2022+) or equivalents
type SQLServerLedgerReportingAdapter struct {
	db          *sql.DB
	tableConfig TableConfig
}

// NewSQLServerLedgerReportingAdapter creates a new SQL Server ledger reporting adapter.
func NewSQLServerLedgerReportingAdapter(db *sql.DB, config TableConfig) *SQLServerLedgerReportingAdapter {
	return &SQLServerLedgerReportingAdapter{db: db, tableConfig: config}
}

// GetGrossProfitReport executes a CTE-based SQL query to compute gross profit
// grouped by the requested dimension (product, location, category, or period).
func (a *SQLServerLedgerReportingAdapter) GetGrossProfitReport(
	ctx context.Context,
	req *reportpb.GrossProfitReportRequest,
) (*reportpb.GrossProfitReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetRevenueReport(
	ctx context.Context,
	req *revreportpb.RevenueReportRequest,
) (*revreportpb.RevenueReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetExpenditureReport(
	ctx context.Context,
	req *expreportpb.ExpenditureReportRequest,
) (*expreportpb.ExpenditureReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetDisbursementReport(
	ctx context.Context,
	req *disbreportpb.DisbursementReportRequest,
) (*disbreportpb.DisbursementReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetReceivablesAgingReport(ctx context.Context, req *agingpb.ReceivablesAgingRequest) (*agingpb.ReceivablesAgingResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetPayablesAgingReport(ctx context.Context, req *payagingpb.PayablesAgingRequest) (*payagingpb.PayablesAgingResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
// collection totals grouped by two orthogonal dimensions.
func (a *SQLServerLedgerReportingAdapter) GetCollectionSummaryReport(ctx context.Context, req *collsumpb.CollectionSummaryRequest) (*collsumpb.CollectionSummaryResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetClientStatement(
	ctx context.Context,
	req *clientstmtpb.ClientStatementRequest,
) (*clientstmtpb.ClientStatementResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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
func (a *SQLServerLedgerReportingAdapter) GetSupplierStatement(
	ctx context.Context,
	req *suppstmtpb.SupplierStatementRequest,
) (*suppstmtpb.SupplierStatementResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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

// GetSupplierBalances returns a map of supplier_id → outstanding centavo balance.
func (a *SQLServerLedgerReportingAdapter) GetSupplierBalances(ctx context.Context) (map[string]int64, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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

// GetClientBalances returns a map of client_id → outstanding centavo balance.
func (a *SQLServerLedgerReportingAdapter) GetClientBalances(ctx context.Context) (map[string]int64, error) {
	workspaceID := identity.Must(ctx).WorkspaceID
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

// GetCashBookReport executes a UNION ALL query over revenue and expenditure tables
// to produce a simple chronological cash book showing receipts and disbursements.
// Amounts are stored in pesos in the DB and multiplied by 100 to produce centavos.
//
// SQL Server differences from the postgres gold standard:
//   - $1 → @p1; $2 → @p2
//   - TO_CHAR(date_created, 'YYYY-MM-DD') → CONVERT(varchar, date_created, 23)
//   - LIMIT $2 → ORDER BY tx_date DESC, reference OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY
func (a *SQLServerLedgerReportingAdapter) GetCashBookReport(
	ctx context.Context,
	req *reportpb.CashBookReportRequest,
) (*reportpb.CashBookReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID

	limit := int32(200)
	if req.Limit != nil && req.GetLimit() > 0 {
		limit = req.GetLimit()
	}

	query := `
		SELECT tx_date, description, reference, tx_type, amount
		FROM (
			SELECT
				CONVERT(varchar, date_created, 23) AS tx_date,
				COALESCE(NULLIF(LTRIM(RTRIM(name)), ''), 'Collection') AS description,
				COALESCE(NULLIF(reference_number, ''), '-') AS reference,
				'Receipt' AS tx_type,
				CAST(total_amount * 100 AS bigint) AS amount
			FROM ` + a.tableConfig.Revenue + `
			WHERE status NOT IN ('cancelled', 'draft')
			AND (@p1 IS NULL OR workspace_id = @p1)

			UNION ALL

			SELECT
				CONVERT(varchar, expenditure_date, 23) AS tx_date,
				COALESCE(NULLIF(name, ''), 'Payment') AS description,
				COALESCE(NULLIF(reference_number, ''), '-') AS reference,
				CASE WHEN expenditure_type = 'purchase' THEN 'Purchase' ELSE 'Expense' END AS tx_type,
				CAST(total_amount * 100 AS bigint) AS amount
			FROM ` + a.tableConfig.Expenditure + `
			WHERE status NOT IN ('cancelled', 'draft')
			AND (@p1 IS NULL OR workspace_id = @p1)
		) combined
		ORDER BY tx_date DESC, reference
		OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY
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
// payables per supplier bucketed into 5 aging bands.
// Amounts are stored in pesos in the DB and multiplied by 100 to produce centavos.
//
// SQL Server differences:
//   - $1 → @p1
//   - CURRENT_DATE → CAST(SYSUTCDATETIME() AS date)
//   - ::date casts → CAST(… AS date)
//   - SUM(CASE WHEN …) is already used in the postgres version for these aging bands
func (a *SQLServerLedgerReportingAdapter) GetSimplePayablesAgingReport(
	ctx context.Context,
	req *reportpb.PayablesAgingReportRequest,
) (*reportpb.PayablesAgingReportResponse, error) {
	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		WITH outstanding AS (
			SELECT
				e.id,
				COALESCE(NULLIF(LTRIM(RTRIM(s.company_name)), ''), NULLIF(LTRIM(RTRIM(e.name)), ''), 'Unknown') AS supplier_name,
				e.total_amount - COALESCE(paid.total_paid, 0) AS outstanding_amount,
				DATEDIFF(day, COALESCE(CAST(e.due_date AS date), CAST(e.expenditure_date AS date)), CAST(SYSUTCDATETIME() AS date)) AS days_overdue
			FROM ` + a.tableConfig.Expenditure + ` e
			LEFT JOIN ` + a.tableConfig.Supplier + ` s ON s.id = e.supplier_id
			LEFT JOIN (
				SELECT d.expenditure_id, SUM(d.amount) AS total_paid
				FROM ` + a.tableConfig.TreasuryDisbursement + ` d
				WHERE d.active = 1 AND d.status IN ('paid', 'completed')
				GROUP BY d.expenditure_id
			) paid ON paid.expenditure_id = e.id
			WHERE e.active = 1
			  AND e.expenditure_type IN ('purchase', 'expense')
			  AND e.status NOT IN ('paid', 'cancelled')
			  AND e.total_amount - COALESCE(paid.total_paid, 0) > 0
			  AND (@p1 IS NULL OR e.workspace_id = @p1)
		)
		SELECT
			supplier_name,
			CAST(COALESCE(SUM(CASE WHEN days_overdue <= 0 THEN outstanding_amount ELSE 0 END), 0) * 100 AS bigint) AS current_amt,
			CAST(COALESCE(SUM(CASE WHEN days_overdue BETWEEN 1 AND 30 THEN outstanding_amount ELSE 0 END), 0) * 100 AS bigint) AS days_30,
			CAST(COALESCE(SUM(CASE WHEN days_overdue BETWEEN 31 AND 60 THEN outstanding_amount ELSE 0 END), 0) * 100 AS bigint) AS days_60,
			CAST(COALESCE(SUM(CASE WHEN days_overdue BETWEEN 61 AND 90 THEN outstanding_amount ELSE 0 END), 0) * 100 AS bigint) AS days_90,
			CAST(COALESCE(SUM(CASE WHEN days_overdue > 90 THEN outstanding_amount ELSE 0 END), 0) * 100 AS bigint) AS over_90,
			CAST(COALESCE(SUM(outstanding_amount), 0) * 100 AS bigint) AS total
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

// ListRevenue returns revenue records, optionally filtered by date range.
// SQL Server differences: @pN placeholders; OFFSET 0 ROWS FETCH NEXT 200 ROWS ONLY.
func (a *SQLServerLedgerReportingAdapter) ListRevenue(ctx context.Context, start, end *time.Time) ([]map[string]any, error) {
	workspaceID := identity.Must(ctx).WorkspaceID

	query := `SELECT id, reference_number, status, total_amount, currency,
		COALESCE(customer_first_name, '') + ' ' + COALESCE(customer_last_name, '') AS customer_name,
		COALESCE(notes, '') AS notes,
		COALESCE(location_name, '') AS location_name,
		created_at
		FROM ` + a.tableConfig.Revenue

	var args []any
	var conditions []string
	paramIdx := 1

	if start != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= @p%d", paramIdx))
		args = append(args, *start)
		paramIdx++
	}
	if end != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= @p%d", paramIdx))
		args = append(args, *end)
		paramIdx++
	}
	if workspaceID != "" {
		conditions = append(conditions, fmt.Sprintf("workspace_id = @p%d", paramIdx))
		args = append(args, workspaceID)
		paramIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC OFFSET 0 ROWS FETCH NEXT 200 ROWS ONLY"

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// ListExpenses returns expenditure records of type "expense", optionally filtered by date range.
func (a *SQLServerLedgerReportingAdapter) ListExpenses(ctx context.Context, start, end *time.Time) ([]map[string]any, error) {
	workspaceID := identity.Must(ctx).WorkspaceID

	table := a.tableConfig.Expenditure
	if table == "" {
		return nil, nil
	}
	supplierTable := a.tableConfig.Supplier
	if supplierTable == "" {
		supplierTable = "supplier"
	}

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
		query += fmt.Sprintf(" AND e.date_created >= @p%d", paramIdx)
		args = append(args, *start)
		paramIdx++
	}
	if end != nil {
		query += fmt.Sprintf(" AND e.date_created <= @p%d", paramIdx)
		args = append(args, *end)
		paramIdx++
	}
	if workspaceID != "" {
		query += fmt.Sprintf(" AND e.workspace_id = @p%d", paramIdx)
		args = append(args, workspaceID)
		paramIdx++
	}

	query += " ORDER BY e.date_created DESC OFFSET 0 ROWS FETCH NEXT 200 ROWS ONLY"

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

// nilIfEmpty returns nil for empty strings so they bind as SQL NULL.
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
