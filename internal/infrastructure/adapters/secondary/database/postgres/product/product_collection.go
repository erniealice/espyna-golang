//go:build postgresql

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
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_collection", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductCollectionRepository(dbOps, tableName), nil
	})
}

// PostgresProductCollectionRepository implements product_collection CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_collection_active ON product_collection(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_collection_product_id ON product_collection(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_product_collection_collection_id ON product_collection(collection_id) - FK lookup on collection_id
//   - CREATE INDEX idx_product_collection_date_created ON product_collection(date_created DESC) - Default sorting
type PostgresProductCollectionRepository struct {
	productcollectionpb.UnimplementedProductCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductCollectionRepository creates a new PostgreSQL product collection repository
func NewPostgresProductCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) productcollectionpb.ProductCollectionDomainServiceServer {
	if tableName == "" {
		tableName = "product_collection" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductCollection creates a new product collection using common PostgreSQL operations
func (r *PostgresProductCollectionRepository) CreateProductCollection(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product collection data is required")
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
		return nil, fmt.Errorf("failed to create product collection: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productCollection := &productcollectionpb.ProductCollection{}
	if err := protojson.Unmarshal(resultJSON, productCollection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productcollectionpb.CreateProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{productCollection},
	}, nil
}

// ReadProductCollection retrieves a product collection using common PostgreSQL operations
func (r *PostgresProductCollectionRepository) ReadProductCollection(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) (*productcollectionpb.ReadProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product collection: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productCollection := &productcollectionpb.ProductCollection{}
	if err := protojson.Unmarshal(resultJSON, productCollection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productcollectionpb.ReadProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{productCollection},
	}, nil
}

// UpdateProductCollection updates a product collection using common PostgreSQL operations
func (r *PostgresProductCollectionRepository) UpdateProductCollection(ctx context.Context, req *productcollectionpb.UpdateProductCollectionRequest) (*productcollectionpb.UpdateProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
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
		return nil, fmt.Errorf("failed to update product collection: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productCollection := &productcollectionpb.ProductCollection{}
	if err := protojson.Unmarshal(resultJSON, productCollection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productcollectionpb.UpdateProductCollectionResponse{
		Data: []*productcollectionpb.ProductCollection{productCollection},
	}, nil
}

// DeleteProductCollection deletes a product collection using common PostgreSQL operations
func (r *PostgresProductCollectionRepository) DeleteProductCollection(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product collection: %w", err)
	}

	return &productcollectionpb.DeleteProductCollectionResponse{
		Success: true,
	}, nil
}

// ListProductCollections lists product collections using common PostgreSQL operations
func (r *PostgresProductCollectionRepository) ListProductCollections(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) (*productcollectionpb.ListProductCollectionsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product collections: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productCollections []*productcollectionpb.ProductCollection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		productCollection := &productcollectionpb.ProductCollection{}
		if err := protojson.Unmarshal(resultJSON, productCollection); err != nil {
			// Log error and continue with next item
			continue
		}
		productCollections = append(productCollections, productCollection)
	}

	return &productcollectionpb.ListProductCollectionsResponse{
		Data: productCollections,
	}, nil
}

// GetProductCollectionListPageData retrieves product collections with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresProductCollectionRepository) GetProductCollectionListPageData(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionListPageDataRequest,
) (*productcollectionpb.GetProductCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, product_id, collection_id, active, date_created, date_modified FROM product_collection WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR product_id ILIKE $1 OR collection_id ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var productCollections []*productcollectionpb.ProductCollection
	var totalCount int64
	for rows.Next() {
		var id, productId, collectionId string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &productId, &collectionId, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		productCollection := &productcollectionpb.ProductCollection{Id: id, ProductId: productId, CollectionId: collectionId, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productCollection.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productCollection.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productCollection.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productCollection.DateModifiedString = &dmStr
	}
		productCollections = append(productCollections, productCollection)
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &productcollectionpb.GetProductCollectionListPageDataResponse{ProductCollectionList: productCollections, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetProductCollectionItemPageData retrieves product collection item page data
func (r *PostgresProductCollectionRepository) GetProductCollectionItemPageData(ctx context.Context, req *productcollectionpb.GetProductCollectionItemPageDataRequest) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	if req == nil || req.ProductCollectionId == "" {
		return nil, fmt.Errorf("product collection ID required")
	}
	query := `SELECT id, product_id, collection_id, active, date_created, date_modified FROM product_collection WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.ProductCollectionId)
	var id, productId, collectionId string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &productId, &collectionId, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("product collection not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	productCollection := &productcollectionpb.ProductCollection{Id: id, ProductId: productId, CollectionId: collectionId, Active: active}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productCollection.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productCollection.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productCollection.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productCollection.DateModifiedString = &dmStr
	}
	return &productcollectionpb.GetProductCollectionItemPageDataResponse{ProductCollection: productCollection, Success: true}, nil
}

// parseProductCollectionTimestamp parses various timestamp formats to Unix milliseconds
func parseProductCollectionTimestamp(ts string) (int64, error) {
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

// NewProductCollectionRepository creates a new PostgreSQL product_collection repository (old-style constructor)
func NewProductCollectionRepository(db *sql.DB, tableName string) productcollectionpb.ProductCollectionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductCollectionRepository(dbOps, tableName)
}
