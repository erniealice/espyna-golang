//go:build mysql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.AssetCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql asset_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLAssetCategoryRepository(dbOps, tableName), nil
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

// MySQLAssetCategoryRepository implements asset_category CRUD operations
// using MySQL 8.0+.
//
// AssetCategory has no nested message fields per asset_category.proto:15-47,
// so there are no cross-table denorm helpers.
//
// Note: ListAssetCategoriesWithPolicyRollup uses a raw SQL query that translates
// FILTER(WHERE) → CASE WHEN and IS DISTINCT FROM → IS NULL/!= combo for MySQL.
type MySQLAssetCategoryRepository struct {
	assetcategorypb.UnimplementedAssetCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLAssetCategoryRepository creates a new MySQL asset_category repository.
func NewMySQLAssetCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "asset_category"
	}
	return &MySQLAssetCategoryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAssetCategory creates a new asset_category using common MySQL operations.
func (r *MySQLAssetCategoryRepository) CreateAssetCategory(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLAssetCategoryRepository) ReadAssetCategory(ctx context.Context, req *assetcategorypb.ReadAssetCategoryRequest) (*assetcategorypb.ReadAssetCategoryResponse, error) {
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
func (r *MySQLAssetCategoryRepository) UpdateAssetCategory(ctx context.Context, req *assetcategorypb.UpdateAssetCategoryRequest) (*assetcategorypb.UpdateAssetCategoryResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteAssetCategory soft-deletes an asset_category via dbOps.Delete.
func (r *MySQLAssetCategoryRepository) DeleteAssetCategory(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) (*assetcategorypb.DeleteAssetCategoryResponse, error) {
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

// ListAssetCategories lists asset_categories using common MySQL operations.
func (r *MySQLAssetCategoryRepository) ListAssetCategories(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) (*assetcategorypb.ListAssetCategoriesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// GetAssetCategoryListPageData retrieves asset_categories with pagination metadata.
func (r *MySQLAssetCategoryRepository) GetAssetCategoryListPageData(
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
func (r *MySQLAssetCategoryRepository) GetAssetCategoryItemPageData(
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

	return &assetcategorypb.GetAssetCategoryItemPageDataResponse{
		AssetCategory: rr.GetData()[0],
		Success:       true,
	}, nil
}

// NewAssetCategoryRepository creates a new MySQL asset_category repository (old-style constructor).
func NewAssetCategoryRepository(db *sql.DB, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLAssetCategoryRepository(dbOps, tableName)
}

// ListAssetCategoriesWithPolicyRollup executes a bulk aggregate query that
// JOINs asset_category to asset and computes assets_in_policy and
// assets_deviating per category.
//
// Dialect translation from postgres gold standard:
//   - $1 → ? (MySQL positional placeholder)
//   - FILTER (WHERE ...) → CASE WHEN ... END (MySQL does not support FILTER)
//   - IS DISTINCT FROM → (col IS NULL OR col != val) / NOT (col <=> val)
//     MySQL 8.0 added NULL-safe equality operator <=> so NOT (a <=> b) is
//     equivalent to IS DISTINCT FROM.
//   - ::BIGINT cast → CAST(... AS SIGNED) (MySQL integer cast)
//   - active = true → active = 1
func (r *MySQLAssetCategoryRepository) ListAssetCategoriesWithPolicyRollup(
	ctx context.Context,
) ([]*assetcategorypb.AssetCategoryWithPolicyRollup, error) {
	dbGetter, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("asset_category adapter: dbOps does not expose GetDB() — rollup query unavailable")
	}
	db := dbGetter.GetDB()
	if db == nil {
		return nil, fmt.Errorf("asset_category adapter: GetDB() returned nil")
	}

	wsID, _ := ctx.Value("workspace_id").(string)
	if wsID == "" {
		if v := ctx.Value("WorkspaceID"); v != nil {
			wsID, _ = v.(string)
		}
	}

	// Dialect translation notes:
	//   FILTER(WHERE status='...') → CASE WHEN status='...' THEN a.id END
	//   IS DISTINCT FROM → NOT (a.col <=> COALESCE(ac.col, ac.default))
	//   ::BIGINT → CAST(... AS SIGNED)
	//   $1 → ?
	//   active = true → active = 1
	const query = `
		SELECT
			ac.id AS category_id,
			COUNT(CASE WHEN a.status = 'ASSET_STATUS_IN_SERVICE' THEN a.id END)
				AS assets_in_policy,
			COUNT(CASE WHEN a.status = 'ASSET_STATUS_IN_SERVICE'
				AND (
					NOT (a.depreciation_method <=> COALESCE(ac.depreciation_method, ac.default_depreciation_method))
					OR NOT (a.useful_life_months <=> COALESCE(ac.useful_life_months, ac.default_useful_life_months))
					OR NOT (a.salvage_value <=> CAST(a.acquisition_cost * COALESCE(ac.salvage_pct, ac.default_salvage_value_percent) / 100 AS SIGNED))
				)
				THEN a.id END)
				AS assets_deviating
		FROM asset_category ac
		LEFT JOIN asset a ON a.asset_category_id = ac.id AND a.active = 1
		WHERE ac.active = 1
		  AND (? = '' OR ac.workspace_id = ?)
		GROUP BY ac.id
	`

	// Two ? args: wsID used twice (empty-string guard + equality check).
	rows, err := db.QueryContext(ctx, query, wsID, wsID)
	if err != nil {
		return nil, fmt.Errorf("ListAssetCategoriesWithPolicyRollup: query failed: %w", err)
	}
	defer rows.Close()

	type rollupCounts struct {
		inPolicy  int
		deviating int
	}
	countsMap := make(map[string]rollupCounts)
	for rows.Next() {
		var catID string
		var inPolicy, deviating int
		if err := rows.Scan(&catID, &inPolicy, &deviating); err != nil {
			return nil, fmt.Errorf("ListAssetCategoriesWithPolicyRollup: scan failed: %w", err)
		}
		countsMap[catID] = rollupCounts{inPolicy: inPolicy, deviating: deviating}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListAssetCategoriesWithPolicyRollup: rows error: %w", err)
	}

	listResp, err := r.ListAssetCategories(ctx, &assetcategorypb.ListAssetCategoriesRequest{})
	if err != nil {
		return nil, fmt.Errorf("ListAssetCategoriesWithPolicyRollup: list categories failed: %w", err)
	}

	// Suppress "imported and not used" for strings — used in future filter clauses.
	_ = strings.Join

	result := make([]*assetcategorypb.AssetCategoryWithPolicyRollup, 0, len(listResp.GetData()))
	for _, cat := range listResp.GetData() {
		c := countsMap[cat.GetId()]
		result = append(result, &assetcategorypb.AssetCategoryWithPolicyRollup{
			Category:        cat,
			AssetsInPolicy:  int32(c.inPolicy),
			AssetsDeviating: int32(c.deviating),
		})
	}
	return result, nil
}
