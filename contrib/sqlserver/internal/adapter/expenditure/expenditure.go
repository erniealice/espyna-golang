//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Expenditure, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expenditure repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenditureRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenditureRepository implements expenditure CRUD using SQL Server.
//
// SQL Server differences from the postgres gold standard:
//   - Placeholders: $N → @pN.
//   - ILIKE → LIKE (default CI collation).
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET m ROWS FETCH NEXT n ROWS ONLY.
//   - active = true → active = 1 (BIT literal).
//   - COUNT(*) OVER () retained (SQL Server 2017+).
type SQLServerExpenditureRepository struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerExpenditureRepository creates a new SQL Server expenditure repository.
func NewSQLServerExpenditureRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerExpenditureRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenditure creates a new expenditure record.
func (r *SQLServerExpenditureRepository) CreateExpenditure(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// SQL Server stores timestamp columns as DATETIME2; convert proto int64 millis.
	convertMillisToTime(data, "expenditureDate")
	convertMillisToTime(data, "dueDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	expenditure := &expenditurepb.Expenditure{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expenditure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenditurepb.CreateExpenditureResponse{
		Success: true,
		Data:    []*expenditurepb.Expenditure{expenditure},
	}, nil
}

// ReadExpenditure retrieves an expenditure record by ID.
func (r *SQLServerExpenditureRepository) ReadExpenditure(ctx context.Context, req *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	expenditure := &expenditurepb.Expenditure{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expenditure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenditurepb.ReadExpenditureResponse{
		Success: true,
		Data:    []*expenditurepb.Expenditure{expenditure},
	}, nil
}

// UpdateExpenditure updates an expenditure record.
func (r *SQLServerExpenditureRepository) UpdateExpenditure(ctx context.Context, req *expenditurepb.UpdateExpenditureRequest) (*expenditurepb.UpdateExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "expenditureDate")
	convertMillisToTime(data, "dueDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expenditure: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	expenditure := &expenditurepb.Expenditure{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expenditure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenditurepb.UpdateExpenditureResponse{
		Success: true,
		Data:    []*expenditurepb.Expenditure{expenditure},
	}, nil
}

// DeleteExpenditure soft-deletes an expenditure record.
func (r *SQLServerExpenditureRepository) DeleteExpenditure(ctx context.Context, req *expenditurepb.DeleteExpenditureRequest) (*expenditurepb.DeleteExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure: %w", err)
	}
	return &expenditurepb.DeleteExpenditureResponse{Success: true}, nil
}

// ListExpenditures lists expenditure records with optional filters.
func (r *SQLServerExpenditureRepository) ListExpenditures(ctx context.Context, req *expenditurepb.ListExpendituresRequest) (*expenditurepb.ListExpendituresResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditures: %w", err)
	}
	var expenditures []*expenditurepb.Expenditure
	for _, result := range listResult.Data {
		sqlserverCore.ConvertMillisToDateStr(result, "due_date")
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure row: %v", err)
			continue
		}
		expenditure := &expenditurepb.Expenditure{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expenditure); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure: %v", err)
			continue
		}
		expenditures = append(expenditures, expenditure)
	}
	return &expenditurepb.ListExpendituresResponse{
		Success: true,
		Data:    expenditures,
	}, nil
}

