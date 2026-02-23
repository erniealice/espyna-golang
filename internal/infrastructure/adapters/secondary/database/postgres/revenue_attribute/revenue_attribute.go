//go:build postgresql

package revenue_attribute

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
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "revenue_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRevenueAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresRevenueAttributeRepository implements revenue_attribute CRUD operations using PostgreSQL
type PostgresRevenueAttributeRepository struct {
	revenueattributepb.UnimplementedRevenueAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRevenueAttributeRepository creates a new PostgreSQL revenue attribute repository
func NewPostgresRevenueAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) revenueattributepb.RevenueAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "revenue_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRevenueAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRevenueAttribute creates a new revenue attribute
func (r *PostgresRevenueAttributeRepository) CreateRevenueAttribute(ctx context.Context, req *revenueattributepb.CreateRevenueAttributeRequest) (*revenueattributepb.CreateRevenueAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue attribute data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.CreateRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// ReadRevenueAttribute retrieves a revenue attribute by ID
func (r *PostgresRevenueAttributeRepository) ReadRevenueAttribute(ctx context.Context, req *revenueattributepb.ReadRevenueAttributeRequest) (*revenueattributepb.ReadRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.ReadRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// UpdateRevenueAttribute updates a revenue attribute
func (r *PostgresRevenueAttributeRepository) UpdateRevenueAttribute(ctx context.Context, req *revenueattributepb.UpdateRevenueAttributeRequest) (*revenueattributepb.UpdateRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenueattributepb.UpdateRevenueAttributeResponse{
		Data: []*revenueattributepb.RevenueAttribute{attr},
	}, nil
}

// DeleteRevenueAttribute deletes a revenue attribute (soft delete)
func (r *PostgresRevenueAttributeRepository) DeleteRevenueAttribute(ctx context.Context, req *revenueattributepb.DeleteRevenueAttributeRequest) (*revenueattributepb.DeleteRevenueAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete revenue attribute: %w", err)
	}

	return &revenueattributepb.DeleteRevenueAttributeResponse{
		Success: true,
	}, nil
}

// ListRevenueAttributes lists revenue attributes with optional filters
func (r *PostgresRevenueAttributeRepository) ListRevenueAttributes(ctx context.Context, req *revenueattributepb.ListRevenueAttributesRequest) (*revenueattributepb.ListRevenueAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenue attributes: %w", err)
	}

	var attrs []*revenueattributepb.RevenueAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal revenue_attribute row: %v", err)
			continue
		}

		attr := &revenueattributepb.RevenueAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
			log.Printf("WARN: protojson unmarshal revenue_attribute: %v", err)
			continue
		}
		attrs = append(attrs, attr)
	}

	return &revenueattributepb.ListRevenueAttributesResponse{
		Data: attrs,
	}, nil
}

// GetRevenueAttributeListPageData retrieves revenue attributes with pagination, sorting, and search using CTE
// Joins with revenue table for enriched display
func (r *PostgresRevenueAttributeRepository) GetRevenueAttributeListPageData(
	ctx context.Context,
	req *revenueattributepb.GetRevenueAttributeListPageDataRequest,
) (*revenueattributepb.GetRevenueAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue attribute list page data request is required")
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

	sortField := "ra.date_created"
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
				ra.id,
				ra.revenue_id,
				ra.attribute_id,
				ra.value,
				ra.date_created,
				ra.date_modified,
				ra.active,
				COALESCE(rv.name, '') as revenue_name
			FROM revenue_attribute ra
			LEFT JOIN revenue rv ON ra.revenue_id = rv.id AND rv.active = true
			WHERE ra.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ra.value ILIKE $1 OR
			       rv.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query revenue attribute list page data: %w", err)
	}
	defer rows.Close()

	var attrs []*revenueattributepb.RevenueAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			revenueID    string
			attributeID  string
			value        string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			revenueName  string
			total        int64
		)

		err := rows.Scan(
			&id,
			&revenueID,
			&attributeID,
			&value,
			&dateCreated,
			&dateModified,
			&active,
			&revenueName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue attribute row: %w", err)
		}

		totalCount = total

		attr := &revenueattributepb.RevenueAttribute{
			Id:          id,
			RevenueId:   revenueID,
			AttributeId: attributeID,
			Value:       value,
			Active:      active,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			attr.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			attr.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			attr.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			attr.DateModifiedString = &dmStr
		}

		attrs = append(attrs, attr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue attribute rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenueattributepb.GetRevenueAttributeListPageDataResponse{
		RevenueAttributeList: attrs,
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

// GetRevenueAttributeItemPageData retrieves a single revenue attribute with enriched data
func (r *PostgresRevenueAttributeRepository) GetRevenueAttributeItemPageData(
	ctx context.Context,
	req *revenueattributepb.GetRevenueAttributeItemPageDataRequest,
) (*revenueattributepb.GetRevenueAttributeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue attribute item page data request is required")
	}
	if req.RevenueAttributeId == "" {
		return nil, fmt.Errorf("revenue attribute ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				ra.id,
				ra.revenue_id,
				ra.attribute_id,
				ra.value,
				ra.date_created,
				ra.date_modified,
				ra.active,
				COALESCE(rv.name, '') as revenue_name
			FROM revenue_attribute ra
			LEFT JOIN revenue rv ON ra.revenue_id = rv.id AND rv.active = true
			WHERE ra.id = $1 AND ra.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RevenueAttributeId)

	var (
		id           string
		revenueID    string
		attributeID  string
		value        string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		revenueName  string
	)

	err := row.Scan(
		&id,
		&revenueID,
		&attributeID,
		&value,
		&dateCreated,
		&dateModified,
		&active,
		&revenueName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue attribute with ID '%s' not found", req.RevenueAttributeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue attribute item page data: %w", err)
	}

	attr := &revenueattributepb.RevenueAttribute{
		Id:          id,
		RevenueId:   revenueID,
		AttributeId: attributeID,
		Value:       value,
		Active:      active,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		attr.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		attr.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		attr.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		attr.DateModifiedString = &dmStr
	}

	return &revenueattributepb.GetRevenueAttributeItemPageDataResponse{
		RevenueAttribute: attr,
		Success:          true,
	}, nil
}

// NewRevenueAttributeRepository creates a new PostgreSQL revenue attribute repository (old-style constructor)
func NewRevenueAttributeRepository(db *sql.DB, tableName string) revenueattributepb.RevenueAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresRevenueAttributeRepository(dbOps, tableName)
}
