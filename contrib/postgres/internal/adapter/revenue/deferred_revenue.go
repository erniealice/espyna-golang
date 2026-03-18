
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DeferredRevenue, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres deferred revenue repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresDeferredRevenueRepository(dbOps, tableName), nil
	})
}

// PostgresDeferredRevenueRepository implements deferred revenue CRUD operations using PostgreSQL
type PostgresDeferredRevenueRepository struct {
	deferredrevenuepb.UnimplementedDeferredRevenueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresDeferredRevenueRepository creates a new PostgreSQL deferred revenue repository
func NewPostgresDeferredRevenueRepository(dbOps interfaces.DatabaseOperation, tableName string) deferredrevenuepb.DeferredRevenueDomainServiceServer {
	if tableName == "" {
		tableName = "deferred_revenue"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresDeferredRevenueRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDeferredRevenue creates a new deferred revenue record
func (r *PostgresDeferredRevenueRepository) CreateDeferredRevenue(ctx context.Context, req *deferredrevenuepb.CreateDeferredRevenueRequest) (*deferredrevenuepb.CreateDeferredRevenueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("deferred revenue data is required")
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
	convertMillisToTime(data, "startDate", "start_date")
	convertMillisToTime(data, "endDate", "end_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	deferredRevenue := &deferredrevenuepb.DeferredRevenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, deferredRevenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &deferredrevenuepb.CreateDeferredRevenueResponse{
		Success: true,
		Data:    []*deferredrevenuepb.DeferredRevenue{deferredRevenue},
	}, nil
}

// ReadDeferredRevenue retrieves a deferred revenue record by ID
func (r *PostgresDeferredRevenueRepository) ReadDeferredRevenue(ctx context.Context, req *deferredrevenuepb.ReadDeferredRevenueRequest) (*deferredrevenuepb.ReadDeferredRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	deferredRevenue := &deferredrevenuepb.DeferredRevenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, deferredRevenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &deferredrevenuepb.ReadDeferredRevenueResponse{
		Success: true,
		Data:    []*deferredrevenuepb.DeferredRevenue{deferredRevenue},
	}, nil
}

// UpdateDeferredRevenue updates a deferred revenue record
func (r *PostgresDeferredRevenueRepository) UpdateDeferredRevenue(ctx context.Context, req *deferredrevenuepb.UpdateDeferredRevenueRequest) (*deferredrevenuepb.UpdateDeferredRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
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
	convertMillisToTime(data, "startDate", "start_date")
	convertMillisToTime(data, "endDate", "end_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	deferredRevenue := &deferredrevenuepb.DeferredRevenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, deferredRevenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &deferredrevenuepb.UpdateDeferredRevenueResponse{
		Success: true,
		Data:    []*deferredrevenuepb.DeferredRevenue{deferredRevenue},
	}, nil
}

// DeleteDeferredRevenue deletes a deferred revenue record (soft delete)
func (r *PostgresDeferredRevenueRepository) DeleteDeferredRevenue(ctx context.Context, req *deferredrevenuepb.DeleteDeferredRevenueRequest) (*deferredrevenuepb.DeleteDeferredRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete deferred revenue: %w", err)
	}

	return &deferredrevenuepb.DeleteDeferredRevenueResponse{
		Success: true,
	}, nil
}

// ListDeferredRevenues lists deferred revenue records with optional filters
func (r *PostgresDeferredRevenueRepository) ListDeferredRevenues(ctx context.Context, req *deferredrevenuepb.ListDeferredRevenuesRequest) (*deferredrevenuepb.ListDeferredRevenuesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list deferred revenues: %w", err)
	}

	var deferredRevenues []*deferredrevenuepb.DeferredRevenue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal deferred revenue row: %v", err)
			continue
		}

		deferredRevenue := &deferredrevenuepb.DeferredRevenue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, deferredRevenue); err != nil {
			log.Printf("WARN: protojson unmarshal deferred revenue: %v", err)
			continue
		}
		deferredRevenues = append(deferredRevenues, deferredRevenue)
	}

	return &deferredrevenuepb.ListDeferredRevenuesResponse{
		Success: true,
		Data:    deferredRevenues,
	}, nil
}

