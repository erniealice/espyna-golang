//go:build postgresql

package product_variant_image

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
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_variant_image", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_variant_image repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductVariantImageRepository(dbOps, tableName), nil
	})
}

// PostgresProductVariantImageRepository implements product_variant_image CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_variant_image_active ON product_variant_image(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_variant_image_variant_id ON product_variant_image(product_variant_id) - FK lookup on product_variant_id
//   - CREATE INDEX idx_product_variant_image_is_primary ON product_variant_image(is_primary) WHERE is_primary = true - Primary image lookup
//   - CREATE INDEX idx_product_variant_image_date_created ON product_variant_image(date_created DESC) - Default sorting
type PostgresProductVariantImageRepository struct {
	productvariantimagepb.UnimplementedProductVariantImageDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductVariantImageRepository creates a new PostgreSQL product variant image repository
func NewPostgresProductVariantImageRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantimagepb.ProductVariantImageDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant_image" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductVariantImageRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductVariantImage creates a new product variant image using common PostgreSQL operations
func (r *PostgresProductVariantImageRepository) CreateProductVariantImage(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product variant image data is required")
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
		return nil, fmt.Errorf("failed to create product variant image: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantImage := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantImage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantimagepb.CreateProductVariantImageResponse{
		Data: []*productvariantimagepb.ProductVariantImage{productVariantImage},
	}, nil
}

// ReadProductVariantImage retrieves a product variant image using common PostgreSQL operations
func (r *PostgresProductVariantImageRepository) ReadProductVariantImage(ctx context.Context, req *productvariantimagepb.ReadProductVariantImageRequest) (*productvariantimagepb.ReadProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product variant image: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantImage := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantImage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantimagepb.ReadProductVariantImageResponse{
		Data: []*productvariantimagepb.ProductVariantImage{productVariantImage},
	}, nil
}

// UpdateProductVariantImage updates a product variant image using common PostgreSQL operations
func (r *PostgresProductVariantImageRepository) UpdateProductVariantImage(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
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
		return nil, fmt.Errorf("failed to update product variant image: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantImage := &productvariantimagepb.ProductVariantImage{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantImage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantimagepb.UpdateProductVariantImageResponse{
		Data: []*productvariantimagepb.ProductVariantImage{productVariantImage},
	}, nil
}

// DeleteProductVariantImage deletes a product variant image using common PostgreSQL operations
func (r *PostgresProductVariantImageRepository) DeleteProductVariantImage(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product variant image: %w", err)
	}

	return &productvariantimagepb.DeleteProductVariantImageResponse{
		Success: true,
	}, nil
}

// ListProductVariantImages lists product variant images using common PostgreSQL operations
func (r *PostgresProductVariantImageRepository) ListProductVariantImages(ctx context.Context, req *productvariantimagepb.ListProductVariantImagesRequest) (*productvariantimagepb.ListProductVariantImagesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variant images: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productVariantImages []*productvariantimagepb.ProductVariantImage
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_variant_image row: %v", err)
			continue
		}

		productVariantImage := &productvariantimagepb.ProductVariantImage{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantImage); err != nil {
			log.Printf("WARN: protojson unmarshal product_variant_image: %v", err)
			continue
		}
		productVariantImages = append(productVariantImages, productVariantImage)
	}

	return &productvariantimagepb.ListProductVariantImagesResponse{
		Data: productVariantImages,
	}, nil
}

