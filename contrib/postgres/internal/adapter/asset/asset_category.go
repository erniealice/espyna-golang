//go:build postgresql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AssetCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres asset_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAssetCategoryRepository(dbOps, tableName), nil
	})
}

var assetCategorySortableSQLCols = []string{
	"id", "code", "name",
	"parent_category_id",
	"default_depreciation_method", "default_useful_life_months",
	"depreciation_method", "useful_life_months",
	"active",
	"date_created", "date_modified",
}

var assetCategorySortSpec = espynahttp.SortSpec{AllowedCols: assetCategorySortableSQLCols}

// PostgresAssetCategoryRepository implements asset_category CRUD operations
// using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_asset_category_code ON asset_category(code) - unique lookup
//   - CREATE INDEX idx_asset_category_parent_id ON asset_category(parent_category_id) - tree traversal
//   - CREATE INDEX idx_asset_category_active ON asset_category(active) - filter active records
//
// AssetCategory has no nested message fields per asset_category.proto:15-47,
// so there are no cross-table denorm helpers.
type PostgresAssetCategoryRepository struct {
	assetcategorypb.UnimplementedAssetCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresAssetCategoryRepository creates a new PostgreSQL asset_category repository.
func NewPostgresAssetCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "asset_category" // default fallback
	}

	return &PostgresAssetCategoryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAssetCategory creates a new asset_category using common PostgreSQL operations.
func (r *PostgresAssetCategoryRepository) CreateAssetCategory(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("asset_category data is required")
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
		return nil, fmt.Errorf("failed to create asset_category: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cat := &assetcategorypb.AssetCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &assetcategorypb.CreateAssetCategoryResponse{
		Data:    []*assetcategorypb.AssetCategory{cat},
		Success: true,
	}, nil
}

// ReadAssetCategory retrieves an asset_category by ID.
func (r *PostgresAssetCategoryRepository) ReadAssetCategory(ctx context.Context, req *assetcategorypb.ReadAssetCategoryRequest) (*assetcategorypb.ReadAssetCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_category ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset_category: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("asset_category with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cat := &assetcategorypb.AssetCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &assetcategorypb.ReadAssetCategoryResponse{
		Data:    []*assetcategorypb.AssetCategory{cat},
		Success: true,
	}, nil
}

// UpdateAssetCategory updates an asset_category.
func (r *PostgresAssetCategoryRepository) UpdateAssetCategory(ctx context.Context, req *assetcategorypb.UpdateAssetCategoryRequest) (*assetcategorypb.UpdateAssetCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_category ID is required")
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
		return nil, fmt.Errorf("failed to update asset_category: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cat := &assetcategorypb.AssetCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &assetcategorypb.UpdateAssetCategoryResponse{
		Data:    []*assetcategorypb.AssetCategory{cat},
		Success: true,
	}, nil
}

// DeleteAssetCategory deletes an asset_category (soft delete via dbOps.Delete).
func (r *PostgresAssetCategoryRepository) DeleteAssetCategory(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) (*assetcategorypb.DeleteAssetCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_category ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete asset_category: %w", err)
	}

	return &assetcategorypb.DeleteAssetCategoryResponse{
		Success: true,
	}, nil
}

// ListAssetCategories lists asset_categories using common PostgreSQL operations.
func (r *PostgresAssetCategoryRepository) ListAssetCategories(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) (*assetcategorypb.ListAssetCategoriesResponse, error) {
	if err := espynahttp.ValidateSortColumns(assetCategorySortSpec, req.GetSort(), "asset_category"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list asset_categories: %w", err)
	}

	var cats []*assetcategorypb.AssetCategory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		cat := &assetcategorypb.AssetCategory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cat); err != nil {
			continue
		}
		cats = append(cats, cat)
	}

	return &assetcategorypb.ListAssetCategoriesResponse{
		Data:    cats,
		Success: true,
	}, nil
}

// GetAssetCategoryListPageData retrieves asset_categories via composition
// over the canonical ListAssetCategories. AssetCategory has no nested
// message fields per the proto, so no per-row denorm pass is needed.
//
// Page header (pagination metadata) is computed locally from len(rows) —
// the canonical ListAssetCategories does not yet emit a windowed total count.
func (r *PostgresAssetCategoryRepository) GetAssetCategoryListPageData(
	ctx context.Context,
	req *assetcategorypb.GetAssetCategoryListPageDataRequest,
) (*assetcategorypb.GetAssetCategoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_category list page data request is required")
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	listResp, err := r.ListAssetCategories(ctx, &assetcategorypb.ListAssetCategoriesRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list asset_categories for page data: %w", err)
	}
	cats := listResp.GetData()

	totalItems := int32(len(cats))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &assetcategorypb.GetAssetCategoryListPageDataResponse{
		AssetCategoryList: cats,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetAssetCategoryItemPageData retrieves a single asset_category via
// composition over ReadAssetCategory.
//
// The asset_category proto's GetAssetCategoryItemPageDataRequest carries
// asset_category_id (not nested Data.Id) — see asset_category.proto:132-134.
func (r *PostgresAssetCategoryRepository) GetAssetCategoryItemPageData(
	ctx context.Context,
	req *assetcategorypb.GetAssetCategoryItemPageDataRequest,
) (*assetcategorypb.GetAssetCategoryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_category item page data request is required")
	}
	if req.AssetCategoryId == "" {
		return nil, fmt.Errorf("asset_category ID is required")
	}

	rr, err := r.ReadAssetCategory(ctx, &assetcategorypb.ReadAssetCategoryRequest{Data: &assetcategorypb.AssetCategory{Id: req.AssetCategoryId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("asset_category with ID '%s' not found", req.AssetCategoryId)
	}
	cat := rr.GetData()[0]

	return &assetcategorypb.GetAssetCategoryItemPageDataResponse{
		AssetCategory: cat,
		Success:       true,
	}, nil
}

// NewAssetCategoryRepository creates a new PostgreSQL asset_category repository (old-style constructor).
func NewAssetCategoryRepository(db *sql.DB, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresAssetCategoryRepository(dbOps, tableName)
}
