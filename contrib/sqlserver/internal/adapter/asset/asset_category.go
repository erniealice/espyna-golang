//go:build sqlserver

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.AssetCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver asset_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAssetCategoryRepository(dbOps, tableName), nil
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

// SQLServerAssetCategoryRepository implements asset_category CRUD operations using SQL Server.
//
// SQL Server dialect differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …).
//   - Identifier quoting: [ident] (not "ident").
//   - ILIKE → LIKE.
//   - FILTER (WHERE …) → CASE WHEN … END.
//   - Pagination: OFFSET @p ROWS FETCH NEXT @p ROWS ONLY.
//
// ListAssetCategoriesWithPolicyRollup uses CASE WHEN instead of FILTER (WHERE)
// for SQL Server conditional aggregation, and @p1 placeholder instead of $1.
type SQLServerAssetCategoryRepository struct {
	assetcategorypb.UnimplementedAssetCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerAssetCategoryRepository creates a new SQL Server asset_category repository.
func NewSQLServerAssetCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "asset_category"
	}
	return &SQLServerAssetCategoryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAssetCategory creates a new asset_category using SQL Server operations.
func (r *SQLServerAssetCategoryRepository) CreateAssetCategory(ctx context.Context, req *assetcategorypb.CreateAssetCategoryRequest) (*assetcategorypb.CreateAssetCategoryResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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
func (r *SQLServerAssetCategoryRepository) ReadAssetCategory(ctx context.Context, req *assetcategorypb.ReadAssetCategoryRequest) (*assetcategorypb.ReadAssetCategoryResponse, error) {
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
func (r *SQLServerAssetCategoryRepository) UpdateAssetCategory(ctx context.Context, req *assetcategorypb.UpdateAssetCategoryRequest) (*assetcategorypb.UpdateAssetCategoryResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// DeleteAssetCategory soft-deletes an asset_category.
func (r *SQLServerAssetCategoryRepository) DeleteAssetCategory(ctx context.Context, req *assetcategorypb.DeleteAssetCategoryRequest) (*assetcategorypb.DeleteAssetCategoryResponse, error) {
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

// ListAssetCategories lists asset_categories using SQL Server operations.
func (r *SQLServerAssetCategoryRepository) ListAssetCategories(ctx context.Context, req *assetcategorypb.ListAssetCategoriesRequest) (*assetcategorypb.ListAssetCategoriesResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// GetAssetCategoryListPageData retrieves asset_categories with pagination metadata via
// composition over ListAssetCategories. AssetCategory has no nested message fields per
// the proto, so no per-row denorm pass is needed.
func (r *SQLServerAssetCategoryRepository) GetAssetCategoryListPageData(
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

// GetAssetCategoryItemPageData retrieves a single asset_category via composition
// over ReadAssetCategory.
func (r *SQLServerAssetCategoryRepository) GetAssetCategoryItemPageData(
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

// ListAssetCategoriesWithPolicyRollup computes per-category asset counts using
// SQL Server conditional aggregation (CASE WHEN instead of FILTER (WHERE)).
//
// SQL Server differences from the postgres gold standard:
//   - FILTER (WHERE …) → CASE WHEN … THEN 1 ELSE NULL END inside COUNT.
//   - $1 → @p1.
//   - active = true → active = 1 (SQL Server BIT).
//   - IS DISTINCT FROM → (col IS NULL AND other IS NOT NULL) OR (col IS NOT NULL AND other IS NULL) OR col <> other.
//     Simplified here as a NULL-safe inequality check using SQL Server idiom.
func (r *SQLServerAssetCategoryRepository) ListAssetCategoriesWithPolicyRollup(
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

	// SQL Server translation notes:
	//   - COUNT(a.id) FILTER (WHERE condition) → COUNT(CASE WHEN condition THEN 1 ELSE NULL END)
	//   - active = true → active = 1
	//   - $1 → @p1
	//   - IS DISTINCT FROM → NOT (col = other OR (col IS NULL AND other IS NULL))
	//     simplified to: (col IS NULL AND other IS NOT NULL) OR (col IS NOT NULL AND other IS NULL) OR col <> other
	const query = `
		SELECT
			ac.id                       AS category_id,
			COUNT(CASE WHEN a.status = 'ASSET_STATUS_IN_SERVICE' THEN 1 ELSE NULL END) AS assets_in_policy,
			COUNT(CASE WHEN a.status = 'ASSET_STATUS_IN_SERVICE' AND (
				(a.depreciation_method IS NULL AND COALESCE(ac.depreciation_method, ac.default_depreciation_method) IS NOT NULL)
				OR (a.depreciation_method IS NOT NULL AND COALESCE(ac.depreciation_method, ac.default_depreciation_method) IS NULL)
				OR a.depreciation_method <> COALESCE(ac.depreciation_method, ac.default_depreciation_method)
				OR (a.useful_life_months IS NULL AND COALESCE(ac.useful_life_months, ac.default_useful_life_months) IS NOT NULL)
				OR (a.useful_life_months IS NOT NULL AND COALESCE(ac.useful_life_months, ac.default_useful_life_months) IS NULL)
				OR a.useful_life_months <> COALESCE(ac.useful_life_months, ac.default_useful_life_months)
				OR (a.salvage_value IS NULL AND CAST(a.acquisition_cost * COALESCE(ac.salvage_pct, ac.default_salvage_value_percent) / 100 AS BIGINT) IS NOT NULL)
				OR a.salvage_value <> CAST(a.acquisition_cost * COALESCE(ac.salvage_pct, ac.default_salvage_value_percent) / 100 AS BIGINT)
			) THEN 1 ELSE NULL END)    AS assets_deviating
		FROM asset_category ac
		LEFT JOIN asset a ON a.asset_category_id = ac.id AND a.active = 1
		WHERE ac.active = 1
		  AND (@p1 = '' OR ac.workspace_id = @p1)
		GROUP BY ac.id
	`

	rows, err := db.QueryContext(ctx, query, wsID)
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

// NewAssetCategoryRepository creates a new SQL Server asset_category repository (old-style constructor).
func NewAssetCategoryRepository(db *sql.DB, tableName string) assetcategorypb.AssetCategoryDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerAssetCategoryRepository(dbOps, tableName)
}
