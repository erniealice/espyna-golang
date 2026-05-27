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
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.DepreciationSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver depreciation_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerDepreciationScheduleRepository(dbOps, tableName), nil
	})
}

var depreciationScheduleSortableSQLCols = []string{
	"id", "asset_id", "period_start_date", "period_end_date",
	"method", "amount", "accumulated_depreciation", "book_value_after",
	"depreciation_run_id", "journal_entry_id",
	"active", "date_created", "date_modified",
}

var depreciationScheduleSortSpec = espynahttp.SortSpec{AllowedCols: depreciationScheduleSortableSQLCols}

// SQLServerDepreciationScheduleRepository implements depreciation_schedule CRUD
// operations using SQL Server.
type SQLServerDepreciationScheduleRepository struct {
	depschpb.UnimplementedDepreciationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerDepreciationScheduleRepository creates a new SQL Server depreciation_schedule repository.
func NewSQLServerDepreciationScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) depschpb.DepreciationDomainServiceServer {
	if tableName == "" {
		tableName = "depreciation_schedule"
	}
	return &SQLServerDepreciationScheduleRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDepreciationSchedule inserts a new depreciation_schedule row.
func (r *SQLServerDepreciationScheduleRepository) CreateDepreciationSchedule(ctx context.Context, req *depschpb.CreateDepreciationScheduleRequest) (*depschpb.CreateDepreciationScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("depreciation_schedule data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_schedule protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_schedule JSON: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create depreciation_schedule: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_schedule result: %w", err)
	}

	sched := &depschpb.DepreciationSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sched); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_schedule result: %w", err)
	}

	return &depschpb.CreateDepreciationScheduleResponse{
		Data:    []*depschpb.DepreciationSchedule{sched},
		Success: true,
	}, nil
}

// ReadDepreciationSchedule retrieves a single depreciation_schedule row by ID.
func (r *SQLServerDepreciationScheduleRepository) ReadDepreciationSchedule(ctx context.Context, req *depschpb.ReadDepreciationScheduleRequest) (*depschpb.ReadDepreciationScheduleResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read depreciation_schedule: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("depreciation_schedule with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_schedule result: %w", err)
	}

	sched := &depschpb.DepreciationSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sched); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_schedule result: %w", err)
	}

	return &depschpb.ReadDepreciationScheduleResponse{
		Data:    []*depschpb.DepreciationSchedule{sched},
		Success: true,
	}, nil
}

// UpdateDepreciationSchedule updates a depreciation_schedule row (admin-level only).
func (r *SQLServerDepreciationScheduleRepository) UpdateDepreciationSchedule(ctx context.Context, req *depschpb.UpdateDepreciationScheduleRequest) (*depschpb.UpdateDepreciationScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_schedule ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_schedule protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_schedule JSON: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update depreciation_schedule: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_schedule result: %w", err)
	}

	sched := &depschpb.DepreciationSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sched); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_schedule result: %w", err)
	}

	return &depschpb.UpdateDepreciationScheduleResponse{
		Data:    []*depschpb.DepreciationSchedule{sched},
		Success: true,
	}, nil
}

// DeleteDepreciationSchedule soft-deletes a depreciation_schedule row.
func (r *SQLServerDepreciationScheduleRepository) DeleteDepreciationSchedule(ctx context.Context, req *depschpb.DeleteDepreciationScheduleRequest) (*depschpb.DeleteDepreciationScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_schedule ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete depreciation_schedule: %w", err)
	}

	return &depschpb.DeleteDepreciationScheduleResponse{
		Success: true,
	}, nil
}

// ListDepreciationSchedules lists depreciation_schedule rows using SQL Server operations.
func (r *SQLServerDepreciationScheduleRepository) ListDepreciationSchedules(ctx context.Context, req *depschpb.ListDepreciationSchedulesRequest) (*depschpb.ListDepreciationSchedulesResponse, error) {
	if err := espynahttp.ValidateSortColumns(depreciationScheduleSortSpec, req.GetSort(), "depreciation_schedule"); err != nil {
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
		return nil, fmt.Errorf("failed to list depreciation_schedules: %w", err)
	}

	var scheds []*depschpb.DepreciationSchedule
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		sched := &depschpb.DepreciationSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sched); err != nil {
			continue
		}
		scheds = append(scheds, sched)
	}

	return &depschpb.ListDepreciationSchedulesResponse{
		Data:    scheds,
		Success: true,
	}, nil
}

// GetDepreciationScheduleListPageData retrieves depreciation_schedules with pagination metadata.
func (r *SQLServerDepreciationScheduleRepository) GetDepreciationScheduleListPageData(
	ctx context.Context,
	req *depschpb.GetDepreciationScheduleListPageDataRequest,
) (*depschpb.GetDepreciationScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get depreciation_schedule list page data request is required")
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

	listResp, err := r.ListDepreciationSchedules(ctx, &depschpb.ListDepreciationSchedulesRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list depreciation_schedules for page data: %w", err)
	}
	scheds := listResp.GetData()

	totalItems := int32(len(scheds))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &depschpb.GetDepreciationScheduleListPageDataResponse{
		DepreciationScheduleList: scheds,
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

// GetDepreciationScheduleItemPageData retrieves a single depreciation_schedule via
// composition over ReadDepreciationSchedule.
func (r *SQLServerDepreciationScheduleRepository) GetDepreciationScheduleItemPageData(
	ctx context.Context,
	req *depschpb.GetDepreciationScheduleItemPageDataRequest,
) (*depschpb.GetDepreciationScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get depreciation_schedule item page data request is required")
	}
	if req.DepreciationScheduleId == "" {
		return nil, fmt.Errorf("depreciation_schedule ID is required")
	}

	rr, err := r.ReadDepreciationSchedule(ctx, &depschpb.ReadDepreciationScheduleRequest{Data: &depschpb.DepreciationSchedule{Id: req.DepreciationScheduleId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("depreciation_schedule with ID '%s' not found", req.DepreciationScheduleId)
	}

	return &depschpb.GetDepreciationScheduleItemPageDataResponse{
		DepreciationSchedule: rr.GetData()[0],
		Success:              true,
	}, nil
}

// NewDepreciationScheduleRepository creates a new SQL Server depreciation_schedule repository (old-style constructor).
func NewDepreciationScheduleRepository(db *sql.DB, tableName string) depschpb.DepreciationDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerDepreciationScheduleRepository(dbOps, tableName)
}
