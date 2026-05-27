//go:build sqlserver

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PayrollRemittance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver payroll_remittance repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPayrollRemittanceRepository(db, dbOps, tableName), nil
	})
}

// SQLServerPayrollRemittanceRepository implements payroll remittance CRUD using SQL Server.
type SQLServerPayrollRemittanceRepository struct {
	payrollremittancepb.UnimplementedPayrollRemittanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerPayrollRemittanceRepository creates a new SQL Server payroll remittance repository.
func NewSQLServerPayrollRemittanceRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) payrollremittancepb.PayrollRemittanceDomainServiceServer {
	if tableName == "" {
		tableName = "payroll_remittance"
	}
	return &SQLServerPayrollRemittanceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePayrollRemittance creates a new payroll remittance record.
func (r *SQLServerPayrollRemittanceRepository) CreatePayrollRemittance(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) (*payrollremittancepb.CreatePayrollRemittanceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payroll remittance data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "filedAt", "filed_at")
	convertMillisToTime(data, "paidAt", "paid_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payroll_remittance: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	prm := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollremittancepb.CreatePayrollRemittanceResponse{Success: true, Data: []*payrollremittancepb.PayrollRemittance{prm}}, nil
}

// ReadPayrollRemittance retrieves a payroll remittance by ID.
func (r *SQLServerPayrollRemittanceRepository) ReadPayrollRemittance(ctx context.Context, req *payrollremittancepb.ReadPayrollRemittanceRequest) (*payrollremittancepb.ReadPayrollRemittanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll_remittance: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	prm := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollremittancepb.ReadPayrollRemittanceResponse{Success: true, Data: []*payrollremittancepb.PayrollRemittance{prm}}, nil
}

// UpdatePayrollRemittance updates a payroll remittance record.
func (r *SQLServerPayrollRemittanceRepository) UpdatePayrollRemittance(ctx context.Context, req *payrollremittancepb.UpdatePayrollRemittanceRequest) (*payrollremittancepb.UpdatePayrollRemittanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "filedAt", "filed_at")
	convertMillisToTime(data, "paidAt", "paid_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payroll_remittance: %w", err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	prm := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollremittancepb.UpdatePayrollRemittanceResponse{Success: true, Data: []*payrollremittancepb.PayrollRemittance{prm}}, nil
}

// DeletePayrollRemittance soft-deletes a payroll remittance record.
func (r *SQLServerPayrollRemittanceRepository) DeletePayrollRemittance(ctx context.Context, req *payrollremittancepb.DeletePayrollRemittanceRequest) (*payrollremittancepb.DeletePayrollRemittanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete payroll_remittance: %w", err)
	}
	return &payrollremittancepb.DeletePayrollRemittanceResponse{Success: true}, nil
}

// ListPayrollRemittances lists payroll remittance records with optional filters.
func (r *SQLServerPayrollRemittanceRepository) ListPayrollRemittances(ctx context.Context, req *payrollremittancepb.ListPayrollRemittancesRequest) (*payrollremittancepb.ListPayrollRemittancesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll_remittances: %w", err)
	}
	var items []*payrollremittancepb.PayrollRemittance
	for _, result := range listResult.Data {
		sqlserverCore.ConvertMillisToDateStr(result, "due_date")
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal payroll_remittance row: %v", err)
			continue
		}
		prm := &payrollremittancepb.PayrollRemittance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prm); err != nil {
			log.Printf("WARN: protojson unmarshal payroll_remittance: %v", err)
			continue
		}
		items = append(items, prm)
	}
	return &payrollremittancepb.ListPayrollRemittancesResponse{Success: true, Data: items}, nil
}

// payrollRemittanceSortableSQLCols is the A2 sort whitelist for payroll_remittance list pages.
var payrollRemittanceSortableSQLCols = []string{
	"rem.id", "rem.date_created",
	"rem.payroll_run_id", "rem.remittance_type",
	"rem.amount", "rem.due_date", "rem.status",
	"rem.filed_at", "rem.paid_at", "rem.reference_number",
}

