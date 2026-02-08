//go:build postgres

package product

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
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresProductAttributeRepository implements product_attribute CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_attribute_active ON product_attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_attribute_product_id ON product_attribute(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_product_attribute_key ON product_attribute(key) - Search on key field
//   - CREATE INDEX idx_product_attribute_value ON product_attribute(value) - Search on value field
//   - CREATE INDEX idx_product_attribute_date_created ON product_attribute(date_created DESC) - Default sorting
type PostgresProductAttributeRepository struct {
	productattributepb.UnimplementedProductAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductAttributeRepository creates a new PostgreSQL product attribute repository
func NewPostgresProductAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) productattributepb.ProductAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "product_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductAttribute creates a new product attribute using common PostgreSQL operations
func (r *PostgresProductAttributeRepository) CreateProductAttribute(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product attribute data is required")
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
		return nil, fmt.Errorf("failed to create product attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productAttribute := &productattributepb.ProductAttribute{}
	if err := protojson.Unmarshal(resultJSON, productAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productattributepb.CreateProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{productAttribute},
	}, nil
}

// ReadProductAttribute retrieves a product attribute using common PostgreSQL operations
func (r *PostgresProductAttributeRepository) ReadProductAttribute(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) (*productattributepb.ReadProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productAttribute := &productattributepb.ProductAttribute{}
	if err := protojson.Unmarshal(resultJSON, productAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productattributepb.ReadProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{productAttribute},
	}, nil
}

// UpdateProductAttribute updates a product attribute using common PostgreSQL operations
func (r *PostgresProductAttributeRepository) UpdateProductAttribute(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
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
		return nil, fmt.Errorf("failed to update product attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productAttribute := &productattributepb.ProductAttribute{}
	if err := protojson.Unmarshal(resultJSON, productAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productattributepb.UpdateProductAttributeResponse{
		Data: []*productattributepb.ProductAttribute{productAttribute},
	}, nil
}

// DeleteProductAttribute deletes a product attribute using common PostgreSQL operations
func (r *PostgresProductAttributeRepository) DeleteProductAttribute(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product attribute: %w", err)
	}

	return &productattributepb.DeleteProductAttributeResponse{
		Success: true,
	}, nil
}

// ListProductAttributes lists product attributes using common PostgreSQL operations
func (r *PostgresProductAttributeRepository) ListProductAttributes(ctx context.Context, req *productattributepb.ListProductAttributesRequest) (*productattributepb.ListProductAttributesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list product attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productAttributes []*productattributepb.ProductAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		productAttribute := &productattributepb.ProductAttribute{}
		if err := protojson.Unmarshal(resultJSON, productAttribute); err != nil {
			// Log error and continue with next item
			continue
		}
		productAttributes = append(productAttributes, productAttribute)
	}

	return &productattributepb.ListProductAttributesResponse{
		Data: productAttributes,
	}, nil
}

// GetProductAttributeListPageData retrieves product attributes with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresProductAttributeRepository) GetProductAttributeListPageData(
	ctx context.Context,
	req *productattributepb.GetProductAttributeListPageDataRequest,
) (*productattributepb.GetProductAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
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
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Attribute table pattern with attribute_id/value search
	query := `
		WITH enriched AS (
			SELECT
				pa.id,
				pa.product_id,
				pa.attribute_id,
				pa.value,
				pa.date_created,
				pa.date_modified
			FROM product_attribute pa
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       pa.product_id ILIKE $1 OR
			       pa.attribute_id ILIKE $1 OR
			       pa.value ILIKE $1)
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
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var productAttributes []*productattributepb.ProductAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			productId          string
			attributeId        string
			value              string
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		if err := rows.Scan(
			&id,
			&productId,
			&attributeId,
			&value,
			&dateCreated,
			&dateModified,
			&total,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		totalCount = total

		productAttribute := &productattributepb.ProductAttribute{
			Id:          id,
			ProductId:   productId,
			AttributeId: attributeId,
			Value:       value,
		}

		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productAttribute.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productAttribute.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productAttribute.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productAttribute.DateModifiedString = &dmStr
	}

		productAttributes = append(productAttributes, productAttribute)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productattributepb.GetProductAttributeListPageDataResponse{
		ProductAttributeList: productAttributes,
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

// GetProductAttributeItemPageData retrieves a single product attribute with enhanced item page data
func (r *PostgresProductAttributeRepository) GetProductAttributeItemPageData(
	ctx context.Context,
	req *productattributepb.GetProductAttributeItemPageDataRequest,
) (*productattributepb.GetProductAttributeItemPageDataResponse, error) {
	if req == nil || req.ProductAttributeId == "" {
		return nil, fmt.Errorf("product attribute ID required")
	}

	query := `
		SELECT
			pa.id,
			pa.product_id,
			pa.attribute_id,
			pa.value,
			pa.date_created,
			pa.date_modified
		FROM product_attribute pa
		WHERE pa.id = $1
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductAttributeId)

	var (
		id                 string
		productId          string
		attributeId        string
		value              string
		dateCreated        time.Time
		dateModified       time.Time
	)

	if err := row.Scan(
		&id,
		&productId,
		&attributeId,
		&value,
		&dateCreated,
		&dateModified,
	); err == sql.ErrNoRows {
		return nil, fmt.Errorf("product attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	productAttribute := &productattributepb.ProductAttribute{
		Id:          id,
		ProductId:   productId,
		AttributeId: attributeId,
		Value:       value,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productAttribute.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productAttribute.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productAttribute.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productAttribute.DateModifiedString = &dmStr
	}

	return &productattributepb.GetProductAttributeItemPageDataResponse{
		ProductAttribute: productAttribute,
		Success:          true,
	}, nil
}

// parseProductAttributeTimestamp parses various timestamp formats to Unix milliseconds
func parseProductAttributeTimestamp(ts string) (int64, error) {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.UnixMilli(), nil
	}
	for _, fmt := range []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z"} {
		if t, err := time.Parse(fmt, ts); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, nil
}

// NewProductAttributeRepository creates a new PostgreSQL product_attribute repository (old-style constructor)
func NewProductAttributeRepository(db *sql.DB, tableName string) productattributepb.ProductAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductAttributeRepository(dbOps, tableName)
}
