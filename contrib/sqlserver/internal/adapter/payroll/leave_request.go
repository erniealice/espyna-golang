//go:build sqlserver

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	leaverequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_request"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.LeaveRequest, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver leave_request repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerLeaveRequestRepository(dbOps, tableName), nil
	})
}

// SQLServerLeaveRequestRepository implements leave request CRUD operations using SQL Server.
type SQLServerLeaveRequestRepository struct {
	leaverequestpb.UnimplementedLeaveRequestDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerLeaveRequestRepository creates a new SQL Server leave request repository.
func NewSQLServerLeaveRequestRepository(dbOps interfaces.DatabaseOperation, tableName string) leaverequestpb.LeaveRequestDomainServiceServer {
	if tableName == "" {
		tableName = "leave_request"
	}
	return &SQLServerLeaveRequestRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLeaveRequest creates a new leave request record.
func (r *SQLServerLeaveRequestRepository) CreateLeaveRequest(ctx context.Context, req *leaverequestpb.CreateLeaveRequestRequest) (*leaverequestpb.CreateLeaveRequestResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("leave request data is required")
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
		return nil, fmt.Errorf("failed to create leave_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lr := &leaverequestpb.LeaveRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leaverequestpb.CreateLeaveRequestResponse{Success: true, Data: []*leaverequestpb.LeaveRequest{lr}}, nil
}

// ReadLeaveRequest retrieves a leave request by ID.
func (r *SQLServerLeaveRequestRepository) ReadLeaveRequest(ctx context.Context, req *leaverequestpb.ReadLeaveRequestRequest) (*leaverequestpb.ReadLeaveRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave request ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lr := &leaverequestpb.LeaveRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leaverequestpb.ReadLeaveRequestResponse{Success: true, Data: []*leaverequestpb.LeaveRequest{lr}}, nil
}

// UpdateLeaveRequest updates a leave request record.
func (r *SQLServerLeaveRequestRepository) UpdateLeaveRequest(ctx context.Context, req *leaverequestpb.UpdateLeaveRequestRequest) (*leaverequestpb.UpdateLeaveRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave request ID is required")
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
		return nil, fmt.Errorf("failed to update leave_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lr := &leaverequestpb.LeaveRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leaverequestpb.UpdateLeaveRequestResponse{Success: true, Data: []*leaverequestpb.LeaveRequest{lr}}, nil
}

// DeleteLeaveRequest soft-deletes a leave request.
func (r *SQLServerLeaveRequestRepository) DeleteLeaveRequest(ctx context.Context, req *leaverequestpb.DeleteLeaveRequestRequest) (*leaverequestpb.DeleteLeaveRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave request ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete leave_request: %w", err)
	}
	return &leaverequestpb.DeleteLeaveRequestResponse{Success: true}, nil
}

// ListLeaveRequests lists leave request records with optional filters.
func (r *SQLServerLeaveRequestRepository) ListLeaveRequests(ctx context.Context, req *leaverequestpb.ListLeaveRequestsRequest) (*leaverequestpb.ListLeaveRequestsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list leave_requests: %w", err)
	}
	var items []*leaverequestpb.LeaveRequest
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_request row: %v", err)
			continue
		}
		lr := &leaverequestpb.LeaveRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
			log.Printf("WARN: protojson unmarshal leave_request: %v", err)
			continue
		}
		items = append(items, lr)
	}
	return &leaverequestpb.ListLeaveRequestsResponse{Success: true, Data: items}, nil
}

// GetLeaveRequestListPageData retrieves leave requests with pagination, filtering, sorting, and search.
func (r *SQLServerLeaveRequestRepository) GetLeaveRequestListPageData(
	ctx context.Context,
	req *leaverequestpb.GetLeaveRequestListPageDataRequest,
) (*leaverequestpb.GetLeaveRequestListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get leave request list page data request is required")
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
		return nil, fmt.Errorf("failed to list leave_request list page data: %w", err)
	}

	var items []*leaverequestpb.LeaveRequest
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_request row: %v", err)
			continue
		}
		lr := &leaverequestpb.LeaveRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
			log.Printf("WARN: protojson unmarshal leave_request: %v", err)
			continue
		}
		items = append(items, lr)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &leaverequestpb.GetLeaveRequestListPageDataResponse{
		LeaveRequestList: items,
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

// GetLeaveRequestItemPageData retrieves a single leave request.
func (r *SQLServerLeaveRequestRepository) GetLeaveRequestItemPageData(
	ctx context.Context,
	req *leaverequestpb.GetLeaveRequestItemPageDataRequest,
) (*leaverequestpb.GetLeaveRequestItemPageDataResponse, error) {
	if req == nil || req.GetLeaveRequestId() == "" {
		return nil, fmt.Errorf("leave request ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetLeaveRequestId())
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_request item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lr := &leaverequestpb.LeaveRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leaverequestpb.GetLeaveRequestItemPageDataResponse{
		LeaveRequest: lr,
		Success:      true,
	}, nil
}