// GetPayrollRemittanceListPageData retrieves payroll remittances with pagination, filtering, and sorting.
// Joins with payroll_run to enforce workspace-scoped A1 tenant guard.
func (r *SQLServerPayrollRemittanceRepository) GetPayrollRemittanceListPageData(
	ctx context.Context,
	req *payrollremittancepb.GetPayrollRemittanceListPageDataRequest,
) (*payrollremittancepb.GetPayrollRemittanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payroll remittance list page data request is required")
	}

	// A1: workspace predicate enforced via payroll_run join — empty workspaceID returns zero rows.
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	// A2: sort whitelist enforced by BuildOrderBy — unrecognised column returns error.
	orderByClause, err := sqlserverCore.BuildOrderBy(payrollRemittanceSortableSQLCols, req.GetSort(), "rem.date_created DESC")
	if err != nil {
		return nil, err
	}

	// A3: COUNT(*) OVER() replaces a separate count query — one pass.
	// Dialect: @p1..@p4 placeholders; LIKE instead of ILIKE (SQL Server is CI by default).
	// Pagination: ORDER BY required before OFFSET/FETCH in SQL Server.
	query := fmt.Sprintf(`
		SELECT
			rem.id,
			rem.date_created,
			rem.payroll_run_id,
			rem.remittance_type,
			rem.amount,
			rem.due_date,
			rem.due_date_string,
			rem.status,
			rem.filed_at,
			rem.filed_at_string,
			rem.paid_at,
			rem.paid_at_string,
			rem.reference_number,
			COUNT(*) OVER() AS total
		FROM [payroll_remittance] rem
		LEFT JOIN [payroll_run] pr ON pr.id = rem.payroll_run_id
		WHERE pr.workspace_id = @p1
		  AND (@p2 = '' OR rem.reference_number LIKE @p2)
		%s
		ORDER BY rem.date_created DESC
		OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY;
	`, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query payroll_remittance list page data: %w", err)
	}
	defer rows.Close()

	var items []*payrollremittancepb.PayrollRemittance
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     int64
			payrollRunID    string
			rTypeStr        string
			amount          int64
			dueDate         int64
			dueDateString   *string
			statusStr       string
			filedAt         *int64
			filedAtString   *string
			paidAt          *int64
			paidAtString    *string
			referenceNumber *string
			total           int64
		)
		if err := rows.Scan(
			&id,
			&dateCreated,
			&payrollRunID,
			&rTypeStr,
			&amount,
			&dueDate,
			&dueDateString,
			&statusStr,
			&filedAt,
			&filedAtString,
			&paidAt,
			&paidAtString,
			&referenceNumber,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan payroll_remittance row: %w", err)
		}
		totalCount = total

		prm := &payrollremittancepb.PayrollRemittance{
			Id:              id,
			PayrollRunId:    payrollRunID,
			Amount:          amount,
			FiledAtString:   filedAtString,
			PaidAtString:    paidAtString,
			ReferenceNumber: referenceNumber,
		}
		if val, ok := payrollremittancepb.RemittanceType_value[rTypeStr]; ok {
			prm.RemittanceType = payrollremittancepb.RemittanceType(val)
		}
		if val, ok := payrollremittancepb.RemittanceStatus_value[statusStr]; ok {
			prm.Status = payrollremittancepb.RemittanceStatus(val)
		}
		if dueDate > 0 {
			prm.DueDate = time.UnixMilli(dueDate).UTC().Format("2006-01-02")
		}
		if filedAt != nil && *filedAt > 0 {
			prm.FiledAt = filedAt
		}
		if paidAt != nil && *paidAt > 0 {
			prm.PaidAt = paidAt
		}
		if dateCreated > 0 {
			prm.DateCreated = &dateCreated
		}
		items = append(items, prm)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll_remittance rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &payrollremittancepb.GetPayrollRemittanceListPageDataResponse{
		PayrollRemittanceList: items,
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

// GetPayrollRemittanceItemPageData retrieves a single payroll remittance.
func (r *SQLServerPayrollRemittanceRepository) GetPayrollRemittanceItemPageData(
	ctx context.Context,
	req *payrollremittancepb.GetPayrollRemittanceItemPageDataRequest,
) (*payrollremittancepb.GetPayrollRemittanceItemPageDataResponse, error) {
	if req == nil || req.PayrollRemittanceId == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PayrollRemittanceId)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll_remittance '%s': %w", req.PayrollRemittanceId, err)
	}
	sqlserverCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	prm := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &payrollremittancepb.GetPayrollRemittanceItemPageDataResponse{
		PayrollRemittance: prm,
		Success:           true,
	}, nil
}

// NewPayrollRemittanceRepository creates a SQL Server payroll remittance repository (old-style constructor).
func NewPayrollRemittanceRepository(db *sql.DB, tableName string) payrollremittancepb.PayrollRemittanceDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerPayrollRemittanceRepository(db, dbOps, tableName)
}
