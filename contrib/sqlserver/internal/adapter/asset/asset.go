//go:build sqlserver

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Asset, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver asset repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAssetRepository(dbOps, tableName), nil
	})
}

var assetSortableSQLCols = []string{
	"id", "asset_number", "name", "asset_type", "asset_category_id", "location_id",
	"serial_number", "tag_number", "manufacturer", "model",
	"custodian_id", "vendor_id",
	"acquisition_date", "acquisition_cost", "currency",
	"salvage_value", "book_value", "useful_life_months",
	"depreciation_method", "accumulated_depreciation",
	"status", "active",
	"date_created", "date_modified",
}

var assetSortSpec = espynahttp.SortSpec{AllowedCols: assetSortableSQLCols}

// SQLServerAssetRepository implements asset CRUD operations using SQL Server.
//
// SQL Server dialect differences from the postgres gold standard:
//   - Placeholders: @p1, @p2, … (not $1, $2, …).
//   - Identifier quoting: [ident] (not "ident").
//   - ILIKE → LIKE (SQL Server default CI collation is case-insensitive).
//   - FILTER (WHERE …) → CASE WHEN … END in aggregate expressions.
//   - Pagination: OFFSET @p ROWS FETCH NEXT @p ROWS ONLY (requires ORDER BY).
//   - INSERT RETURNING * → INSERT … OUTPUT inserted.* (handled by core).
//   - UPDATE RETURNING * → UPDATE … OUTPUT inserted.* (handled by core).
//
// Out of scope (stubbed Unimplemented): AcquireAsset, DisposeAsset,
// TransferAsset, RunDepreciation, GetDepreciationSchedule, RevalueAsset.
// See docs/plan/20260503-asset-typed-stack-buildout/plan.md.
type SQLServerAssetRepository struct {
	assetpb.UnimplementedAssetDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerAssetRepository creates a new SQL Server asset repository.
func NewSQLServerAssetRepository(dbOps interfaces.DatabaseOperation, tableName string) assetpb.AssetDomainServiceServer {
	if tableName == "" {
		tableName = "asset"
	}
	return &SQLServerAssetRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAsset creates a new asset using SQL Server operations.
func (r *SQLServerAssetRepository) CreateAsset(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("asset data is required")
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
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	asset := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, asset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &assetpb.CreateAssetResponse{
		Data:    []*assetpb.Asset{asset},
		Success: true,
	}, nil
}

// ReadAsset retrieves an asset by ID using the canonical dbOps.Read +
// protojson DiscardUnknown round-trip.
//
// Cross-table denorms: Asset.asset_category and Asset.location are populated
// via the loadAssetCategory and loadAssetLocation helpers after the canonical scan.
func (r *SQLServerAssetRepository) ReadAsset(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("asset with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	asset := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, asset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	if cat, err := r.loadAssetCategory(ctx, asset.GetAssetCategoryId()); err == nil && cat != nil {
		asset.AssetCategory = cat
	}
	if loc, err := r.loadAssetLocation(ctx, asset.GetLocationId()); err == nil && loc != nil {
		asset.Location = loc
	}

	return &assetpb.ReadAssetResponse{
		Data:    []*assetpb.Asset{asset},
		Success: true,
	}, nil
}

// loadAssetCategory fetches the AssetCategory row for an asset.
// Returns (nil, nil) if categoryId is empty or row is missing.
func (r *SQLServerAssetRepository) loadAssetCategory(ctx context.Context, categoryId string) (*assetcategorypb.AssetCategory, error) {
	if categoryId == "" {
		return nil, nil
	}
	result, err := r.dbOps.Read(ctx, "asset_category", categoryId)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset_category for asset: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_category result to JSON: %w", err)
	}

	cat := &assetcategorypb.AssetCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_category JSON to protobuf: %w", err)
	}
	return cat, nil
}

// loadAssetLocation fetches the Location row for an asset.
// Returns (nil, nil) if locationId is empty or row is missing.
func (r *SQLServerAssetRepository) loadAssetLocation(ctx context.Context, locationId string) (*locationpb.Location, error) {
	if locationId == "" {
		return nil, nil
	}
	result, err := r.dbOps.Read(ctx, "location", locationId)
	if err != nil {
		return nil, fmt.Errorf("failed to read location for asset: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal location result to JSON: %w", err)
	}

	loc := &locationpb.Location{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location JSON to protobuf: %w", err)
	}
	return loc, nil
}

// UpdateAsset updates an asset using SQL Server operations.
func (r *SQLServerAssetRepository) UpdateAsset(ctx context.Context, req *assetpb.UpdateAssetRequest) (*assetpb.UpdateAssetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset ID is required")
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
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	asset := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, asset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &assetpb.UpdateAssetResponse{
		Data:    []*assetpb.Asset{asset},
		Success: true,
	}, nil
}

// DeleteAsset soft-deletes an asset (sets active=false) using SQL Server operations.
func (r *SQLServerAssetRepository) DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete asset: %w", err)
	}

	return &assetpb.DeleteAssetResponse{
		Success: true,
	}, nil
}

