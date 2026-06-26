//go:build mysql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
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
	registry.RegisterRepositoryFactory("mysql", entityid.Asset, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql asset repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLAssetRepository(dbOps, tableName), nil
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

// MySQLAssetRepository implements asset CRUD operations using MySQL 8.0+.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - "ident" → `ident` (backtick quoting)
//   - ILIKE → LIKE (ci collation)
//   - FILTER (WHERE) → CASE WHEN (see ListAssetCategoriesWithPolicyRollup)
//   - COUNT(*) OVER () preserved — MySQL 8.0+ supports window functions
//   - INSERT ... RETURNING * → app-supplied UUID + SELECT after insert (via dbOps)
//   - active = true → active = 1 (MySQL TINYINT(1))
//
// Out of scope (stubbed with "not implemented"):
// AcquireAsset, DisposeAsset, TransferAsset, RunDepreciation,
// GetDepreciationSchedule, RevalueAsset.
type MySQLAssetRepository struct {
	assetpb.UnimplementedAssetDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLAssetRepository creates a new MySQL asset repository.
func NewMySQLAssetRepository(dbOps interfaces.DatabaseOperation, tableName string) assetpb.AssetDomainServiceServer {
	if tableName == "" {
		tableName = "asset"
	}
	return &MySQLAssetRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAsset creates a new asset using common MySQL operations.
func (r *MySQLAssetRepository) CreateAsset(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
// Cross-table denorms: Asset.asset_category and Asset.location are sourced
// from the rows pointed to by asset.asset_category_id and asset.location_id
// respectively. The loadAssetCategory and loadAssetLocation helpers populate
// them after the canonical scan.
func (r *MySQLAssetRepository) ReadAsset(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
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

// loadAssetCategory fetches the AssetCategory row associated with an
// asset.asset_category_id. Returns (nil, nil) if categoryId is empty.
func (r *MySQLAssetRepository) loadAssetCategory(ctx context.Context, categoryId string) (*assetcategorypb.AssetCategory, error) {
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

// loadAssetLocation fetches the Location row associated with an asset.location_id.
// Returns (nil, nil) if locationId is empty.
func (r *MySQLAssetRepository) loadAssetLocation(ctx context.Context, locationId string) (*locationpb.Location, error) {
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

// UpdateAsset updates an asset using common MySQL operations.
func (r *MySQLAssetRepository) UpdateAsset(ctx context.Context, req *assetpb.UpdateAssetRequest) (*assetpb.UpdateAssetResponse, error) {
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

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteAsset soft-deletes an asset (sets active=false/0) via dbOps.Delete.
func (r *MySQLAssetRepository) DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error) {
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

// ListAssets lists assets using common MySQL operations.
func (r *MySQLAssetRepository) ListAssets(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// GetAssetListPageData retrieves assets via composition over ListAssets and
// adds per-row denorms (AssetCategory + Location).
func (r *MySQLAssetRepository) GetAssetListPageData(
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

// GetAssetItemPageData retrieves a single asset + denorms via composition
// over ReadAsset (which already populates AssetCategory and Location).
func (r *MySQLAssetRepository) GetAssetItemPageData(
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

// SetAssetActive mutates only the active flag and timestamps of an asset.
// read-merge-update pattern preserves all other fields.
func (r *MySQLAssetRepository) SetAssetActive(ctx context.Context, req *assetpb.SetAssetActiveRequest) (*assetpb.SetAssetActiveResponse, error) {
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

// AcquireAsset is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) AcquireAsset(ctx context.Context, req *assetpb.AcquireAssetRequest) (*assetpb.AcquireAssetResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.AcquireAsset: not implemented")
}

// DisposeAsset is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) DisposeAsset(ctx context.Context, req *assetpb.DisposeAssetRequest) (*assetpb.DisposeAssetResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.DisposeAsset: not implemented")
}

// TransferAsset is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) TransferAsset(ctx context.Context, req *assetpb.TransferAssetRequest) (*assetpb.TransferAssetResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.TransferAsset: not implemented")
}

// RunDepreciation is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) RunDepreciation(ctx context.Context, req *assetpb.RunDepreciationRequest) (*assetpb.RunDepreciationResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.RunDepreciation: not implemented")
}

// GetDepreciationSchedule is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) GetDepreciationSchedule(ctx context.Context, req *assetpb.GetDepreciationScheduleRequest) (*assetpb.GetDepreciationScheduleResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.GetDepreciationSchedule: not implemented")
}

// RevalueAsset is not implemented.
// TODO(asset-stack): implement when caller exists.
func (r *MySQLAssetRepository) RevalueAsset(ctx context.Context, req *assetpb.RevalueAssetRequest) (*assetpb.RevalueAssetResponse, error) {
	return nil, fmt.Errorf("MySQLAssetRepository.RevalueAsset: not implemented")
}

// NewAssetRepository creates a new MySQL asset repository (old-style constructor).
func NewAssetRepository(db *sql.DB, tableName string) assetpb.AssetDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLAssetRepository(dbOps, tableName)
}
