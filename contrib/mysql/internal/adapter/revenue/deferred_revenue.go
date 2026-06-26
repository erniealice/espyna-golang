//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders)
//   - ILIKE → LIKE (ci collation)
//   - $1::text IS NULL OR $1::text = ” → ? IS NULL OR ? = ” pattern handled
//     by passing searchPattern twice (empty string is treated as no-search by
//     MySQL's LIKE ” fallback); for simplicity we use the same pattern as
//     postgres: pass "" as searchPattern when no query, and LIKE checks
//     collapse to false for ” in MySQL — so we wrap with a length guard.
//   - LIMIT $2 OFFSET $3 → LIMIT ? OFFSET ?
//   - active = true → active = 1
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.DeferredRevenue, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql deferred_revenue repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLDeferredRevenueRepository(dbOps, tableName), nil
	})
}

// MySQLDeferredRevenueRepository implements deferred_revenue CRUD using MySQL 8.0+.
type MySQLDeferredRevenueRepository struct {
	deferredrevenuepb.UnimplementedDeferredRevenueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLDeferredRevenueRepository creates a new MySQL deferred revenue repository.
func NewMySQLDeferredRevenueRepository(dbOps interfaces.DatabaseOperation, tableName string) deferredrevenuepb.DeferredRevenueDomainServiceServer {
	if tableName == "" {
		tableName = "deferred_revenue"
	}
	return &MySQLDeferredRevenueRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateDeferredRevenue creates a new deferred revenue record.
func (r *MySQLDeferredRevenueRepository) CreateDeferredRevenue(ctx context.Context, req *deferredrevenuepb.CreateDeferredRevenueRequest) (*deferredrevenuepb.CreateDeferredRevenueResponse, error) {
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

	convertMillisToTime(data, "startDate")
	convertMillisToTime(data, "endDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// ReadDeferredRevenue retrieves a deferred revenue record by ID.
func (r *MySQLDeferredRevenueRepository) ReadDeferredRevenue(ctx context.Context, req *deferredrevenuepb.ReadDeferredRevenueRequest) (*deferredrevenuepb.ReadDeferredRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// UpdateDeferredRevenue updates a deferred revenue record.
func (r *MySQLDeferredRevenueRepository) UpdateDeferredRevenue(ctx context.Context, req *deferredrevenuepb.UpdateDeferredRevenueRequest) (*deferredrevenuepb.UpdateDeferredRevenueResponse, error) {
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

	convertMillisToTime(data, "startDate")
	convertMillisToTime(data, "endDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update deferred revenue: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteDeferredRevenue soft-deletes a deferred revenue record.
func (r *MySQLDeferredRevenueRepository) DeleteDeferredRevenue(ctx context.Context, req *deferredrevenuepb.DeleteDeferredRevenueRequest) (*deferredrevenuepb.DeleteDeferredRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("deferred revenue ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete deferred revenue: %w", err)
	}

	return &deferredrevenuepb.DeleteDeferredRevenueResponse{Success: true}, nil
}

// ListDeferredRevenues lists deferred revenue records with optional filters.
func (r *MySQLDeferredRevenueRepository) ListDeferredRevenues(ctx context.Context, req *deferredrevenuepb.ListDeferredRevenuesRequest) (*deferredrevenuepb.ListDeferredRevenuesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal deferred_revenue row: %v", err)
			continue
		}
		dr := &deferredrevenuepb.DeferredRevenue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, dr); err != nil {
			log.Printf("WARN: protojson unmarshal deferred_revenue: %v", err)
			continue
		}
		deferredRevenues = append(deferredRevenues, dr)
	}

	return &deferredrevenuepb.ListDeferredRevenuesResponse{
		Success: true,
		Data:    deferredRevenues,
	}, nil
}

// GetDeferredRevenueListPageData retrieves deferred revenues with pagination,
// filtering, sorting, and search using CTE.
//
// Dialect changes: $1/$2/$3 → ?; ILIKE → LIKE; active = true → active = 1;
// counted CTE preserved (COUNT(*) OVER() valid MySQL 8.0+).
func (r *MySQLDeferredRevenueRepository) GetDeferredRevenueListPageData(
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

	// Dialect: $1::text IS NULL OR ... ILIKE $1 →
	// (? = '' OR dr.description LIKE ? OR dr.customer_name LIKE ?)
	// Pass searchPattern three times: one for the empty-check and two for LIKE.
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
				dr.end_date,
				dr.recognition_months,
				dr.status,
				dr.liability_account_id,
				dr.revenue_account_id
			FROM deferred_revenue dr
			WHERE dr.active = 1
			  AND (? = '' OR dr.description LIKE ? OR dr.customer_name LIKE ?)
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

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	// Args: [searchPattern (empty check), searchPattern (LIKE desc), searchPattern (LIKE customer), limit, offset]
	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, limit, offset)
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
			totalAmount        int64
			recognizedAmount   int64
			remainingAmount    int64
			startDate          *string
			endDate            *string
			recognitionMonths  int32
			statusStr          string
			liabilityAccountID *string
			revenueAccountID   *string
			total              int64
		)

		if err := rows.Scan(
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
			&endDate,
			&recognitionMonths,
			&statusStr,
			&liabilityAccountID,
			&revenueAccountID,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deferred revenue row: %w", err)
		}

		totalCount = total

		dr := &deferredrevenuepb.DeferredRevenue{
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
		}

		if val, ok := deferredrevenuepb.DeferredRevenueStatus_value[statusStr]; ok {
			dr.Status = deferredrevenuepb.DeferredRevenueStatus(val)
		}
		if startDate != nil {
			dr.StartDate = *startDate
		}
		if endDate != nil {
			dr.EndDate = *endDate
		}
		if dateCreated > 0 {
			dr.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			dr.DateModified = &dateModified
		}

		deferredRevenues = append(deferredRevenues, dr)
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

// GetDeferredRevenueItemPageData retrieves a single deferred revenue with enriched data.
func (r *MySQLDeferredRevenueRepository) GetDeferredRevenueItemPageData(
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// NewDeferredRevenueRepository creates a new MySQL deferred revenue repository (old-style constructor).
func NewDeferredRevenueRepository(db *sql.DB, tableName string) deferredrevenuepb.DeferredRevenueDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLDeferredRevenueRepository(dbOps, tableName)
}
