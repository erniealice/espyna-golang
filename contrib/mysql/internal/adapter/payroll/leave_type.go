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
	leavetypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_type"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.LeaveType, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql leave_type repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLLeaveTypeRepository(dbOps, tableName), nil
	})
}

// MySQLLeaveTypeRepository implements leave type CRUD operations using MySQL 8.0+.
type MySQLLeaveTypeRepository struct {
	leavetypepb.UnimplementedLeaveTypeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLLeaveTypeRepository creates a new MySQL leave type repository.
func NewMySQLLeaveTypeRepository(dbOps interfaces.DatabaseOperation, tableName string) leavetypepb.LeaveTypeDomainServiceServer {
	if tableName == "" {
		tableName = "leave_type"
	}
	return &MySQLLeaveTypeRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLeaveType creates a new leave type record.
func (r *MySQLLeaveTypeRepository) CreateLeaveType(ctx context.Context, req *leavetypepb.CreateLeaveTypeRequest) (*leavetypepb.CreateLeaveTypeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("leave type data is required")
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
		return nil, fmt.Errorf("failed to create leave_type: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lt := &leavetypepb.LeaveType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavetypepb.CreateLeaveTypeResponse{Success: true, Data: []*leavetypepb.LeaveType{lt}}, nil
}

// ReadLeaveType retrieves a leave type by ID.
func (r *MySQLLeaveTypeRepository) ReadLeaveType(ctx context.Context, req *leavetypepb.ReadLeaveTypeRequest) (*leavetypepb.ReadLeaveTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave type ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_type: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lt := &leavetypepb.LeaveType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavetypepb.ReadLeaveTypeResponse{Success: true, Data: []*leavetypepb.LeaveType{lt}}, nil
}

// UpdateLeaveType updates a leave type record.
func (r *MySQLLeaveTypeRepository) UpdateLeaveType(ctx context.Context, req *leavetypepb.UpdateLeaveTypeRequest) (*leavetypepb.UpdateLeaveTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave type ID is required")
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
		return nil, fmt.Errorf("failed to update leave_type: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lt := &leavetypepb.LeaveType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavetypepb.UpdateLeaveTypeResponse{Success: true, Data: []*leavetypepb.LeaveType{lt}}, nil
}

// DeleteLeaveType soft-deletes a leave type.
func (r *MySQLLeaveTypeRepository) DeleteLeaveType(ctx context.Context, req *leavetypepb.DeleteLeaveTypeRequest) (*leavetypepb.DeleteLeaveTypeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave type ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete leave_type: %w", err)
	}
	return &leavetypepb.DeleteLeaveTypeResponse{Success: true}, nil
}

// ListLeaveTypes lists leave type records with optional filters.
func (r *MySQLLeaveTypeRepository) ListLeaveTypes(ctx context.Context, req *leavetypepb.ListLeaveTypesRequest) (*leavetypepb.ListLeaveTypesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list leave_types: %w", err)
	}
	var items []*leavetypepb.LeaveType
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_type row: %v", err)
			continue
		}
		lt := &leavetypepb.LeaveType{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
			log.Printf("WARN: protojson unmarshal leave_type: %v", err)
			continue
		}
		items = append(items, lt)
	}
	return &leavetypepb.ListLeaveTypesResponse{Success: true, Data: items}, nil
}

// GetLeaveTypeListPageData retrieves leave types with pagination, filtering, sorting, and search.
func (r *MySQLLeaveTypeRepository) GetLeaveTypeListPageData(
	ctx context.Context,
	req *leavetypepb.GetLeaveTypeListPageDataRequest,
) (*leavetypepb.GetLeaveTypeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get leave type list page data request is required")
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
		return nil, fmt.Errorf("failed to list leave_type list page data: %w", err)
	}

	var items []*leavetypepb.LeaveType
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_type row: %v", err)
			continue
		}
		lt := &leavetypepb.LeaveType{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
			log.Printf("WARN: protojson unmarshal leave_type: %v", err)
			continue
		}
		items = append(items, lt)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &leavetypepb.GetLeaveTypeListPageDataResponse{
		LeaveTypeList: items,
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

// GetLeaveTypeItemPageData retrieves a single leave type.
func (r *MySQLLeaveTypeRepository) GetLeaveTypeItemPageData(
	ctx context.Context,
	req *leavetypepb.GetLeaveTypeItemPageDataRequest,
) (*leavetypepb.GetLeaveTypeItemPageDataResponse, error) {
	if req == nil || req.GetLeaveTypeId() == "" {
		return nil, fmt.Errorf("leave type ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetLeaveTypeId())
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_type item: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lt := &leavetypepb.LeaveType{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavetypepb.GetLeaveTypeItemPageDataResponse{
		LeaveType: lt,
		Success:   true,
	}, nil
}
