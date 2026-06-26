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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.ProductVariant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql product_variant repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLProductVariantRepository(dbOps, tableName), nil
	})
}

// MySQLProductVariantRepository implements product_variant CRUD operations using MySQL 8.0+.
type MySQLProductVariantRepository struct {
	productvariantpb.UnimplementedProductVariantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLProductVariantRepository creates a new MySQL product variant repository.
func NewMySQLProductVariantRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantpb.ProductVariantDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant" // default fallback
	}
	return &MySQLProductVariantRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateProductVariant creates a new product variant using common MySQL operations.
func (r *MySQLProductVariantRepository) CreateProductVariant(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product variant data is required")
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
		return nil, fmt.Errorf("failed to create product variant: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.CreateProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// ReadProductVariant retrieves a product variant using common MySQL operations.
func (r *MySQLProductVariantRepository) ReadProductVariant(ctx context.Context, req *productvariantpb.ReadProductVariantRequest) (*productvariantpb.ReadProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product variant: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.ReadProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// UpdateProductVariant updates a product variant using common MySQL operations.
func (r *MySQLProductVariantRepository) UpdateProductVariant(ctx context.Context, req *productvariantpb.UpdateProductVariantRequest) (*productvariantpb.UpdateProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
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
		return nil, fmt.Errorf("failed to update product variant: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.UpdateProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// DeleteProductVariant deletes a product variant using common MySQL operations (soft delete).
func (r *MySQLProductVariantRepository) DeleteProductVariant(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product variant: %w", err)
	}

	return &productvariantpb.DeleteProductVariantResponse{
		Success: true,
	}, nil
}

// ListProductVariants lists product variants using common MySQL operations.
func (r *MySQLProductVariantRepository) ListProductVariants(ctx context.Context, req *productvariantpb.ListProductVariantsRequest) (*productvariantpb.ListProductVariantsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variants: %w", err)
	}

	var productVariants []*productvariantpb.ProductVariant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_variant row: %v", err)
			continue
		}

		productVariant := &productvariantpb.ProductVariant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
			log.Printf("WARN: protojson unmarshal product_variant: %v", err)
			continue
		}
		productVariants = append(productVariants, productVariant)
	}

	return &productvariantpb.ListProductVariantsResponse{
		Data: productVariants,
	}, nil
}

var productVariantSortableSQLCols = []string{
	"pv.id", "pv.active", "pv.product_id", "pv.sku", "pv.price_override",
	"pv.date_created", "pv.date_modified",
}

