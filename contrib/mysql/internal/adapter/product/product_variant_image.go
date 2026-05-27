//go:build mysql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.ProductVariantImage, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql product_variant_image repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLProductVariantImageRepository(dbOps, tableName), nil
	})
}

// MySQLProductVariantImageRepository implements product_variant_image CRUD operations
// using MySQL 8.0+.
type MySQLProductVariantImageRepository struct {
	productvariantimagepb.UnimplementedProductVariantImageDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLProductVariantImageRepository creates a new MySQL product variant image repository.
func NewMySQLProductVariantImageRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantimagepb.ProductVariantImageDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant_image" // default fallback
	}
	return &MySQLProductVariantImageRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateProductVariantImage creates a new product variant image using common MySQL operations.
func (r *MySQLProductVariantImageRepository) CreateProductVariantImage(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product variant image data is required")
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
		return nil, fmt.Errorf("failed to create product variant image: %w", err)
	}

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

// ReadProductVariantImage retrieves a product variant image using common MySQL operations.
func (r *MySQLProductVariantImageRepository) ReadProductVariantImage(ctx context.Context, req *productvariantimagepb.ReadProductVariantImageRequest) (*productvariantimagepb.ReadProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product variant image: %w", err)
	}

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

// UpdateProductVariantImage updates a product variant image using common MySQL operations.
func (r *MySQLProductVariantImageRepository) UpdateProductVariantImage(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
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
		return nil, fmt.Errorf("failed to update product variant image: %w", err)
	}

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

// DeleteProductVariantImage deletes a product variant image using common MySQL operations (soft delete).
func (r *MySQLProductVariantImageRepository) DeleteProductVariantImage(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product variant image: %w", err)
	}

	return &productvariantimagepb.DeleteProductVariantImageResponse{
		Success: true,
	}, nil
}

// ListProductVariantImages lists product variant images using common MySQL operations.
func (r *MySQLProductVariantImageRepository) ListProductVariantImages(ctx context.Context, req *productvariantimagepb.ListProductVariantImagesRequest) (*productvariantimagepb.ListProductVariantImagesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variant images: %w", err)
	}

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

var productVariantImageSortableSQLCols = []string{
	"pvi.id", "pvi.active", "pvi.product_variant_id", "pvi.image_url",
	"pvi.alt_text", "pvi.sort_order", "pvi.is_primary",
	"pvi.date_created", "pvi.date_modified",
}

// GetProductVariantImageListPageData retrieves product variant images with
// advanced filtering, sorting, searching, and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,$3 → ? (same arg order: searchPattern, limit, offset)
//   - $1::text IS NULL OR $1::text = ” → ? = ” OR — MySQL empty-string guard
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1 (MySQL TINYINT(1))
//   - core.BuildOrderBy for backtick-quoted sort column
//   - COUNT(*) OVER () — MySQL 8.0+ supports window functions
//   - LIMIT ? OFFSET ? (positional, appended last)
func (r *MySQLProductVariantImageRepository) GetProductVariantImageListPageData(
	ctx context.Context,
	req *productvariantimagepb.GetProductVariantImageListPageDataRequest,
) (*productvariantimagepb.GetProductVariantImageListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant image list page data request is required")
	}

	// Build search condition.
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values.
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard).
	orderByClause, err := mysqlCore.BuildOrderBy(productVariantImageSortableSQLCols, req.GetSort(), "`pvi`.`date_created` DESC")
	if err != nil {
		return nil, err
	}

	// Dialect: ILIKE → LIKE; $1::text IS NULL OR ... → searchPattern != "" guard.
	whereSearch := ""
	if searchPattern != "" {
		whereSearch = "AND (pvi.image_url LIKE ? OR pvi.alt_text LIKE ? OR pv.sku LIKE ?)"
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				pvi.id,
				pvi.date_created,
				pvi.date_modified,
				pvi.active,
				pvi.product_variant_id,
				pvi.image_url,
				COALESCE(pvi.alt_text, '') AS alt_text,
				pvi.sort_order,
				pvi.is_primary,
				COALESCE(pv.sku, '') AS variant_sku
			FROM product_variant_image pvi
			LEFT JOIN product_variant pv ON pvi.product_variant_id = pv.id AND pv.active = 1
			WHERE pvi.active = 1
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s
		LIMIT ? OFFSET ?;
	`, whereSearch, orderByClause)

	// Build args: [searchPattern x3 (if present), limit, offset]
	queryArgs := []any{}
	if searchPattern != "" {
		queryArgs = append(queryArgs, searchPattern, searchPattern, searchPattern)
	}
	queryArgs = append(queryArgs, limit, offset)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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

		if err := rows.Scan(
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
		); err != nil {
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

		if altText != "" {
			productVariantImage.AltText = &altText
		}

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

		// variantSku is available but not directly mapped to the ProductVariantImage proto.

		productVariantImages = append(productVariantImages, productVariantImage)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product variant image rows: %w", err)
	}

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

// GetProductVariantImageItemPageData retrieves a single product variant image
// with enhanced item page data.
//
// Dialect translation from postgres gold standard:
//   - $1 → ? (productVariantImageId)
//   - active = true → active = 1
//   - SELECT * FROM enriched LIMIT 1 stays unchanged
func (r *MySQLProductVariantImageRepository) GetProductVariantImageItemPageData(
	ctx context.Context,
	req *productvariantimagepb.GetProductVariantImageItemPageDataRequest,
) (*productvariantimagepb.GetProductVariantImageItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant image item page data request is required")
	}
	if req.ProductVariantImageId == "" {
		return nil, fmt.Errorf("product variant image ID is required")
	}

	// Dialect: $1 → ?, active = true → active = 1.
	const query = `
		WITH enriched AS (
			SELECT
				pvi.id,
				pvi.date_created,
				pvi.date_modified,
				pvi.active,
				pvi.product_variant_id,
				pvi.image_url,
				COALESCE(pvi.alt_text, '') AS alt_text,
				pvi.sort_order,
				pvi.is_primary,
				COALESCE(pv.sku, '') AS variant_sku
			FROM product_variant_image pvi
			LEFT JOIN product_variant pv ON pvi.product_variant_id = pv.id AND pv.active = 1
			WHERE pvi.id = ? AND pvi.active = 1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.ProductVariantImageId)

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

	if altText != "" {
		productVariantImage.AltText = &altText
	}

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

	// variantSku is available for the product_variant reference
	// but not directly mapped to the ProductVariantImage protobuf.

	return &productvariantimagepb.GetProductVariantImageItemPageDataResponse{
		ProductVariantImage: productVariantImage,
		Success:             true,
	}, nil
}

// NewProductVariantImageRepository creates a new MySQL product variant image repository (old-style constructor).
func NewProductVariantImageRepository(db *sql.DB, tableName string) productvariantimagepb.ProductVariantImageDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLProductVariantImageRepository(dbOps, tableName)
}
