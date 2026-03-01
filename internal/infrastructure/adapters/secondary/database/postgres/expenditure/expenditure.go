//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "expenditure", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expenditure repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresExpenditureRepository(dbOps, tableName), nil
	})
}

// PostgresExpenditureRepository implements expenditure CRUD operations using PostgreSQL
type PostgresExpenditureRepository struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresExpenditureRepository creates a new PostgreSQL expenditure repository
func NewPostgresExpenditureRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresExpenditureRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenditure creates a new expenditure record
func (r *PostgresExpenditureRepository) CreateExpenditure(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "expenditureDate", "expenditure_date")
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// ReadExpenditure retrieves an expenditure record by ID
func (r *PostgresExpenditureRepository) ReadExpenditure(ctx context.Context, req *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// UpdateExpenditure updates an expenditure record
func (r *PostgresExpenditureRepository) UpdateExpenditure(ctx context.Context, req *expenditurepb.UpdateExpenditureRequest) (*expenditurepb.UpdateExpenditureResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "expenditureDate", "expenditure_date")
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expenditure: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// DeleteExpenditure deletes an expenditure record (soft delete)
func (r *PostgresExpenditureRepository) DeleteExpenditure(ctx context.Context, req *expenditurepb.DeleteExpenditureRequest) (*expenditurepb.DeleteExpenditureResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expenditure: %w", err)
	}

	return &expenditurepb.DeleteExpenditureResponse{
		Success: true,
	}, nil
}

// ListExpenditures lists expenditure records with optional filters
func (r *PostgresExpenditureRepository) ListExpenditures(ctx context.Context, req *expenditurepb.ListExpendituresRequest) (*expenditurepb.ListExpendituresResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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

// GetExpenditureListPageData retrieves expenditures with pagination, filtering, sorting, and search using CTE
// Joins with client (as vendor) and location tables for enriched display
func (r *PostgresExpenditureRepository) GetExpenditureListPageData(
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

	query := `
		WITH enriched AS (
			SELECT
				ex.id,
				ex.date_created,
				ex.date_modified,
				ex.active,
				ex.name,
				ex.expenditure_type,
				ex.vendor_id,
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
				COALESCE(c.name, '') as vendor_name,
				COALESCE(l.name, '') as location_name
			FROM expenditure ex
			LEFT JOIN client c ON ex.vendor_id = c.id AND c.active = true
			LEFT JOIN location l ON ex.location_id = l.id AND l.active = true
			WHERE ex.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ex.name ILIKE $1 OR
			       ex.reference_number ILIKE $1 OR
			       ex.status ILIKE $1 OR
			       c.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
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
			totalAmount           float64
			currency              *string
			status                *string
			referenceNumber       *string
			notes                 *string
			expenditureCategoryID *string
			locationID            *string
			paymentTerms          *string
			dueDate               *time.Time
			approvedBy            *string
			vendorName            string
			locationName          string
			total                 int64
		)

		err := rows.Scan(
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
			&vendorName,
			&locationName,
			&total,
		)
		if err != nil {
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
			expenditure.VendorId = *vendorID
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
		if dueDate != nil && !dueDate.IsZero() {
			ts := dueDate.UnixMilli()
			expenditure.DueDate = &ts
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

// GetExpenditureItemPageData retrieves a single expenditure with enriched data using CTE
func (r *PostgresExpenditureRepository) GetExpenditureItemPageData(
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
				ex.vendor_id,
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
				COALESCE(c.name, '') as vendor_name,
				COALESCE(l.name, '') as location_name
			FROM expenditure ex
			LEFT JOIN client c ON ex.vendor_id = c.id AND c.active = true
			LEFT JOIN location l ON ex.location_id = l.id AND l.active = true
			WHERE ex.id = $1 AND ex.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

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
		totalAmount           float64
		currency              *string
		status                *string
		referenceNumber       *string
		notes                 *string
		expenditureCategoryID *string
		locationID            *string
		paymentTerms          *string
		dueDate               *time.Time
		approvedBy            *string
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
		expenditure.VendorId = *vendorID
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
	if dueDate != nil && !dueDate.IsZero() {
		ts := dueDate.UnixMilli()
		expenditure.DueDate = &ts
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

// NewExpenditureRepository creates a new PostgreSQL expenditure repository (old-style constructor)
func NewExpenditureRepository(db *sql.DB, tableName string) expenditurepb.ExpenditureDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresExpenditureRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
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
