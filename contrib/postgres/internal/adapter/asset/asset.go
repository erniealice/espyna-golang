//go:build postgresql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Asset, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres asset repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAssetRepository(dbOps, tableName), nil
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

// PostgresAssetRepository implements asset CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_asset_asset_category_id ON asset(asset_category_id) - FK denorm join
//   - CREATE INDEX idx_asset_location_id ON asset(location_id) - FK denorm join
//   - CREATE INDEX idx_asset_active ON asset(active) - filter active records
//   - CREATE INDEX idx_asset_asset_number ON asset(asset_number) - unique lookups + default sort
//   - CREATE INDEX idx_asset_status ON asset(status) - lifecycle filtering
//
// Out of scope for this adapter (stubbed with "not implemented"):
// AcquireAsset, DisposeAsset, TransferAsset, RunDepreciation,
// GetDepreciationSchedule, RevalueAsset. See
// docs/plan/20260503-asset-typed-stack-buildout/plan.md for the rationale —
// each lifecycle/depreciation rpc is its own substantial feature with no
// current caller in fycha block.go or anywhere else this adapter unblocks.
type PostgresAssetRepository struct {
	assetpb.UnimplementedAssetDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresAssetRepository creates a new PostgreSQL asset repository.
func NewPostgresAssetRepository(dbOps interfaces.DatabaseOperation, tableName string) assetpb.AssetDomainServiceServer {
	if tableName == "" {
		tableName = "asset" // default fallback
	}

	return &PostgresAssetRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAsset creates a new asset using common PostgreSQL operations.
func (r *PostgresAssetRepository) CreateAsset(ctx context.Context, req *assetpb.CreateAssetRequest) (*assetpb.CreateAssetResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("asset data is required")
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
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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
// protojson DiscardUnknown round-trip, so new Asset proto fields are picked
// up automatically without column-whitelist drift.
//
// Cross-table denorms: Asset.asset_category and Asset.location are sourced
// from the rows pointed to by asset.asset_category_id and asset.location_id
// respectively, NOT from columns on the asset row. The loadAssetCategory
// and loadAssetLocation helpers populate them after the canonical scan.
func (r *PostgresAssetRepository) ReadAsset(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
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

	// AssetCategory denorm — populate Asset.AssetCategory from the asset_category row.
	if cat, err := r.loadAssetCategory(ctx, asset.GetAssetCategoryId()); err == nil && cat != nil {
		asset.AssetCategory = cat
	}

	// Location denorm — populate Asset.Location from the location row.
	if loc, err := r.loadAssetLocation(ctx, asset.GetLocationId()); err == nil && loc != nil {
		asset.Location = loc
	}

	return &assetpb.ReadAssetResponse{
		Data:    []*assetpb.Asset{asset},
		Success: true,
	}, nil
}

// loadAssetCategory fetches the AssetCategory row associated with an
// asset.asset_category_id and returns a populated AssetCategory proto.
// Returns (nil, nil) if categoryId is empty or the row is missing — keeps
// Asset.AssetCategory optional behavior intact.
//
// Mirrors loadClientUser at entity/client.go:171-193.
func (r *PostgresAssetRepository) loadAssetCategory(ctx context.Context, categoryId string) (*assetcategorypb.AssetCategory, error) {
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

// loadAssetLocation fetches the Location row associated with an
// asset.location_id and returns a populated Location proto. Returns
// (nil, nil) if locationId is empty or the row is missing.
//
// Mirrors loadClientUser at entity/client.go:171-193.
func (r *PostgresAssetRepository) loadAssetLocation(ctx context.Context, locationId string) (*locationpb.Location, error) {
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

// UpdateAsset updates an asset using common PostgreSQL operations.
func (r *PostgresAssetRepository) UpdateAsset(ctx context.Context, req *assetpb.UpdateAssetRequest) (*assetpb.UpdateAssetResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset ID is required")
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
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// DeleteAsset deletes an asset using common PostgreSQL operations.
//
// Soft-delete: dbOps.Delete sets active=false on the row rather than
// removing it. This matches the legacy fycha block.go:489 closure behavior
// (`UPDATE asset SET active = false`). Hard delete would silently regress
// the user-facing experience by vanishing history from any active=false view.
func (r *PostgresAssetRepository) DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error) {
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

// ListAssets lists assets using common PostgreSQL operations.
func (r *PostgresAssetRepository) ListAssets(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
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
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		asset := &assetpb.Asset{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, asset); err != nil {
			// Log error and continue with next item
			continue
		}
		assets = append(assets, asset)
	}

	return &assetpb.ListAssetsResponse{
		Data:    assets,
		Success: true,
	}, nil
}

// GetAssetListPageData retrieves assets via composition over the canonical
// ListAssets (which routes through dbOps.List + protojson DiscardUnknown),
// and adds the page-data denorms (Asset.AssetCategory + Asset.Location per
// row).
//
// Caveat: cross-table sort/search by category-name or location-name is
// intentionally dropped from this path — the canonical List* primitive
// operates on a single table. Filters and search over asset-table columns
// are preserved by passing req.Filters / req.Search through unchanged.
// Callers needing category/location-field sort should sort client-side
// over the populated Asset.AssetCategory.* / Asset.Location.* fields.
//
// The legacy fycha block.go:506-515 raw query used a LEFT JOIN
// asset_category + LEFT JOIN location; that join shape is preserved here as
// per-row denorm via the two loaders, not via a hand-written SQL JOIN.
//
// Page header (pagination metadata) is computed locally from len(rows) —
// the canonical ListAssets does not yet emit a windowed total count.
// total_items reflects the page size, not the global count, until
// ListAssets gains pagination metadata.
func (r *PostgresAssetRepository) GetAssetListPageData(
	ctx context.Context,
	req *assetpb.GetAssetListPageDataRequest,
) (*assetpb.GetAssetListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset list page data request is required")
	}

	// Default pagination — preserved so the response pagination block matches
	// the legacy shape even though total_items is best-effort.
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

	// Delegate row fetch to canonical ListAssets — passes filters+search
	// through. Active = true is enforced by dbOps.List default.
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

	// Denorm pass: populate Asset.AssetCategory and Asset.Location per row.
	// Bounded by page size (≤ limit, default 50). Each row triggers two
	// PK-indexed reads (asset_category + location) — acceptable for typical pages.
	for _, a := range assets {
		if cat, err := r.loadAssetCategory(ctx, a.GetAssetCategoryId()); err == nil && cat != nil {
			a.AssetCategory = cat
		}
		if loc, err := r.loadAssetLocation(ctx, a.GetLocationId()); err == nil && loc != nil {
			a.Location = loc
		}
	}

	// Pagination response — total_items is page-bounded since ListAssets
	// does not emit a windowed count. See doc comment above.
	totalItems := int32(len(assets))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		// Likely more pages exist; we cannot know without a count query.
		// Mark hasNext true so the UI keeps offering Next.
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
// over the canonical ReadAsset (which already populates AssetCategory and
// Location). Wraps in the page-data response shape.
//
// Note the asset proto's GetAssetItemPageDataRequest carries asset_id
// (not nested Data.Id) — see asset.proto:216-218.
func (r *PostgresAssetRepository) GetAssetItemPageData(
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
	asset := rr.GetData()[0]

	return &assetpb.GetAssetItemPageDataResponse{
		Asset:   asset,
		Success: true,
	}, nil
}

// SetAssetActive mutates only the active flag and timestamps of an asset.
// All other fields stay untouched. Mirrors PostgresBillingEventRepository.SetStatus
// at billing_event.go:418-477 — read-merge-update inside the adapter so the
// proto3 zero-bool for active is handled safely without a full UpdateAsset payload.
//
// req.Reason is captured in the request but not applied — asset has no reason
// column. Reserved for future audit integration.
func (r *PostgresAssetRepository) SetAssetActive(ctx context.Context, req *assetpb.SetAssetActiveRequest) (*assetpb.SetAssetActiveResponse, error) {
	if req == nil || req.AssetId == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	// Read first to fold the new active value into the existing row.
	read, err := r.dbOps.Read(ctx, r.tableName, req.AssetId)
	if err != nil {
		return nil, fmt.Errorf("read asset: %w", err)
	}

	// Convert raw map back to proto.
	readJSON, _ := json.Marshal(read)
	current := &assetpb.Asset{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(readJSON, current); err != nil {
		return nil, fmt.Errorf("unmarshal asset: %w", err)
	}

	// Mutate the active flag and bump timestamps.
	current.Active = req.Active
	now := time.Now().UnixMilli()
	current.DateModified = &now
	dms := time.Now().Format(time.RFC3339)
	current.DateModifiedString = &dms

	// req.Reason captured but not applied — asset has no reason column. Reserved for future audit integration.

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

// AcquireAsset is not implemented in this adapter. It is part of the asset
// lifecycle rpcs that have no current caller in fycha block.go or anywhere
// else this adapter is unblocking. A future plan will design and implement
// it when product needs lifecycle UI.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) AcquireAsset(ctx context.Context, req *assetpb.AcquireAssetRequest) (*assetpb.AcquireAssetResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.AcquireAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// DisposeAsset is not implemented in this adapter. Disposal has financial
// implications (proceeds, write-off journals) that require their own design.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) DisposeAsset(ctx context.Context, req *assetpb.DisposeAssetRequest) (*assetpb.DisposeAssetResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.DisposeAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// TransferAsset is not implemented in this adapter. Transfer modifies
// location_id and emits an audit row; spec deferred until a caller exists.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) TransferAsset(ctx context.Context, req *assetpb.TransferAssetRequest) (*assetpb.TransferAssetResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.TransferAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// RunDepreciation is not implemented in this adapter. The depreciation
// engine is a periodic process that writes thousands of rows per run and
// deserves its own plan.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) RunDepreciation(ctx context.Context, req *assetpb.RunDepreciationRequest) (*assetpb.RunDepreciationResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.RunDepreciation: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// GetDepreciationSchedule is not implemented in this adapter. It pairs
// with RunDepreciation and ships in the same future plan.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) GetDepreciationSchedule(ctx context.Context, req *assetpb.GetDepreciationScheduleRequest) (*assetpb.GetDepreciationScheduleResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.GetDepreciationSchedule: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// RevalueAsset is not implemented in this adapter. IFRS revaluation is
// niche and product-input-dependent; defer until a caller exists.
//
// TODO(asset-stack): implement when caller exists; see docs/plan/20260503-asset-typed-stack-buildout/plan.md
func (r *PostgresAssetRepository) RevalueAsset(ctx context.Context, req *assetpb.RevalueAssetRequest) (*assetpb.RevalueAssetResponse, error) {
	return nil, fmt.Errorf("PostgresAssetRepository.RevalueAsset: not implemented; see docs/plan/20260503-asset-typed-stack-buildout/plan.md")
}

// NewAssetRepository creates a new PostgreSQL asset repository (old-style constructor).
func NewAssetRepository(db *sql.DB, tableName string) assetpb.AssetDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresAssetRepository(dbOps, tableName)
}