// GetProductVariantImageListPageData retrieves product variant images with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the product_variant table to include the variant SKU
func (r *PostgresProductVariantImageRepository) GetProductVariantImageListPageData(
	ctx context.Context,
	req *productvariantimagepb.GetProductVariantImageListPageDataRequest,
) (*productvariantimagepb.GetProductVariantImageListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant image list page data request is required")
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
	sortField := "pvi.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with product_variant join for SKU
	query := `
		WITH enriched AS (
			SELECT
				pvi.id,
				pvi.date_created,
				pvi.date_modified,
				pvi.active,
				pvi.product_variant_id,
				pvi.image_url,
				COALESCE(pvi.alt_text, '') as alt_text,
				pvi.sort_order,
				pvi.is_primary,
				COALESCE(pv.sku, '') as variant_sku
			FROM product_variant_image pvi
			LEFT JOIN product_variant pv ON pvi.product_variant_id = pv.id AND pv.active = true
			WHERE pvi.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pvi.image_url ILIKE $1 OR
			       pvi.alt_text ILIKE $1 OR
			       pv.sku ILIKE $1)
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
		return nil, fmt.Errorf("failed to query product variant image list page data: %w", err)
	}
	defer rows.Close()

	var productVariantImages []*productvariantimagepb.ProductVariantImage
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			productVariantID string
			imageURL         string
			altText          string
			sortOrderVal     int32
			isPrimary        bool
			variantSku       string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productVariantID,
			&imageURL,
			&altText,
			&sortOrderVal,
			&isPrimary,
			&variantSku,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product variant image row: %w", err)
		}

		totalCount = total

		productVariantImage := &productvariantimagepb.ProductVariantImage{
			Id:               id,
			Active:           active,
			ProductVariantId: productVariantID,
			ImageUrl:         imageURL,
			SortOrder:        sortOrderVal,
			IsPrimary:        isPrimary,
		}

		// Set optional alt_text field
		if altText != "" {
			productVariantImage.AltText = &altText
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productVariantImage.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productVariantImage.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productVariantImage.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productVariantImage.DateModifiedString = &dmStr
		}

		// Note: variantSku is available but not directly mapped to the ProductVariantImage protobuf.

		productVariantImages = append(productVariantImages, productVariantImage)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product variant image rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productvariantimagepb.GetProductVariantImageListPageDataResponse{
		ProductVariantImageList: productVariantImages,
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

// GetProductVariantImageItemPageData retrieves a single product variant image with enhanced item page data using CTE
// This method joins with the product_variant table for the variant reference
func (r *PostgresProductVariantImageRepository) GetProductVariantImageItemPageData(
	ctx context.Context,
	req *productvariantimagepb.GetProductVariantImageItemPageDataRequest,
) (*productvariantimagepb.GetProductVariantImageItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant image item page data request is required")
	}
	if req.ProductVariantImageId == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	// CTE Query - Single round-trip with product_variant join
	query := `
		WITH enriched AS (
			SELECT
				pvi.id,
				pvi.date_created,
				pvi.date_modified,
				pvi.active,
				pvi.product_variant_id,
				pvi.image_url,
				COALESCE(pvi.alt_text, '') as alt_text,
				pvi.sort_order,
				pvi.is_primary,
				COALESCE(pv.sku, '') as variant_sku
			FROM product_variant_image pvi
			LEFT JOIN product_variant pv ON pvi.product_variant_id = pv.id AND pv.active = true
			WHERE pvi.id = $1 AND pvi.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductVariantImageId)

	var (
		id               string
		dateCreated      time.Time
		dateModified     time.Time
		active           bool
		productVariantID string
		imageURL         string
		altText          string
		sortOrderVal     int32
		isPrimary        bool
		variantSku       string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productVariantID,
		&imageURL,
		&altText,
		&sortOrderVal,
		&isPrimary,
		&variantSku,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product variant image with ID '%s' not found", req.ProductVariantImageId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product variant image item page data: %w", err)
	}

	productVariantImage := &productvariantimagepb.ProductVariantImage{
		Id:               id,
		Active:           active,
		ProductVariantId: productVariantID,
		ImageUrl:         imageURL,
		SortOrder:        sortOrderVal,
		IsPrimary:        isPrimary,
	}

	// Set optional alt_text field
	if altText != "" {
		productVariantImage.AltText = &altText
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productVariantImage.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productVariantImage.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productVariantImage.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productVariantImage.DateModifiedString = &dmStr
	}

	// Note: variantSku is available for the product_variant reference
	// but not directly mapped to the ProductVariantImage protobuf.

	return &productvariantimagepb.GetProductVariantImageItemPageDataResponse{
		ProductVariantImage: productVariantImage,
		Success:             true,
	}, nil
}

// NewProductVariantImageRepository creates a new PostgreSQL product variant image repository (old-style constructor)
func NewProductVariantImageRepository(db *sql.DB, tableName string) productvariantimagepb.ProductVariantImageDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductVariantImageRepository(dbOps, tableName)
}