// GetProductVariantListPageData retrieves product variants with advanced
// filtering, sorting, searching, and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,$3 → ? (same arg order: searchPattern, limit, offset)
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - $1::text IS NULL OR $1::text = ” → ? = ” OR — MySQL empty-string guard
//   - core.BuildOrderBy for backtick-quoted sort column
//   - COUNT(*) OVER () — MySQL 8.0+ supports window functions
//   - active = true → active = 1 (MySQL TINYINT(1))
//   - LIMIT ? OFFSET ? (positional, appended last)
//
// CRITICAL: workspace_id isolation is enforced by WorkspaceAwareOperations on
// the CRUD path; raw-SQL path adds workspace context via WorkspaceAwareOperations
// executor which already carries workspace context from the calling context.
func (r *MySQLProductVariantRepository) GetProductVariantListPageData(
	ctx context.Context,
	req *productvariantpb.GetProductVariantListPageDataRequest,
) (*productvariantpb.GetProductVariantListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant list page data request is required")
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
	orderByClause, err := mysqlCore.BuildOrderBy(productVariantSortableSQLCols, req.GetSort(), "`pv`.`date_created` DESC")
	if err != nil {
		return nil, err
	}

	// Dialect: $1::text IS NULL OR $1::text = '' → ? = '' OR (MySQL).
	// searchPattern appears in args once; two LIKE comparisons each consume one ?.
	whereSearch := ""
	if searchPattern != "" {
		whereSearch = "AND (pv.sku LIKE ? OR p.name LIKE ?)"
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				pv.id,
				pv.date_created,
				pv.date_modified,
				pv.active,
				pv.product_id,
				pv.sku,
				pv.price_override,
				COALESCE(p.name, '') AS product_name
			FROM product_variant pv
			LEFT JOIN product p ON pv.product_id = p.id AND p.active = 1
			WHERE pv.active = 1
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

	// Build args: [searchPattern x2 (if present), limit, offset]
	queryArgs := []any{}
	if searchPattern != "" {
		queryArgs = append(queryArgs, searchPattern, searchPattern)
	}
	queryArgs = append(queryArgs, limit, offset)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query product variant list page data: %w", err)
	}
	defer rows.Close()

	var productVariants []*productvariantpb.ProductVariant
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			productID     string
			sku           string
			priceOverride sql.NullInt64
			productName   string
			total         int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productID,
			&sku,
			&priceOverride,
			&productName,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product variant row: %w", err)
		}

		totalCount = total

		productVariant := &productvariantpb.ProductVariant{
			Id:        id,
			Active:    active,
			ProductId: productID,
			Sku:       sku,
		}
		if priceOverride.Valid {
			po := priceOverride.Int64
			productVariant.PriceOverride = &po
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productVariant.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productVariant.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productVariant.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productVariant.DateModifiedString = &dmStr
		}

		// productName is available but not mapped to the ProductVariant proto in
		// this list view.

		productVariants = append(productVariants, productVariant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product variant rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &productvariantpb.GetProductVariantListPageDataResponse{
		ProductVariantList: productVariants,
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

// GetProductVariantItemPageData retrieves a single product variant with enhanced
// item page data.
//
// Dialect translation from postgres gold standard:
//   - $1 → ? (productVariantId)
//   - active = true → active = 1
//   - SELECT * FROM enriched LIMIT 1 stays unchanged
func (r *MySQLProductVariantRepository) GetProductVariantItemPageData(
	ctx context.Context,
	req *productvariantpb.GetProductVariantItemPageDataRequest,
) (*productvariantpb.GetProductVariantItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant item page data request is required")
	}
	if req.ProductVariantId == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	// Dialect: $1 → ?, active = true → active = 1.
	const query = `
		WITH enriched AS (
			SELECT
				pv.id,
				pv.date_created,
				pv.date_modified,
				pv.active,
				pv.product_id,
				pv.sku,
				pv.price_override,
				COALESCE(p.name, '')     AS product_name,
				COALESCE(p.price, 0)     AS product_price,
				COALESCE(p.currency, '') AS product_currency
			FROM product_variant pv
			LEFT JOIN product p ON pv.product_id = p.id AND p.active = 1
			WHERE pv.id = ? AND pv.active = 1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.ProductVariantId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		productID       string
		sku             string
		priceOverride   sql.NullInt64
		productName     string
		productPrice    int64
		productCurrency string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productID,
		&sku,
		&priceOverride,
		&productName,
		&productPrice,
		&productCurrency,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product variant with ID '%s' not found", req.ProductVariantId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product variant item page data: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{
		Id:        id,
		Active:    active,
		ProductId: productID,
		Sku:       sku,
	}
	if priceOverride.Valid {
		po := priceOverride.Int64
		productVariant.PriceOverride = &po
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productVariant.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productVariant.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productVariant.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productVariant.DateModifiedString = &dmStr
	}

	// productName, productPrice, productCurrency are available for the product
	// reference but not directly mapped to the ProductVariant protobuf.

	return &productvariantpb.GetProductVariantItemPageDataResponse{
		ProductVariant: productVariant,
		Success:        true,
	}, nil
}

// NewProductVariantRepository creates a new MySQL product variant repository (old-style constructor).
func NewProductVariantRepository(db *sql.DB, tableName string) productvariantpb.ProductVariantDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLProductVariantRepository(dbOps, tableName)
}