// GetExpenditureListPageData retrieves expenditures with pagination, filtering, sorting, and search.
// Joins with supplier and location tables for enriched display.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - @p1 = workspaceID, @p2 = searchPattern, @p3 = offset, @p4 = limit.
//   - ($1::text IS NULL OR ...) → (@p1 IS NULL OR @p1 = ”).
//   - ILIKE → LIKE.
//   - active = true → active = 1.
//   - Pagination: ORDER BY … OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY.
//   - COUNT(*) OVER () retained.
func (r *SQLServerExpenditureRepository) GetExpenditureListPageData(
	ctx context.Context,
	req *expenditurepb.GetExpenditureListPageDataRequest,
) (*expenditurepb.GetExpenditureListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get expenditure list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	sortField := "ex.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build filter/search WHERE clauses (@p1 is reserved for workspace_id, start at @p2).
	// BuildFilterWhere emits @pN placeholders and LIKE (not ILIKE) for SQL Server.
	searchFields := []string{"ex.name", "ex.reference_number", "ex.status", "s.name"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)
	_ = searchPattern // use BuildFilterWhere for search if req.Search set; otherwise pass nil

	whereSQL := "WHERE ex.active = 1 AND (@p1 IS NULL OR @p1 = '' OR ex.workspace_id = @p1)"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	// CTE mirrors the postgres gold standard with SQL Server dialect changes.
	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ex.id,
				ex.date_created,
				ex.date_modified,
				ex.active,
				ex.name,
				ex.expenditure_type,
				ex.supplier_id AS supplier_id_primary,
				ex.expenditure_date,
				ex.expenditure_date_string,
				ex.total_amount,
				ex.currency,
				ex.status,
				ex.reference_number,
				ex.notes,
				ex.expenditure_category_id,
				ex.location_id,
				ex.payment_terms,
				ex.due_date,
				ex.approved_by,
				ex.purchase_order_id,
				ex.supplier_id,
				ex.run_id,
				COALESCE(s.name, '') as vendor_name,
				COALESCE(l.name, '') as location_name,
				COUNT(*) OVER() AS total
			FROM expenditure ex
			LEFT JOIN supplier s ON ex.supplier_id = s.id AND s.active = 1
			LEFT JOIN location l ON ex.location_id = l.id AND l.active = 1
			%s
		)
		SELECT * FROM enriched
		ORDER BY `+sortField+` `+sortOrder+`
		OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, offsetIdx, limitIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenditure list page data: %w", err)
	}
	defer rows.Close()

	var expenditures []*expenditurepb.Expenditure
	var totalCount int64

	for rows.Next() {
		var (
			id                    string
			dateCreated           time.Time
			dateModified          time.Time
			active                bool
			name                  string
			expenditureType       *string
			vendorID              *string
			expenditureDate       *time.Time
			expenditureDateString *string
			totalAmount           int64
			currency              *string
			status                *string
			referenceNumber       *string
			notes                 *string
			expenditureCategoryID *string
			locationID            *string
			paymentTerms          *string
			dueDate               *string
			approvedBy            *string
			purchaseOrderID       *string
			supplierID            *string
			runID                 *string
			vendorName            string
			locationName          string
			total                 int64
		)
		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name, &expenditureType,
			&vendorID, &expenditureDate, &expenditureDateString, &totalAmount, &currency,
			&status, &referenceNumber, &notes, &expenditureCategoryID, &locationID,
			&paymentTerms, &dueDate, &approvedBy, &purchaseOrderID, &supplierID, &runID,
			&vendorName, &locationName, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expenditure row: %w", err)
		}
		totalCount = total

		expenditure := &expenditurepb.Expenditure{
			Id:                    id,
			Active:                active,
			Name:                  name,
			TotalAmount:           totalAmount,
			ReferenceNumber:       referenceNumber,
			Notes:                 notes,
			ExpenditureCategoryId: expenditureCategoryID,
			PaymentTerms:          paymentTerms,
			ApprovedBy:            approvedBy,
		}
		if expenditureType != nil {
			expenditure.ExpenditureType = *expenditureType
		}
		if vendorID != nil {
			expenditure.SupplierId = vendorID
		}
		if locationID != nil {
			expenditure.LocationId = *locationID
		}
		if currency != nil {
			expenditure.Currency = *currency
		}
		if status != nil {
			expenditure.Status = *status
		}
		if expenditureDateString != nil {
			expenditure.ExpenditureDateString = expenditureDateString
		}
		if expenditureDate != nil && !expenditureDate.IsZero() {
			ts := expenditureDate.UnixMilli()
			expenditure.ExpenditureDate = &ts
		}
		if dueDate != nil {
			expenditure.DueDate = dueDate
		}
		if purchaseOrderID != nil {
			expenditure.PurchaseOrderId = purchaseOrderID
		}
		if supplierID != nil {
			expenditure.SupplierId = supplierID
		}
		if runID != nil {
			expenditure.RunId = runID
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			expenditure.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			expenditure.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			expenditure.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			expenditure.DateModifiedString = &dmStr
		}
		expenditures = append(expenditures, expenditure)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenditure rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &expenditurepb.GetExpenditureListPageDataResponse{
		ExpenditureList: expenditures,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetExpenditureItemPageData retrieves a single expenditure with enriched data.
// CRITICAL: Always filters by workspace_id for multi-tenancy via context.
//
// SQL Server differences: @p1 → id; active = 1; TOP 1 instead of LIMIT 1.
func (r *SQLServerExpenditureRepository) GetExpenditureItemPageData(
	ctx context.Context,
	req *expenditurepb.GetExpenditureItemPageDataRequest,
) (*expenditurepb.GetExpenditureItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get expenditure item page data request is required")
	}
	if req.ExpenditureId == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				ex.id,
				ex.date_created,
				ex.date_modified,
				ex.active,
				ex.name,
				ex.expenditure_type,
				ex.supplier_id AS supplier_id_primary,
				ex.expenditure_date,
				ex.expenditure_date_string,
				ex.total_amount,
				ex.currency,
				ex.status,
				ex.reference_number,
				ex.notes,
				ex.expenditure_category_id,
				ex.location_id,
				ex.payment_terms,
				ex.due_date,
				ex.approved_by,
				ex.purchase_order_id,
				ex.supplier_id,
				ex.run_id,
				COALESCE(s.name, '') as vendor_name,
				COALESCE(l.name, '') as location_name
			FROM expenditure ex
			LEFT JOIN supplier s ON ex.supplier_id = s.id AND s.active = 1
			LEFT JOIN location l ON ex.location_id = l.id AND l.active = 1
			WHERE ex.id = @p1 AND ex.active = 1
		)
		SELECT TOP 1 * FROM enriched;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.ExpenditureId)

	var (
		id                    string
		dateCreated           time.Time
		dateModified          time.Time
		active                bool
		name                  string
		expenditureType       *string
		vendorID              *string
		expenditureDate       *time.Time
		expenditureDateString *string
		totalAmount           int64
		currency              *string
		status                *string
		referenceNumber       *string
		notes                 *string
		expenditureCategoryID *string
		locationID            *string
		paymentTerms          *string
		dueDate               *string
		approvedBy            *string
		purchaseOrderID       *string
		supplierID            *string
		runID                 *string
		vendorName            string
		locationName          string
	)
	if err := row.Scan(
		&id, &dateCreated, &dateModified, &active, &name, &expenditureType,
		&vendorID, &expenditureDate, &expenditureDateString, &totalAmount, &currency,
		&status, &referenceNumber, &notes, &expenditureCategoryID, &locationID,
		&paymentTerms, &dueDate, &approvedBy, &purchaseOrderID, &supplierID, &runID,
		&vendorName, &locationName,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("expenditure with ID '%s' not found", req.ExpenditureId)
		}
		return nil, fmt.Errorf("failed to query expenditure item page data: %w", err)
	}

	expenditure := &expenditurepb.Expenditure{
		Id:                    id,
		Active:                active,
		Name:                  name,
		TotalAmount:           totalAmount,
		ReferenceNumber:       referenceNumber,
		Notes:                 notes,
		ExpenditureCategoryId: expenditureCategoryID,
		PaymentTerms:          paymentTerms,
		ApprovedBy:            approvedBy,
	}
	if expenditureType != nil {
		expenditure.ExpenditureType = *expenditureType
	}
	if vendorID != nil {
		expenditure.SupplierId = vendorID
	}
	if locationID != nil {
		expenditure.LocationId = *locationID
	}
	if currency != nil {
		expenditure.Currency = *currency
	}
	if status != nil {
		expenditure.Status = *status
	}
	if expenditureDateString != nil {
		expenditure.ExpenditureDateString = expenditureDateString
	}
	if expenditureDate != nil && !expenditureDate.IsZero() {
		ts := expenditureDate.UnixMilli()
		expenditure.ExpenditureDate = &ts
	}
	if dueDate != nil {
		expenditure.DueDate = dueDate
	}
	if purchaseOrderID != nil {
		expenditure.PurchaseOrderId = purchaseOrderID
	}
	if supplierID != nil {
		expenditure.SupplierId = supplierID
	}
	if runID != nil {
		expenditure.RunId = runID
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		expenditure.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		expenditure.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		expenditure.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		expenditure.DateModifiedString = &dmStr
	}

	return &expenditurepb.GetExpenditureItemPageDataResponse{
		Expenditure: expenditure,
		Success:     true,
	}, nil
}

// NewExpenditureRepository creates a new SQL Server expenditure repository (old-style constructor).
func NewExpenditureRepository(db *sql.DB, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerExpenditureRepository(dbOps, tableName)
}

// convertMillisToTime converts a proto millis-epoch value (JSON string or float64)
// in data[key] to time.Time so SQL Server DATETIME2 columns accept it.
func convertMillisToTime(data map[string]any, key string) {
	v, ok := data[key]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[key] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[key] = time.UnixMilli(int64(val))
		}
	}
}
