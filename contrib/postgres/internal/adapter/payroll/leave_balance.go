//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	leavebalancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_balance"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.LeaveBalance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres leave_balance repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresLeaveBalanceRepository(dbOps, tableName), nil
	})
}

// PostgresLeaveBalanceRepository implements leave balance CRUD operations using PostgreSQL.
type PostgresLeaveBalanceRepository struct {
	leavebalancepb.UnimplementedLeaveBalanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresLeaveBalanceRepository creates a new PostgreSQL leave balance repository.
func NewPostgresLeaveBalanceRepository(dbOps interfaces.DatabaseOperation, tableName string) leavebalancepb.LeaveBalanceDomainServiceServer {
	if tableName == "" {
		tableName = "leave_balance"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresLeaveBalanceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLeaveBalance creates a new leave balance record.
func (r *PostgresLeaveBalanceRepository) CreateLeaveBalance(ctx context.Context, req *leavebalancepb.CreateLeaveBalanceRequest) (*leavebalancepb.CreateLeaveBalanceResponse, error) {
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
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveBalanceRepository) ReadLeaveBalance(ctx context.Context, req *leavebalancepb.ReadLeaveBalanceRequest) (*leavebalancepb.ReadLeaveBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave balance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read leave_balance: %w", err)
	}
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveBalanceRepository) UpdateLeaveBalance(ctx context.Context, req *leavebalancepb.UpdateLeaveBalanceRequest) (*leavebalancepb.UpdateLeaveBalanceResponse, error) {
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
	resultJSON, err := json.Marshal(result)
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
func (r *PostgresLeaveBalanceRepository) DeleteLeaveBalance(ctx context.Context, req *leavebalancepb.DeleteLeaveBalanceRequest) (*leavebalancepb.DeleteLeaveBalanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("leave balance ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete leave_balance: %w", err)
	}
	return &leavebalancepb.DeleteLeaveBalanceResponse{Success: true}, nil
}

// ListLeaveBalances lists leave balance records with optional filters.
func (r *PostgresLeaveBalanceRepository) ListLeaveBalances(ctx context.Context, req *leavebalancepb.ListLeaveBalancesRequest) (*leavebalancepb.ListLeaveBalancesResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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

// leaveBalanceSortableSQLCols is the A2 sort whitelist for leave_balance list pages.
var leaveBalanceSortableSQLCols = []string{
	"lb.id", "lb.workspace_id", "lb.supplier_id", "lb.leave_type_id",
	"lb.year", "lb.accrued_days", "lb.used_days", "lb.carryover_days",
	"lb.last_accrued_on", "lb.active", "lb.date_created", "lb.date_modified",
}

// GetLeaveBalanceListPageData retrieves leave balances with pagination, filtering, sorting, and search.
// A1: workspace_id = $1 (strict, from context).
// A2: sort column whitelisted via core.BuildOrderBy.
// A3: COUNT(*) OVER() for accurate total without a second query.
func (r *PostgresLeaveBalanceRepository) GetLeaveBalanceListPageData(
	ctx context.Context,
	req *leavebalancepb.GetLeaveBalanceListPageDataRequest,
) (*leavebalancepb.GetLeaveBalanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get leave balance list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("GetLeaveBalanceListPageData requires raw *sql.DB")
	}

	// A1: strict workspace predicate.
	workspaceID := identity.Must(ctx).WorkspaceID

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	// A2: sort guard — fail-closed via core.BuildOrderBy whitelist.
	orderByClause, err := postgresCore.BuildOrderBy(leaveBalanceSortableSQLCols, req.GetSort(), "lb.date_created DESC")
	if err != nil {
		return nil, err
	}

	// A3: COUNT(*) OVER() — accurate total in one pass.
	query := fmt.Sprintf(`
		SELECT
			lb.id,
			lb.workspace_id,
			lb.supplier_id,
			lb.leave_type_id,
			lb.year,
			lb.accrued_days,
			lb.used_days,
			lb.carryover_days,
			lb.last_accrued_on,
			lb.active,
			lb.date_created,
			lb.date_modified,
			COUNT(*) OVER() AS total
		FROM %s lb
		WHERE lb.workspace_id = $1
		%s
		LIMIT $2 OFFSET $3;
	`, r.tableName, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query leave_balance list page data: %w", err)
	}
	defer rows.Close()

	var items []*leavebalancepb.LeaveBalance
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			wsID          string
			supplierID    string
			leaveTypeID   string
			year          int32
			accruedDays   int32
			usedDays      int32
			carryoverDays int32
			lastAccruedOn *string
			active        bool
			dateCreated   *int64
			dateModified  *int64
			total         int64
		)
		if scanErr := rows.Scan(
			&id, &wsID, &supplierID, &leaveTypeID,
			&year, &accruedDays, &usedDays, &carryoverDays,
			&lastAccruedOn, &active,
			&dateCreated, &dateModified,
			&total,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan leave_balance row: %w", scanErr)
		}
		totalCount = total

		lb := &leavebalancepb.LeaveBalance{
			Id:            id,
			WorkspaceId:   wsID,
			SupplierId:    supplierID,
			LeaveTypeId:   leaveTypeID,
			Year:          year,
			AccruedDays:   accruedDays,
			UsedDays:      usedDays,
			CarryoverDays: carryoverDays,
			LastAccruedOn: lastAccruedOn,
			Active:        active,
			DateCreated:   dateCreated,
			DateModified:  dateModified,
		}
		items = append(items, lb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leave_balance rows: %w", err)
	}

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
func (r *PostgresLeaveBalanceRepository) GetLeaveBalanceItemPageData(
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
	resultJSON, err := json.Marshal(result)
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
