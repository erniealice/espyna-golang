//go:build mysql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.RateBand, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql rate_band repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRateBandRepository(dbOps, tableName), nil
	})
}

// MySQLRateBandRepository implements rate band CRUD operations using MySQL 8.0+.
type MySQLRateBandRepository struct {
	ratebandpb.UnimplementedRateBandDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLRateBandRepository creates a new MySQL rate band repository.
func NewMySQLRateBandRepository(dbOps interfaces.DatabaseOperation, tableName string) ratebandpb.RateBandDomainServiceServer {
	if tableName == "" {
		tableName = "rate_band"
	}
	return &MySQLRateBandRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateRateBand creates a new rate band record.
func (r *MySQLRateBandRepository) CreateRateBand(ctx context.Context, req *ratebandpb.CreateRateBandRequest) (*ratebandpb.CreateRateBandResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rate band data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.CreateRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// ReadRateBand retrieves a rate band by ID.
func (r *MySQLRateBandRepository) ReadRateBand(ctx context.Context, req *ratebandpb.ReadRateBandRequest) (*ratebandpb.ReadRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.ReadRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// UpdateRateBand updates a rate band record.
func (r *MySQLRateBandRepository) UpdateRateBand(ctx context.Context, req *ratebandpb.UpdateRateBandRequest) (*ratebandpb.UpdateRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.UpdateRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// DeleteRateBand soft-deletes a rate band.
func (r *MySQLRateBandRepository) DeleteRateBand(ctx context.Context, req *ratebandpb.DeleteRateBandRequest) (*ratebandpb.DeleteRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rate_band: %w", err)
	}
	return &ratebandpb.DeleteRateBandResponse{Success: true}, nil
}

// ListRateBands lists rate band records with optional filters.
func (r *MySQLRateBandRepository) ListRateBands(ctx context.Context, req *ratebandpb.ListRateBandsRequest) (*ratebandpb.ListRateBandsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate_bands: %w", err)
	}
	var items []*ratebandpb.RateBand
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal rate_band row: %v", err)
			continue
		}
		rb := &ratebandpb.RateBand{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
			log.Printf("WARN: protojson unmarshal rate_band: %v", err)
			continue
		}
		items = append(items, rb)
	}
	return &ratebandpb.ListRateBandsResponse{Success: true, Data: items}, nil
}

// GetRateBandListPageData retrieves rate bands with pagination, filtering, sorting, and search.
func (r *MySQLRateBandRepository) GetRateBandListPageData(
	ctx context.Context,
	req *ratebandpb.GetRateBandListPageDataRequest,
) (*ratebandpb.GetRateBandListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get rate band list page data request is required")
	}

	var params *interfaces.ListParams
	if req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
			}
		}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate_band list page data: %w", err)
	}

	var items []*ratebandpb.RateBand
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal rate_band row: %v", err)
			continue
		}
		rb := &ratebandpb.RateBand{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
			log.Printf("WARN: protojson unmarshal rate_band: %v", err)
			continue
		}
		items = append(items, rb)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &ratebandpb.GetRateBandListPageDataResponse{
		RateBandList: items,
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

// GetRateBandItemPageData retrieves a single rate band.
func (r *MySQLRateBandRepository) GetRateBandItemPageData(
	ctx context.Context,
	req *ratebandpb.GetRateBandItemPageDataRequest,
) (*ratebandpb.GetRateBandItemPageDataResponse, error) {
	if req == nil || req.GetRateBandId() == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetRateBandId())
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_band item: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.GetRateBandItemPageDataResponse{
		RateBand: rb,
		Success:  true,
	}, nil
}
