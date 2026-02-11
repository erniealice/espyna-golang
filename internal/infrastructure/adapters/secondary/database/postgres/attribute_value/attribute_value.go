//go:build postgres

package attribute_value

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "attribute_value", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres attribute_value repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresAttributeValueRepository(dbOps, tableName), nil
	})
}

// PostgresAttributeValueRepository implements attribute_value CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_attribute_value_active ON attribute_value(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_attribute_value_attribute_id ON attribute_value(attribute_id) - FK lookup on attribute_id
//   - CREATE INDEX idx_attribute_value_value ON attribute_value(value) - Search on value field
//   - CREATE INDEX idx_attribute_value_sort_order ON attribute_value(sort_order) - Sorting by sort_order
//   - CREATE INDEX idx_attribute_value_date_created ON attribute_value(date_created DESC) - Default sorting
type PostgresAttributeValueRepository struct {
	commonpb.UnimplementedAttributeValueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresAttributeValueRepository creates a new PostgreSQL attribute value repository
func NewPostgresAttributeValueRepository(dbOps interfaces.DatabaseOperation, tableName string) commonpb.AttributeValueDomainServiceServer {
	if tableName == "" {
		tableName = "attribute_value" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresAttributeValueRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAttributeValue creates a new attribute value using common PostgreSQL operations
func (r *PostgresAttributeValueRepository) CreateAttributeValue(ctx context.Context, req *commonpb.CreateAttributeValueRequest) (*commonpb.CreateAttributeValueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("attribute value data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute value: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attributeValue := &commonpb.AttributeValue{}
	if err := protojson.Unmarshal(resultJSON, attributeValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.CreateAttributeValueResponse{
		Data: []*commonpb.AttributeValue{attributeValue},
	}, nil
}

// ReadAttributeValue retrieves an attribute value using common PostgreSQL operations
func (r *PostgresAttributeValueRepository) ReadAttributeValue(ctx context.Context, req *commonpb.ReadAttributeValueRequest) (*commonpb.ReadAttributeValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute value ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read attribute value: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attributeValue := &commonpb.AttributeValue{}
	if err := protojson.Unmarshal(resultJSON, attributeValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.ReadAttributeValueResponse{
		Data: []*commonpb.AttributeValue{attributeValue},
	}, nil
}

// UpdateAttributeValue updates an attribute value using common PostgreSQL operations
func (r *PostgresAttributeValueRepository) UpdateAttributeValue(ctx context.Context, req *commonpb.UpdateAttributeValueRequest) (*commonpb.UpdateAttributeValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute value ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update attribute value: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attributeValue := &commonpb.AttributeValue{}
	if err := protojson.Unmarshal(resultJSON, attributeValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &commonpb.UpdateAttributeValueResponse{
		Data: []*commonpb.AttributeValue{attributeValue},
	}, nil
}

// DeleteAttributeValue deletes an attribute value using common PostgreSQL operations
func (r *PostgresAttributeValueRepository) DeleteAttributeValue(ctx context.Context, req *commonpb.DeleteAttributeValueRequest) (*commonpb.DeleteAttributeValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("attribute value ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete attribute value: %w", err)
	}

	return &commonpb.DeleteAttributeValueResponse{
		Success: true,
	}, nil
}

// ListAttributeValues lists attribute values using common PostgreSQL operations
func (r *PostgresAttributeValueRepository) ListAttributeValues(ctx context.Context, req *commonpb.ListAttributeValuesRequest) (*commonpb.ListAttributeValuesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list attribute values: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var attributeValues []*commonpb.AttributeValue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		attributeValue := &commonpb.AttributeValue{}
		if err := protojson.Unmarshal(resultJSON, attributeValue); err != nil {
			// Log error and continue with next item
			continue
		}
		attributeValues = append(attributeValues, attributeValue)
	}

	return &commonpb.ListAttributeValuesResponse{
		Data: attributeValues,
	}, nil
}

// GetAttributeValueListPageData retrieves attribute values with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the attribute table to include the parent attribute name
func (r *PostgresAttributeValueRepository) GetAttributeValueListPageData(
	ctx context.Context,
	req *commonpb.GetAttributeValueListPageDataRequest,
) (*commonpb.GetAttributeValueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get attribute value list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "av.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with attribute join for parent attribute name
	query := `
		WITH enriched AS (
			SELECT
				av.id,
				av.date_created,
				av.date_modified,
				av.active,
				av.attribute_id,
				av.value,
				av.sort_order,
				COALESCE(a.name, '') as attribute_name
			FROM attribute_value av
			LEFT JOIN attribute a ON av.attribute_id = a.id AND a.active = true
			WHERE av.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       av.value ILIKE $1 OR
			       a.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query attribute value list page data: %w", err)
	}
	defer rows.Close()

	var attributeValues []*commonpb.AttributeValue
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			attributeID   string
			value         string
			sortOrder     int32
			attributeName string
			total         int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&attributeID,
			&value,
			&sortOrder,
			&attributeName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attribute value row: %w", err)
		}

		totalCount = total

		attributeValue := &commonpb.AttributeValue{
			Id:          id,
			Active:      active,
			AttributeId: attributeID,
			Value:       value,
			SortOrder:   sortOrder,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			attributeValue.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			attributeValue.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			attributeValue.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			attributeValue.DateModifiedString = &dmStr
		}

		// Note: attributeName is available from the join but not directly mapped
		// to the AttributeValue protobuf. Could be populated via the Attribute field
		// if needed for frontend display.

		attributeValues = append(attributeValues, attributeValue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attribute value rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &commonpb.GetAttributeValueListPageDataResponse{
		AttributeValueList: attributeValues,
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

// GetAttributeValueItemPageData retrieves a single attribute value with enhanced item page data using CTE
// This method joins with the attribute table for the parent attribute reference
func (r *PostgresAttributeValueRepository) GetAttributeValueItemPageData(
	ctx context.Context,
	req *commonpb.GetAttributeValueItemPageDataRequest,
) (*commonpb.GetAttributeValueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get attribute value item page data request is required")
	}
	if req.AttributeValueId == "" {
		return nil, fmt.Errorf("attribute value ID is required")
	}

	// CTE Query - Single round-trip with attribute join
	query := `
		WITH enriched AS (
			SELECT
				av.id,
				av.date_created,
				av.date_modified,
				av.active,
				av.attribute_id,
				av.value,
				av.sort_order,
				COALESCE(a.name, '') as attribute_name
			FROM attribute_value av
			LEFT JOIN attribute a ON av.attribute_id = a.id AND a.active = true
			WHERE av.id = $1 AND av.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.AttributeValueId)

	var (
		id            string
		dateCreated   time.Time
		dateModified  time.Time
		active        bool
		attributeID   string
		value         string
		sortOrder     int32
		attributeName string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&attributeID,
		&value,
		&sortOrder,
		&attributeName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attribute value with ID '%s' not found", req.AttributeValueId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query attribute value item page data: %w", err)
	}

	attributeValue := &commonpb.AttributeValue{
		Id:          id,
		Active:      active,
		AttributeId: attributeID,
		Value:       value,
		SortOrder:   sortOrder,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		attributeValue.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		attributeValue.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		attributeValue.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		attributeValue.DateModifiedString = &dmStr
	}

	// Note: attributeName is available from the join but not directly mapped.

	return &commonpb.GetAttributeValueItemPageDataResponse{
		AttributeValue: attributeValue,
		Success:        true,
	}, nil
}

// NewAttributeValueRepository creates a new PostgreSQL attribute value repository (old-style constructor)
func NewAttributeValueRepository(db *sql.DB, tableName string) commonpb.AttributeValueDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresAttributeValueRepository(dbOps, tableName)
}
