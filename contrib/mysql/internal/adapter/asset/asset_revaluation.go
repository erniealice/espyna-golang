//go:build mysql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	revaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.AssetRevaluation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql asset_revaluation repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLAssetRevaluationRepository(dbOps, tableName), nil
	})
}

var assetRevaluationSortableSQLCols = []string{
	"id", "asset_id", "revaluation_date", "fair_value",
	"prior_book_value", "revaluation_amount",
	"recognized_in_pnl", "recognized_in_oci",
	"performed_by",
	"active", "date_created", "date_modified",
}

var assetRevaluationSortSpec = espynahttp.SortSpec{AllowedCols: assetRevaluationSortableSQLCols}

// MySQLAssetRevaluationRepository implements asset_revaluation CRUD operations
// using MySQL 8.0+. Mirrors postgres/internal/adapter/asset/asset_revaluation.go
// with dialect translation: backtick identifiers, positional ?, no RETURNING.
type MySQLAssetRevaluationRepository struct {
	revaluationpb.UnimplementedAssetRevaluationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLAssetRevaluationRepository creates a new MySQL asset_revaluation repository.
func NewMySQLAssetRevaluationRepository(dbOps interfaces.DatabaseOperation, tableName string) revaluationpb.AssetRevaluationDomainServiceServer {
	if tableName == "" {
		tableName = "asset_revaluation"
	}
	return &MySQLAssetRevaluationRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAssetRevaluation inserts a new asset_revaluation row.
func (r *MySQLAssetRevaluationRepository) CreateAssetRevaluation(ctx context.Context, req *revaluationpb.CreateAssetRevaluationRequest) (*revaluationpb.CreateAssetRevaluationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("asset_revaluation data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_revaluation protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_revaluation JSON: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset_revaluation: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_revaluation result: %w", err)
	}

	rev := &revaluationpb.AssetRevaluation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_revaluation result: %w", err)
	}

	return &revaluationpb.CreateAssetRevaluationResponse{
		Data:    []*revaluationpb.AssetRevaluation{rev},
		Success: true,
	}, nil
}

// ReadAssetRevaluation retrieves a single asset_revaluation row by ID.
func (r *MySQLAssetRevaluationRepository) ReadAssetRevaluation(ctx context.Context, req *revaluationpb.ReadAssetRevaluationRequest) (*revaluationpb.ReadAssetRevaluationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_revaluation ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset_revaluation: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("asset_revaluation with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_revaluation result: %w", err)
	}

	rev := &revaluationpb.AssetRevaluation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_revaluation result: %w", err)
	}

	return &revaluationpb.ReadAssetRevaluationResponse{
		Data:    []*revaluationpb.AssetRevaluation{rev},
		Success: true,
	}, nil
}

// UpdateAssetRevaluation patches an asset_revaluation row (admin-level only).
func (r *MySQLAssetRevaluationRepository) UpdateAssetRevaluation(ctx context.Context, req *revaluationpb.UpdateAssetRevaluationRequest) (*revaluationpb.UpdateAssetRevaluationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_revaluation ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_revaluation protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_revaluation JSON: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update asset_revaluation: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_revaluation result: %w", err)
	}

	rev := &revaluationpb.AssetRevaluation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_revaluation result: %w", err)
	}

	return &revaluationpb.UpdateAssetRevaluationResponse{
		Data:    []*revaluationpb.AssetRevaluation{rev},
		Success: true,
	}, nil
}

// DeleteAssetRevaluation soft-deletes an asset_revaluation row.
func (r *MySQLAssetRevaluationRepository) DeleteAssetRevaluation(ctx context.Context, req *revaluationpb.DeleteAssetRevaluationRequest) (*revaluationpb.DeleteAssetRevaluationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_revaluation ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete asset_revaluation: %w", err)
	}

	return &revaluationpb.DeleteAssetRevaluationResponse{
		Success: true,
	}, nil
}

// ListAssetRevaluations lists asset_revaluation rows.
func (r *MySQLAssetRevaluationRepository) ListAssetRevaluations(ctx context.Context, req *revaluationpb.ListAssetRevaluationsRequest) (*revaluationpb.ListAssetRevaluationsResponse, error) {
	if err := espynahttp.ValidateSortColumns(assetRevaluationSortSpec, req.GetSort(), "asset_revaluation"); err != nil {
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
		return nil, fmt.Errorf("failed to list asset_revaluations: %w", err)
	}

	var revs []*revaluationpb.AssetRevaluation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		rev := &revaluationpb.AssetRevaluation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rev); err != nil {
			continue
		}
		revs = append(revs, rev)
	}

	return &revaluationpb.ListAssetRevaluationsResponse{
		Data:    revs,
		Success: true,
	}, nil
}

// GetAssetRevaluationListPageData retrieves asset_revaluations with pagination metadata.
func (r *MySQLAssetRevaluationRepository) GetAssetRevaluationListPageData(
	ctx context.Context,
	req *revaluationpb.GetAssetRevaluationListPageDataRequest,
) (*revaluationpb.GetAssetRevaluationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_revaluation list page data request is required")
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

	listResp, err := r.ListAssetRevaluations(ctx, &revaluationpb.ListAssetRevaluationsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list asset_revaluations for page data: %w", err)
	}
	revs := listResp.GetData()

	totalItems := int32(len(revs))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &revaluationpb.GetAssetRevaluationListPageDataResponse{
		AssetRevaluationList: revs,
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

// GetAssetRevaluationItemPageData retrieves a single asset_revaluation via
// composition over ReadAssetRevaluation.
func (r *MySQLAssetRevaluationRepository) GetAssetRevaluationItemPageData(
	ctx context.Context,
	req *revaluationpb.GetAssetRevaluationItemPageDataRequest,
) (*revaluationpb.GetAssetRevaluationItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_revaluation item page data request is required")
	}
	if req.AssetRevaluationId == "" {
		return nil, fmt.Errorf("asset_revaluation ID is required")
	}

	rr, err := r.ReadAssetRevaluation(ctx, &revaluationpb.ReadAssetRevaluationRequest{Data: &revaluationpb.AssetRevaluation{Id: req.AssetRevaluationId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("asset_revaluation with ID '%s' not found", req.AssetRevaluationId)
	}

	return &revaluationpb.GetAssetRevaluationItemPageDataResponse{
		AssetRevaluation: rr.GetData()[0],
		Success:          true,
	}, nil
}

// NewAssetRevaluationRepository creates a new MySQL asset_revaluation repository (old-style constructor).
func NewAssetRevaluationRepository(db *sql.DB, tableName string) revaluationpb.AssetRevaluationDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLAssetRevaluationRepository(dbOps, tableName)
}
