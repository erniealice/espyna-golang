//go:build sqlserver

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PayrollRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver payroll run repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPayrollRunRepository(db, dbOps, tableName), nil
	})
}

// SQLServerPayrollRunRepository implements payroll run CRUD operations using SQL Server.
type SQLServerPayrollRunRepository struct {
	payrollrunpb.UnimplementedPayrollRunDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerPayrollRunRepository creates a new SQL Server payroll run repository.
func NewSQLServerPayrollRunRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) payrollrunpb.PayrollRunDomainServiceServer {
	if tableName == "" {
		tableName = "payroll_run"
	}
	return &SQLServerPayrollRunRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePayrollRun creates a new payroll run record.
func (r *SQLServerPayrollRunRepository) CreatePayrollRun(ctx context.Context, req *payrollrunpb.CreatePayrollRunRequest) (*payrollrunpb.CreatePayrollRunResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payroll run data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "payPeriodStart", "pay_period_start")
	convertMillisToTime(data, "payPeriodEnd", "pay_period_end")
	convertMillisToTime(data, "postedAt", "posted_at")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payroll run: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	payrollRun := &payrollrunpb.PayrollRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollrunpb.CreatePayrollRunResponse{Success: true, Data: []*payrollrunpb.PayrollRun{payrollRun}}, nil
}

// ReadPayrollRun retrieves a payroll run record by ID.
func (r *SQLServerPayrollRunRepository) ReadPayrollRun(ctx context.Context, req *payrollrunpb.ReadPayrollRunRequest) (*payrollrunpb.ReadPayrollRunResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll run ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll run: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	payrollRun := &payrollrunpb.PayrollRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollrunpb.ReadPayrollRunResponse{Success: true, Data: []*payrollrunpb.PayrollRun{payrollRun}}, nil
}

// UpdatePayrollRun updates a payroll run record.
func (r *SQLServerPayrollRunRepository) UpdatePayrollRun(ctx context.Context, req *payrollrunpb.UpdatePayrollRunRequest) (*payrollrunpb.UpdatePayrollRunResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll run ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "payPeriodStart", "pay_period_start")
	convertMillisToTime(data, "payPeriodEnd", "pay_period_end")
	convertMillisToTime(data, "postedAt", "posted_at")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payroll run: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	payrollRun := &payrollrunpb.PayrollRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollrunpb.UpdatePayrollRunResponse{Success: true, Data: []*payrollrunpb.PayrollRun{payrollRun}}, nil
}

// DeletePayrollRun deletes a payroll run record (soft delete).
func (r *SQLServerPayrollRunRepository) DeletePayrollRun(ctx context.Context, req *payrollrunpb.DeletePayrollRunRequest) (*payrollrunpb.DeletePayrollRunResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll run ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete payroll run: %w", err)
	}
	return &payrollrunpb.DeletePayrollRunResponse{Success: true}, nil
}

// ListPayrollRuns lists payroll run records with optional filters.
func (r *SQLServerPayrollRunRepository) ListPayrollRuns(ctx context.Context, req *payrollrunpb.ListPayrollRunsRequest) (*payrollrunpb.ListPayrollRunsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll runs: %w", err)
	}
	var payrollRuns []*payrollrunpb.PayrollRun
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal payroll run row: %v", err)
			continue
		}
		payrollRun := &payrollrunpb.PayrollRun{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRun); err != nil {
			log.Printf("WARN: protojson unmarshal payroll run: %v", err)
			continue
		}
		payrollRuns = append(payrollRuns, payrollRun)
	}
	return &payrollrunpb.ListPayrollRunsResponse{Success: true, Data: payrollRuns}, nil
}

// payrollRunSortableSQLCols is the A2 sort whitelist for payroll_run list pages.
var payrollRunSortableSQLCols = []string{
	"pr.id", "pr.date_created", "pr.date_modified",
	"pr.run_number", "pr.pay_period_start", "pr.pay_period_end",
	"pr.total_gross", "pr.total_deductions", "pr.total_net",
	"pr.employee_count", "pr.status", "pr.posted_at",
}