// ListAssets lists assets using SQL Server operations.
func (r *SQLServerAssetRepository) ListAssets(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
	if err := espynahttp.ValidateSortColumns(assetSortSpec, req.GetSort(), "asset"); err != nil {
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
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	var assets []*assetpb.Asset
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		asset := &assetpb.Asset{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, asset); err != nil {
			continue
		}
		assets = append(assets, asset)
	}

	return &assetpb.ListAssetsResponse{
		Data:    assets,
		Success: true,
	}, nil
}

// GetAssetListPageData retrieves assets with pagination metadata and per-row
// denorms (AssetCategory, Location) via composition over ListAssets.
func (r *SQLServerAssetRepository) GetAssetListPageData(
	ctx context.Context,
	req *assetpb.GetAssetListPageDataRequest,
) (*assetpb.GetAssetListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset list page data request is required")
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

	listResp, err := r.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list assets for page data: %w", err)
	}
	assets := listResp.GetData()

	// Denorm pass — bounded by page size (typically ≤ 50).
	for _, a := range assets {
		if cat, err := r.loadAssetCategory(ctx, a.GetAssetCategoryId()); err == nil && cat != nil {
			a.AssetCategory = cat
		}
		if loc, err := r.loadAssetLocation(ctx, a.GetLocationId()); err == nil && loc != nil {
			a.Location = loc
		}
	}

	totalItems := int32(len(assets))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &assetpb.GetAssetListPageDataResponse{
		AssetList: assets,
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

// GetAssetItemPageData retrieves a single asset + denorms via composition over ReadAsset.
func (r *SQLServerAssetRepository) GetAssetItemPageData(
	ctx context.Context,
	req *assetpb.GetAssetItemPageDataRequest,
) (*assetpb.GetAssetItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset item page data request is required")
	}
	if req.AssetId == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	rr, err := r.ReadAsset(ctx, &assetpb.ReadAssetRequest{Data: &assetpb.Asset{Id: req.AssetId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("asset with ID '%s' not found", req.AssetId)
	}

	return &assetpb.GetAssetItemPageDataResponse{
		Asset:   rr.GetData()[0],
		Success: true,
	}, nil
}

// SetAssetActive mutates only the active flag of an asset via read-merge-update.
func (r *SQLServerAssetRepository) SetAssetActive(ctx context.Context, req *assetpb.SetAssetActiveRequest) (*assetpb.SetAssetActiveResponse, error) {
	if req == nil || req.AssetId == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	read, err := r.dbOps.Read(ctx, r.tableName, req.AssetId)
	if err != nil {
		return nil, fmt.Errorf("read asset: %w", err)
	}

	readJSON, _ := json.Marshal(read)
	current := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(readJSON, current); err != nil {
		return nil, fmt.Errorf("unmarshal asset: %w", err)
	}

	current.Active = req.Active
	now := time.Now().UnixMilli()
	current.DateModified = &now
	dms := time.Now().Format(time.RFC3339)
	current.DateModifiedString = &dms

	updateJSON, _ := protojson.Marshal(current)
	var updateMap map[string]any
	if err := json.Unmarshal(updateJSON, &updateMap); err != nil {
		return nil, fmt.Errorf("asset update marshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, current.Id, updateMap)
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}

	resultJSON, _ := json.Marshal(result)
	out := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, out); err != nil {
		return nil, fmt.Errorf("unmarshal updated asset: %w", err)
	}

	return &assetpb.SetAssetActiveResponse{
		Data:    out,
		Success: true,
	}, nil
}

// AcquireAsset is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) AcquireAsset(ctx context.Context, req *assetpb.AcquireAssetRequest) (*assetpb.AcquireAssetResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.AcquireAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// DisposeAsset is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) DisposeAsset(ctx context.Context, req *assetpb.DisposeAssetRequest) (*assetpb.DisposeAssetResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.DisposeAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// TransferAsset is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) TransferAsset(ctx context.Context, req *assetpb.TransferAssetRequest) (*assetpb.TransferAssetResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.TransferAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// RunDepreciation is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) RunDepreciation(ctx context.Context, req *assetpb.RunDepreciationRequest) (*assetpb.RunDepreciationResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.RunDepreciation: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// GetDepreciationSchedule is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) GetDepreciationSchedule(ctx context.Context, req *assetpb.GetDepreciationScheduleRequest) (*assetpb.GetDepreciationScheduleResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.GetDepreciationSchedule: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// RevalueAsset is not implemented. TODO: implement when caller exists.
func (r *SQLServerAssetRepository) RevalueAsset(ctx context.Context, req *assetpb.RevalueAssetRequest) (*assetpb.RevalueAssetResponse, error) {
	return nil, fmt.Errorf("SQLServerAssetRepository.RevalueAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// NewAssetRepository creates a new SQL Server asset repository (old-style constructor).
func NewAssetRepository(db *sql.DB, tableName string) assetpb.AssetDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerAssetRepository(dbOps, tableName)
}
