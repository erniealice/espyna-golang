//go:build postgresql

package revenue

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
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "revenue", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRevenueRepository(dbOps, tableName), nil
	})
}

// PostgresRevenueRepository implements revenue CRUD operations using PostgreSQL
type PostgresRevenueRepository struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRevenueRepository creates a new PostgreSQL revenue repository
func NewPostgresRevenueRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuepb.RevenueDomainServiceServer {
	if tableName == "" {
		tableName = "revenue"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRevenueRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRevenue creates a new revenue record
func (r *PostgresRevenueRepository) CreateRevenue(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue data is required")
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
	convertMillisToTime(data, "revenueDate", "revenue_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.CreateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// ReadRevenue retrieves a revenue record by ID
func (r *PostgresRevenueRepository) ReadRevenue(ctx context.Context, req *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.ReadRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// UpdateRevenue updates a revenue record
func (r *PostgresRevenueRepository) UpdateRevenue(ctx context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
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
	convertMillisToTime(data, "revenueDate", "revenue_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.UpdateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// DeleteRevenue deletes a revenue record (soft delete)
func (r *PostgresRevenueRepository) DeleteRevenue(ctx context.Context, req *revenuepb.DeleteRevenueRequest) (*revenuepb.DeleteRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete revenue: %w", err)
	}

	return &revenuepb.DeleteRevenueResponse{
		Success: true,
	}, nil
}

// ListRevenues lists revenue records with optional filters
func (r *PostgresRevenueRepository) ListRevenues(ctx context.Context, req *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenues: %w", err)
	}

	var revenues []*revenuepb.Revenue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal revenue row: %v", err)
			continue
		}

		revenue := &revenuepb.Revenue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
			log.Printf("WARN: protojson unmarshal revenue: %v", err)
			continue
		}
		revenues = append(revenues, revenue)
	}

	return &revenuepb.ListRevenuesResponse{
		Success: true,
		Data:    revenues,
	}, nil
}

// GetRevenueListPageData retrieves revenues with pagination, filtering, sorting, and search using CTE
// Joins with client and location tables for enriched display
func (r *PostgresRevenueRepository) GetRevenueListPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueListPageDataRequest,
) (*revenuepb.GetRevenueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue list page data request is required")
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

	sortField := "rv.date_created"
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
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name
			FROM revenue rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = true
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = true
			WHERE rv.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       rv.name ILIKE $1 OR
			       rv.reference_number ILIKE $1 OR
			       rv.status ILIKE $1 OR
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
		return nil, fmt.Errorf("failed to query revenue list page data: %w", err)
	}
	defer rows.Close()

	var revenues []*revenuepb.Revenue
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			name              string
			clientID          *string
			revenueDate       *time.Time
			revenueDateString *string
			totalAmount       float64
			currency          *string
			status            *string
			referenceNumber   *string
			notes             *string
			revenueCategoryID *string
			locationID        *string
			clientName        string
			locationName      string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&clientID,
			&revenueDate,
			&revenueDateString,
			&totalAmount,
			&currency,
			&status,
			&referenceNumber,
			&notes,
			&revenueCategoryID,
			&locationID,
			&clientName,
			&locationName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue row: %w", err)
		}

		totalCount = total

		revenue := &revenuepb.Revenue{
			Id:              id,
			Active:          active,
			Name:            name,
			TotalAmount:     totalAmount,
			ReferenceNumber: referenceNumber,
			Notes:           notes,
			RevenueCategoryId: revenueCategoryID,
		}

		if clientID != nil {
			revenue.ClientId = *clientID
		}
		if locationID != nil {
			revenue.LocationId = *locationID
		}
		if currency != nil {
			revenue.Currency = *currency
		}
		if status != nil {
			revenue.Status = *status
		}
		if revenueDateString != nil {
			revenue.RevenueDateString = revenueDateString
		}
		if revenueDate != nil && !revenueDate.IsZero() {
			ts := revenueDate.UnixMilli()
			revenue.RevenueDate = &ts
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			revenue.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			revenue.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			revenue.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			revenue.DateModifiedString = &dmStr
		}

		revenues = append(revenues, revenue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuepb.GetRevenueListPageDataResponse{
		RevenueList: revenues,
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

// GetRevenueItemPageData retrieves a single revenue with enriched data using CTE
func (r *PostgresRevenueRepository) GetRevenueItemPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueItemPageDataRequest,
) (*revenuepb.GetRevenueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue item page data request is required")
	}
	if req.RevenueId == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name
			FROM revenue rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = true
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = true
			WHERE rv.id = $1 AND rv.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RevenueId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		name              string
		clientID          *string
		revenueDate       *time.Time
		revenueDateString *string
		totalAmount       float64
		currency          *string
		status            *string
		referenceNumber   *string
		notes             *string
		revenueCategoryID *string
		locationID        *string
		clientName        string
		locationName      string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&clientID,
		&revenueDate,
		&revenueDateString,
		&totalAmount,
		&currency,
		&status,
		&referenceNumber,
		&notes,
		&revenueCategoryID,
		&locationID,
		&clientName,
		&locationName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue with ID '%s' not found", req.RevenueId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue item page data: %w", err)
	}

	revenue := &revenuepb.Revenue{
		Id:              id,
		Active:          active,
		Name:            name,
		TotalAmount:     totalAmount,
		ReferenceNumber: referenceNumber,
		Notes:           notes,
		RevenueCategoryId: revenueCategoryID,
	}

	if clientID != nil {
		revenue.ClientId = *clientID
	}
	if locationID != nil {
		revenue.LocationId = *locationID
	}
	if currency != nil {
		revenue.Currency = *currency
	}
	if status != nil {
		revenue.Status = *status
	}
	if revenueDateString != nil {
		revenue.RevenueDateString = revenueDateString
	}
	if revenueDate != nil && !revenueDate.IsZero() {
		ts := revenueDate.UnixMilli()
		revenue.RevenueDate = &ts
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		revenue.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		revenue.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		revenue.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		revenue.DateModifiedString = &dmStr
	}

	return &revenuepb.GetRevenueItemPageDataResponse{
		Revenue: revenue,
		Success: true,
	}, nil
}

// NewRevenueRepository creates a new PostgreSQL revenue repository (old-style constructor)
func NewRevenueRepository(db *sql.DB, tableName string) revenuepb.RevenueDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresRevenueRepository(dbOps, tableName)
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
