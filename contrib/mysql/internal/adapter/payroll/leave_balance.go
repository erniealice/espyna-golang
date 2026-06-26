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
	leavebalancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_balance"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.LeaveBalance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql leave_balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLLeaveBalanceRepository(dbOps, tableName), nil
	})
}

// MySQLLeaveBalanceRepository implements leave balance CRUD operations using MySQL 8.0+.
type MySQLLeaveBalanceRepository struct {
	leavebalancepb.UnimplementedLeaveBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLLeaveBalanceRepository creates a new MySQL leave balance repository.
func NewMySQLLeaveBalanceRepository(dbOps interfaces.DatabaseOperation, tableName string) leavebalancepb.LeaveBalanceDomainServiceServer {
	if tableName == "" {
		tableName = "leave_balance"
	}
	return &MySQLLeaveBalanceRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLeaveBalance creates a new leave balance record.
func (r *MySQLLeaveBalanceRepository) CreateLeaveBalance(ctx context.Context, req *leavebalancepb.CreateLeaveBalanceRequest) (*leavebalancepb.CreateLeaveBalanceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("leave balance data is required")
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
		return nil, fmt.Errorf("failed to create leave_balance: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lb := &leavebalancepb.LeaveBalance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavebalancepb.CreateLeaveBalanceResponse{Success: true, Data: []*leavebalancepb.LeaveBalance{lb}}, nil
}

// ReadLeaveBalance retrieves a leave balance by ID.
func (r *MySQLLeaveBalanceRepository) ReadLeaveBalance(ctx context.Context, req *leavebalancepb.ReadLeaveBalanceRequest) (*leavebalancepb.ReadLeaveBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave balance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_balance: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lb := &leavebalancepb.LeaveBalance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavebalancepb.ReadLeaveBalanceResponse{Success: true, Data: []*leavebalancepb.LeaveBalance{lb}}, nil
}

// UpdateLeaveBalance updates a leave balance record.
func (r *MySQLLeaveBalanceRepository) UpdateLeaveBalance(ctx context.Context, req *leavebalancepb.UpdateLeaveBalanceRequest) (*leavebalancepb.UpdateLeaveBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave balance ID is required")
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
		return nil, fmt.Errorf("failed to update leave_balance: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lb := &leavebalancepb.LeaveBalance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavebalancepb.UpdateLeaveBalanceResponse{Success: true, Data: []*leavebalancepb.LeaveBalance{lb}}, nil
}

// DeleteLeaveBalance soft-deletes a leave balance.
func (r *MySQLLeaveBalanceRepository) DeleteLeaveBalance(ctx context.Context, req *leavebalancepb.DeleteLeaveBalanceRequest) (*leavebalancepb.DeleteLeaveBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave balance ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete leave_balance: %w", err)
	}
	return &leavebalancepb.DeleteLeaveBalanceResponse{Success: true}, nil
}

// ListLeaveBalances lists leave balance records with optional filters.
func (r *MySQLLeaveBalanceRepository) ListLeaveBalances(ctx context.Context, req *leavebalancepb.ListLeaveBalancesRequest) (*leavebalancepb.ListLeaveBalancesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list leave_balances: %w", err)
	}
	var items []*leavebalancepb.LeaveBalance
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_balance row: %v", err)
			continue
		}
		lb := &leavebalancepb.LeaveBalance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
			log.Printf("WARN: protojson unmarshal leave_balance: %v", err)
			continue
		}
		items = append(items, lb)
	}
	return &leavebalancepb.ListLeaveBalancesResponse{Success: true, Data: items}, nil
}

// GetLeaveBalanceListPageData retrieves leave balances with pagination, filtering, sorting, and search.
func (r *MySQLLeaveBalanceRepository) GetLeaveBalanceListPageData(
	ctx context.Context,
	req *leavebalancepb.GetLeaveBalanceListPageDataRequest,
) (*leavebalancepb.GetLeaveBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get leave balance list page data request is required")
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
		return nil, fmt.Errorf("failed to list leave_balance list page data: %w", err)
	}

	var items []*leavebalancepb.LeaveBalance
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal leave_balance row: %v", err)
			continue
		}
		lb := &leavebalancepb.LeaveBalance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
			log.Printf("WARN: protojson unmarshal leave_balance: %v", err)
			continue
		}
		items = append(items, lb)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &leavebalancepb.GetLeaveBalanceListPageDataResponse{
		LeaveBalanceList: items,
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

// GetLeaveBalanceItemPageData retrieves a single leave balance.
func (r *MySQLLeaveBalanceRepository) GetLeaveBalanceItemPageData(
	ctx context.Context,
	req *leavebalancepb.GetLeaveBalanceItemPageDataRequest,
) (*leavebalancepb.GetLeaveBalanceItemPageDataResponse, error) {
	if req == nil || req.GetLeaveBalanceId() == "" {
		return nil, fmt.Errorf("leave balance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetLeaveBalanceId())
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_balance item: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	lb := &leavebalancepb.LeaveBalance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &leavebalancepb.GetLeaveBalanceItemPageDataResponse{
		LeaveBalance: lb,
		Success:      true,
	}, nil
}
