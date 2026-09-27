//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	leaverequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_request"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.LeaveRequest, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres leave_request repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresLeaveRequestRepository(dbOps, tableName), nil
	})
}

// PostgresLeaveRequestRepository implements leave request CRUD operations using PostgreSQL.
type PostgresLeaveRequestRepository struct {
	leaverequestpb.UnimplementedLeaveRequestDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresLeaveRequestRepository creates a new PostgreSQL leave request repository.
func NewPostgresLeaveRequestRepository(dbOps interfaces.DatabaseOperation, tableName string) leaverequestpb.LeaveRequestDomainServiceServer {
	if tableName == "" {
		tableName = "leave_request"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresLeaveRequestRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLeaveRequest creates a new leave request record.
func (r *PostgresLeaveRequestRepository) CreateLeaveRequest(ctx context.Context, req *leaverequestpb.CreateLeaveRequestRequest) (*leaverequestpb.CreateLeaveRequestResponse, error) {
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
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveRequestRepository) ReadLeaveRequest(ctx context.Context, req *leaverequestpb.ReadLeaveRequestRequest) (*leaverequestpb.ReadLeaveRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave request ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_request: %w", err)
	}
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveRequestRepository) UpdateLeaveRequest(ctx context.Context, req *leaverequestpb.UpdateLeaveRequestRequest) (*leaverequestpb.UpdateLeaveRequestResponse, error) {
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
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveRequestRepository) DeleteLeaveRequest(ctx context.Context, req *leaverequestpb.DeleteLeaveRequestRequest) (*leaverequestpb.DeleteLeaveRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave request ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete leave_request: %w", err)
	}
	return &leaverequestpb.DeleteLeaveRequestResponse{Success: true}, nil
}

// ListLeaveRequests lists leave request records with optional filters.
func (r *PostgresLeaveRequestRepository) ListLeaveRequests(ctx context.Context, req *leaverequestpb.ListLeaveRequestsRequest) (*leaverequestpb.ListLeaveRequestsResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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

// leaveRequestSortableSQLCols is the A2 sort whitelist for leave_request list pages.
var leaveRequestSortableSQLCols = []string{
	"lr.id", "lr.workspace_id", "lr.supplier_id", "lr.leave_type_id",
	"lr.start_date", "lr.end_date", "lr.days",
	"lr.approved_by_user_id", "lr.approved_on",
	"lr.active", "lr.date_created", "lr.date_modified",
}

// GetLeaveRequestListPageData retrieves leave requests with pagination, filtering, sorting, and search.
// A1: workspace_id = $1 (strict, from context).
// A2: sort column whitelisted via core.BuildOrderBy.
// A3: exact total comes from the same scoped relation as the page and boundary.
func (r *PostgresLeaveRequestRepository) GetLeaveRequestListPageData(
	ctx context.Context,
	req *leaverequestpb.GetLeaveRequestListPageDataRequest,
) (*leaverequestpb.GetLeaveRequestListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get leave request list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("GetLeaveRequestListPageData requires raw *sql.DB")
	}

	// A1: strict workspace predicate.
	workspaceID := identity.Must(ctx).WorkspaceID

	pageRequest, err := postgresCore.BoundedPageRequest(req.GetPagination(), 50)
	if err != nil {
		return nil, err
	}

	// A2: sort guard — fail-closed via core.BuildOrderBy whitelist.
	orderByClause, err := postgresCore.BuildOrderBy(leaveRequestSortableSQLCols, req.GetSort(), "lr.date_created DESC")
	if err != nil {
		return nil, err
	}

	// The scoped relation is counted separately so empty pages retain the exact total.
	scopedSQL := fmt.Sprintf(`
		SELECT
			lr.id,
			lr.workspace_id,
			lr.supplier_id,
			lr.leave_type_id,
			lr.start_date,
			lr.end_date,
			lr.days,
			lr.approved_by_user_id,
			lr.reason,
			lr.approved_on,
			lr.active,
			lr.date_created,
			lr.date_modified
		FROM %s lr
		WHERE lr.workspace_id = $1
	`, r.tableName)

	sortKeys, err := pageSortFromOrderBy(orderByClause)
	if err != nil {
		return nil, err
	}
	set := postgresCore.ScopedPageSet{SQL: scopedSQL, Args: []any{workspaceID}, Sort: sortKeys}
	queries, err := postgresCore.ResolveScopedPage(ctx, r.db, set, pageRequest)
	if err != nil {
		return nil, fmt.Errorf("resolve leave_request page: %w", err)
	}
	var totalCount int64
	if err := r.db.QueryRowContext(ctx, queries.CountSQL, queries.CountArgs...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count leave_request page: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, queries.PageSQL, queries.PageArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leave_request list page data: %w", err)
	}
	defer rows.Close()

	var items []*leaverequestpb.LeaveRequest
	for rows.Next() {
		var (
			id               string
			wsID             string
			supplierID       string
			leaveTypeID      string
			startDate        string
			endDate          string
			days             int32
			approvedByUserID *string
			reason           *string
			approvedOn       *string
			active           bool
			dateCreated      *time.Time
			dateModified     *time.Time
		)
		if scanErr := rows.Scan(
			&id, &wsID, &supplierID, &leaveTypeID,
			&startDate, &endDate, &days,
			&approvedByUserID, &reason, &approvedOn,
			&active, &dateCreated, &dateModified,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan leave_request row: %w", scanErr)
		}

		lr := &leaverequestpb.LeaveRequest{
			Id:               id,
			WorkspaceId:      wsID,
			SupplierId:       supplierID,
			LeaveTypeId:      leaveTypeID,
			StartDate:        startDate,
			EndDate:          endDate,
			Days:             days,
			ApprovedByUserId: approvedByUserID,
			Reason:           reason,
			ApprovedOn:       approvedOn,
			Active:           active,
			DateCreated:      pageMillis(dateCreated),
			DateModified:     pageMillis(dateModified),
		}
		items = append(items, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leave_request rows: %w", err)
	}

	firstID, lastID := "", ""
	if len(items) > 0 {
		firstID = items[0].GetId()
		lastID = items[len(items)-1].GetId()
	}

	return &leaverequestpb.GetLeaveRequestListPageDataResponse{
		LeaveRequestList: items,
		Pagination:       scopedPageMetadata(totalCount, queries.Page, firstID, lastID),
		Success:          true,
	}, nil
}

// GetLeaveRequestItemPageData retrieves a single leave request.
func (r *PostgresLeaveRequestRepository) GetLeaveRequestItemPageData(
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
	resultJSON, err := json.Marshal(result)
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