// GetPayrollRunListPageData retrieves payroll runs with pagination, filtering, sorting, and search.
//
// SQL Server differences from the postgres gold standard:
//   - $1/$2/$3/$4 → @p1/@p2/@p3/@p4.
//   - ILIKE → LIKE (SQL Server default CI collation).
//   - $1::text IS NULL → workspace_id IS NULL OR workspace_id = @p1 pattern replaced
//     with a cleaner @p1 = ” OR pr.workspace_id = @p1 check.
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//     BuildOrderBy returns a bracketed ORDER BY expression required before OFFSET/FETCH.
//   - COUNT(*) OVER () retained — supported in SQL Server 2017+.
func (r *SQLServerPayrollRunRepository) GetPayrollRunListPageData(
	ctx context.Context,
	req *payrollrunpb.GetPayrollRunListPageDataRequest,
) (*payrollrunpb.GetPayrollRunListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payroll run list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// A2: Sort guard — BuildOrderBy uses [bracket] quoting for SQL Server.
	orderByClause, err := sqlserverCore.BuildOrderBy(payrollRunSortableSQLCols, req.GetSort(), "pr.date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build WHERE clauses.
	whereParts := []string{"(@p1 = '' OR pr.workspace_id = @p1)"}
	args := []any{workspaceID}
	nextIdx := 2

	if searchPattern != "" {
		whereParts = append(whereParts, fmt.Sprintf("pr.run_number LIKE @p%d", nextIdx))
		args = append(args, searchPattern)
		nextIdx++
	}

	whereSQL := "WHERE " + strings.Join(whereParts, " AND ")

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	args = append(args, offset, limit)

	// A3: COUNT(*) OVER() — one pass, no second counted CTE needed.
	query := fmt.Sprintf(`
		SELECT
			pr.id,
			pr.date_created,
			pr.date_modified,
			pr.run_number,
			pr.pay_period_start,
			pr.pay_period_end,
			pr.total_gross,
			pr.total_deductions,
			pr.total_net,
			pr.employee_count,
			pr.status,
			pr.approved_by,
			pr.posted_at,
			pr.posted_at_string,
			COUNT(*) OVER() AS total
		FROM %s pr
		%s
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, r.tableName, whereSQL, orderByClause, offsetIdx, limitIdx)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query payroll run list page data: %w", err)
	}
	defer rows.Close()

	var payrollRuns []*payrollrunpb.PayrollRun
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     int64
			dateModified    int64
			runNumber       string
			payPeriodStart  string
			payPeriodEnd    string
			totalGross      int64
			totalDeductions int64
			totalNet        int64
			employeeCount   int32
			statusStr       string
			approvedBy      *string
			postedAt        *int64
			postedAtString  *string
			total           int64
		)

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &runNumber, &payPeriodStart, &payPeriodEnd,
			&totalGross, &totalDeductions, &totalNet, &employeeCount, &statusStr,
			&approvedBy, &postedAt, &postedAtString, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan payroll run row: %w", err)
		}

		totalCount = total

		payrollRun := &payrollrunpb.PayrollRun{
			Id:              id,
			RunNumber:       runNumber,
			TotalGross:      totalGross,
			TotalDeductions: totalDeductions,
			TotalNet:        totalNet,
			EmployeeCount:   employeeCount,
			ApprovedBy:      approvedBy,
			PostedAtString:  postedAtString,
			PayPeriodStart:  payPeriodStart,
			PayPeriodEnd:    payPeriodEnd,
		}
		if val, ok := payrollrunpb.PayrollRunStatus_value[statusStr]; ok {
			payrollRun.Status = payrollrunpb.PayrollRunStatus(val)
		}
		if postedAt != nil && *postedAt > 0 {
			payrollRun.PostedAt = postedAt
		}
		if dateCreated > 0 {
			payrollRun.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			payrollRun.DateModified = &dateModified
		}

		payrollRuns = append(payrollRuns, payrollRun)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll run rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &payrollrunpb.GetPayrollRunListPageDataResponse{
		PayrollRunList: payrollRuns,
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

// GetPayrollRunItemPageData retrieves a single payroll run.
func (r *SQLServerPayrollRunRepository) GetPayrollRunItemPageData(
	ctx context.Context,
	req *payrollrunpb.GetPayrollRunItemPageDataRequest,
) (*payrollrunpb.GetPayrollRunItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payroll run item page data request is required")
	}
	if req.PayrollRunId == "" {
		return nil, fmt.Errorf("payroll run ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PayrollRunId)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll run '%s': %w", req.PayrollRunId, err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	payrollRun := &payrollrunpb.PayrollRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollrunpb.GetPayrollRunItemPageDataResponse{
		PayrollRun: payrollRun,
		Success:    true,
	}, nil
}

// NewPayrollRunRepository creates a new SQL Server payroll run repository (old-style constructor).
func NewPayrollRunRepository(db *sql.DB, tableName string) payrollrunpb.PayrollRunDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerPayrollRunRepository(db, dbOps, tableName)
}
