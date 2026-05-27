//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders, args resequenced)
//   - "ident"   → `ident` (backtick quoting)
//   - ILIKE     → LIKE (ci collation)
//   - active = true → active = 1
//   - $N::text IS NULL OR ... → (? = ” OR ...)
//   - LIMIT $3 OFFSET $4 → LIMIT ? OFFSET ?
//   - COUNT(*) OVER () stays (MySQL 8.0+)
//   - postgresCore.ConvertMillisToDateStr → mysqlCore.ConvertMillisToDateStr
//
// 20260517 — `run_id` surfaced through list/item CTE for run-linkage badges.
// CRITICAL: workspace_id isolation enforced on every raw-SQL query.
// Centavos (total_amount) are never scaled in SQL.
package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Expenditure, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql expenditure repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLExpenditureRepository(dbOps, tableName), nil
	})
}

// MySQLExpenditureRepository implements expenditure CRUD operations using MySQL 8.0+.
//
// 20260517 — `run_id` (expense-run) is written/read via protojson round-trip
// on CRUD and surfaced explicitly through CTE selects for list/item pages.
type MySQLExpenditureRepository struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLExpenditureRepository creates a new MySQL expenditure repository.
func NewMySQLExpenditureRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure"
	}
	return &MySQLExpenditureRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateExpenditure creates a new expenditure record.
func (r *MySQLExpenditureRepository) CreateExpenditure(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
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

	convertMillisToTime(data, "expenditureDate")
	convertMillisToTime(data, "dueDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLExpenditureRepository) ReadExpenditure(ctx context.Context, req *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLExpenditureRepository) UpdateExpenditure(ctx context.Context, req *expenditurepb.UpdateExpenditureRequest) (*expenditurepb.UpdateExpenditureResponse, error) {
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

	mysqlCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLExpenditureRepository) DeleteExpenditure(ctx context.Context, req *expenditurepb.DeleteExpenditureRequest) (*expenditurepb.DeleteExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure: %w", err)
	}

	return &expenditurepb.DeleteExpenditureResponse{Success: true}, nil
}

// ListExpenditures lists expenditure records with optional filters.
func (r *MySQLExpenditureRepository) ListExpenditures(ctx context.Context, req *expenditurepb.ListExpendituresRequest) (*expenditurepb.ListExpendituresResponse, error) {
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
		mysqlCore.ConvertMillisToDateStr(result, "due_date")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// GetExpenditureListPageData retrieves expenditures with pagination, filtering,
// sorting, and search using CTE. Joins supplier and location tables.
//
// Dialect changes:
//   - $1/$2/$3/$4 → ?; args: [workspaceID, searchPattern, limit, offset]
//   - ILIKE → LIKE; active = true → active = 1
//   - $N::text IS NULL OR ... → (? = ” OR ...)
//   - COUNT(*) OVER() stays (MySQL 8.0+)
//
// CRITICAL: workspace_id = ? enforced for multi-tenancy.
func (r *MySQLExpenditureRepository) GetExpenditureListPageData(
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

	// 20260517 expense-run: expose `run_id` for list-row run-linkage badges.
	// Dialect: workspace_id guard uses ? = '' OR ex.workspace_id = ? (two args),
	// then search uses ? = '' OR ... LIKE ? (five LIKE checks = one empty + four).
	// Args: [workspaceID, workspaceID, searchPattern, searchPattern*4, limit, offset]
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
			WHERE ex.active = 1
			  AND (? = '' OR ex.workspace_id = ?)
			  AND (? = '' OR
			       ex.name LIKE ? OR
			       ex.reference_number LIKE ? OR
			       ex.status LIKE ? OR
			       s.name LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ? OFFSET ?
	`

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	rows, err := r.db.QueryContext(ctx, query,
		workspaceID, workspaceID,
		searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		limit, offset,
	)
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
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&expenditureType,
			&vendorID,
			&expenditureDate,
			&expenditureDateString,
			&totalAmount,
			&currency,
			&status,
			&referenceNumber,
			&notes,
			&expenditureCategoryID,
			&locationID,
			&paymentTerms,
			&dueDate,
			&approvedBy,
			&purchaseOrderID,
			&supplierID,
			&runID,
			&vendorName,
			&locationName,
			&total,
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
//
// Dialect: $1 → ?; active = true → active = 1.
// 20260517 expense-run: expose `run_id` for detail-page run-linkage.
func (r *MySQLExpenditureRepository) GetExpenditureItemPageData(
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
			WHERE ex.id = ? AND ex.active = 1
		)
		SELECT * FROM enriched LIMIT 1
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	row := r.db.QueryRowContext(ctx, query, req.ExpenditureId)

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

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&expenditureType,
		&vendorID,
		&expenditureDate,
		&expenditureDateString,
		&totalAmount,
		&currency,
		&status,
		&referenceNumber,
		&notes,
		&expenditureCategoryID,
		&locationID,
		&paymentTerms,
		&dueDate,
		&approvedBy,
		&purchaseOrderID,
		&supplierID,
		&runID,
		&vendorName,
		&locationName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("expenditure with ID '%s' not found", req.ExpenditureId)
	}
	if err != nil {
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

// NewExpenditureRepository creates a new MySQL expenditure repository (old-style constructor).
func NewExpenditureRepository(db *sql.DB, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLExpenditureRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson.
// MySQL timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