// GetDeferredRevenueListPageData retrieves deferred revenues with pagination, filtering, sorting, and search using CTE
// TODO: Add enriched joins with account table for GL account names once CoA is in place
func (r *PostgresDeferredRevenueRepository) GetDeferredRevenueListPageData(
	ctx context.Context,
	req *deferredrevenuepb.GetDeferredRevenueListPageDataRequest,
) (*deferredrevenuepb.GetDeferredRevenueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get deferred revenue list page data request is required")
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

	sortField := "dr.date_created"
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
				dr.id,
				dr.date_created,
				dr.date_modified,
				dr.active,
				dr.description,
				dr.customer_name,
				dr.total_amount,
				dr.recognized_amount,
				dr.remaining_amount,
				dr.start_date,
				dr.start_date_string,
				dr.end_date,
				dr.end_date_string,
				dr.recognition_months,
				dr.status,
				dr.liability_account_id,
				dr.revenue_account_id
			FROM ` + r.tableName + ` dr
			WHERE dr.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       dr.description ILIKE $1 OR
			       dr.customer_name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query deferred revenue list page data: %w", err)
	}
	defer rows.Close()

	var deferredRevenues []*deferredrevenuepb.DeferredRevenue
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        int64
			dateModified       int64
			active             bool
			description        string
			customerName       *string
			totalAmount        float64
			recognizedAmount   float64
			remainingAmount    float64
			startDate          *int64
			startDateString    *string
			endDate            *int64
			endDateString      *string
			recognitionMonths  int32
			statusStr          string
			liabilityAccountID *string
			revenueAccountID   *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&description,
			&customerName,
			&totalAmount,
			&recognizedAmount,
			&remainingAmount,
			&startDate,
			&startDateString,
			&endDate,
			&endDateString,
			&recognitionMonths,
			&statusStr,
			&liabilityAccountID,
			&revenueAccountID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deferred revenue row: %w", err)
		}

		totalCount = total

		deferredRevenue := &deferredrevenuepb.DeferredRevenue{
			Id:                 id,
			Active:             active,
			Description:        description,
			CustomerName:       customerName,
			TotalAmount:        totalAmount,
			RecognizedAmount:   recognizedAmount,
			RemainingAmount:    remainingAmount,
			RecognitionMonths:  recognitionMonths,
			LiabilityAccountId: liabilityAccountID,
			RevenueAccountId:   revenueAccountID,
			StartDateString:    startDateString,
			EndDateString:      endDateString,
		}

		if val, ok := deferredrevenuepb.DeferredRevenueStatus_value[statusStr]; ok {
			deferredRevenue.Status = deferredrevenuepb.DeferredRevenueStatus(val)
		}

		if startDate != nil && *startDate > 0 {
			deferredRevenue.StartDate = *startDate
		}
		if endDate != nil && *endDate > 0 {
			deferredRevenue.EndDate = *endDate
		}
		if dateCreated > 0 {
			deferredRevenue.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			deferredRevenue.DateModified = &dateModified
		}

		deferredRevenues = append(deferredRevenues, deferredRevenue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deferred revenue rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &deferredrevenuepb.GetDeferredRevenueListPageDataResponse{
		DeferredRevenueList: deferredRevenues,
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

// GetDeferredRevenueItemPageData retrieves a single deferred revenue with enriched data
// TODO: Add CTE query with joined account details once CoA is in place
func (r *PostgresDeferredRevenueRepository) GetDeferredRevenueItemPageData(
	ctx context.Context,
	req *deferredrevenuepb.GetDeferredRevenueItemPageDataRequest,
) (*deferredrevenuepb.GetDeferredRevenueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get deferred revenue item page data request is required")
	}
	if req.DeferredRevenueId == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.DeferredRevenueId)
	if err != nil {
		return nil, fmt.Errorf("failed to read deferred revenue '%s': %w", req.DeferredRevenueId, err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	deferredRevenue := &deferredrevenuepb.DeferredRevenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, deferredRevenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &deferredrevenuepb.GetDeferredRevenueItemPageDataResponse{
		DeferredRevenue: deferredRevenue,
		Success:         true,
	}, nil
}

// NewDeferredRevenueRepository creates a new PostgreSQL deferred revenue repository (old-style constructor)
func NewDeferredRevenueRepository(db *sql.DB, tableName string) deferredrevenuepb.DeferredRevenueDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresDeferredRevenueRepository(dbOps, tableName)
}
