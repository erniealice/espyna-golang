//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PayrollRemittance, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres payroll remittance repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPayrollRemittanceRepository(dbOps, tableName), nil
	})
}

// PostgresPayrollRemittanceRepository implements payroll remittance CRUD operations using PostgreSQL
type PostgresPayrollRemittanceRepository struct {
	payrollremittancepb.UnimplementedPayrollRemittanceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPayrollRemittanceRepository creates a new PostgreSQL payroll remittance repository
func NewPostgresPayrollRemittanceRepository(dbOps interfaces.DatabaseOperation, tableName string) payrollremittancepb.PayrollRemittanceDomainServiceServer {
	if tableName == "" {
		tableName = "payroll_remittance"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPayrollRemittanceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePayrollRemittance creates a new payroll remittance record
func (r *PostgresPayrollRemittanceRepository) CreatePayrollRemittance(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) (*payrollremittancepb.CreatePayrollRemittanceResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "filedAt", "filed_at")
	convertMillisToTime(data, "paidAt", "paid_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payroll remittance: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payrollRemittance := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRemittance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &payrollremittancepb.CreatePayrollRemittanceResponse{
		Success: true,
		Data:    []*payrollremittancepb.PayrollRemittance{payrollRemittance},
	}, nil
}

// ReadPayrollRemittance retrieves a payroll remittance record by ID
func (r *PostgresPayrollRemittanceRepository) ReadPayrollRemittance(ctx context.Context, req *payrollremittancepb.ReadPayrollRemittanceRequest) (*payrollremittancepb.ReadPayrollRemittanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll remittance: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payrollRemittance := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRemittance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &payrollremittancepb.ReadPayrollRemittanceResponse{
		Success: true,
		Data:    []*payrollremittancepb.PayrollRemittance{payrollRemittance},
	}, nil
}

// UpdatePayrollRemittance updates a payroll remittance record
func (r *PostgresPayrollRemittanceRepository) UpdatePayrollRemittance(ctx context.Context, req *payrollremittancepb.UpdatePayrollRemittanceRequest) (*payrollremittancepb.UpdatePayrollRemittanceResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dueDate", "due_date")
	convertMillisToTime(data, "filedAt", "filed_at")
	convertMillisToTime(data, "paidAt", "paid_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payroll remittance: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payrollRemittance := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRemittance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &payrollremittancepb.UpdatePayrollRemittanceResponse{
		Success: true,
		Data:    []*payrollremittancepb.PayrollRemittance{payrollRemittance},
	}, nil
}

// DeletePayrollRemittance deletes a payroll remittance record (soft delete)
func (r *PostgresPayrollRemittanceRepository) DeletePayrollRemittance(ctx context.Context, req *payrollremittancepb.DeletePayrollRemittanceRequest) (*payrollremittancepb.DeletePayrollRemittanceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payroll remittance: %w", err)
	}

	return &payrollremittancepb.DeletePayrollRemittanceResponse{
		Success: true,
	}, nil
}

// ListPayrollRemittances lists payroll remittance records with optional filters
func (r *PostgresPayrollRemittanceRepository) ListPayrollRemittances(ctx context.Context, req *payrollremittancepb.ListPayrollRemittancesRequest) (*payrollremittancepb.ListPayrollRemittancesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll remittances: %w", err)
	}

	var payrollRemittances []*payrollremittancepb.PayrollRemittance
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "due_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal payroll remittance row: %v", err)
			continue
		}

		payrollRemittance := &payrollremittancepb.PayrollRemittance{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRemittance); err != nil {
			log.Printf("WARN: protojson unmarshal payroll remittance: %v", err)
			continue
		}
		payrollRemittances = append(payrollRemittances, payrollRemittance)
	}

	return &payrollremittancepb.ListPayrollRemittancesResponse{
		Success: true,
		Data:    payrollRemittances,
	}, nil
}

// GetPayrollRemittanceListPageData retrieves payroll remittances with pagination, filtering, sorting, and search using CTE
// Joins with payroll_run for period context
func (r *PostgresPayrollRemittanceRepository) GetPayrollRemittanceListPageData(
	ctx context.Context,
	req *payrollremittancepb.GetPayrollRemittanceListPageDataRequest,
) (*payrollremittancepb.GetPayrollRemittanceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payroll remittance list page data request is required")
	}

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

	sortField := "rem.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
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
				rem.reference_number
			FROM payroll_remittance rem
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       rem.reference_number ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query payroll remittance list page data: %w", err)
	}
	defer rows.Close()

	var payrollRemittances []*payrollremittancepb.PayrollRemittance
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       int64
			payrollRunID      string
			remittanceTypeStr string
			amount            int64
			dueDate           int64
			dueDateString     *string
			statusStr         string
			filedAt           *int64
			filedAtString     *string
			paidAt            *int64
			paidAtString      *string
			referenceNumber   *string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&payrollRunID,
			&remittanceTypeStr,
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payroll remittance row: %w", err)
		}

		totalCount = total

		payrollRemittance := &payrollremittancepb.PayrollRemittance{
			Id:              id,
			PayrollRunId:    payrollRunID,
			Amount:          amount,
			FiledAtString:   filedAtString,
			PaidAtString:    paidAtString,
			ReferenceNumber: referenceNumber,
		}

		if val, ok := payrollremittancepb.RemittanceType_value[remittanceTypeStr]; ok {
			payrollRemittance.RemittanceType = payrollremittancepb.RemittanceType(val)
		}
		if val, ok := payrollremittancepb.RemittanceStatus_value[statusStr]; ok {
			payrollRemittance.Status = payrollremittancepb.RemittanceStatus(val)
		}

		if dueDate > 0 {
			payrollRemittance.DueDate = time.UnixMilli(dueDate).UTC().Format("2006-01-02")
		}
		if filedAt != nil && *filedAt > 0 {
			payrollRemittance.FiledAt = filedAt
		}
		if paidAt != nil && *paidAt > 0 {
			payrollRemittance.PaidAt = paidAt
		}
		if dateCreated > 0 {
			payrollRemittance.DateCreated = &dateCreated
		}

		payrollRemittances = append(payrollRemittances, payrollRemittance)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payroll remittance rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &payrollremittancepb.GetPayrollRemittanceListPageDataResponse{
		PayrollRemittanceList: payrollRemittances,
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

// GetPayrollRemittanceItemPageData retrieves a single payroll remittance with enriched data
// TODO: Implement full CTE query with joined payroll_run details
func (r *PostgresPayrollRemittanceRepository) GetPayrollRemittanceItemPageData(
	ctx context.Context,
	req *payrollremittancepb.GetPayrollRemittanceItemPageDataRequest,
) (*payrollremittancepb.GetPayrollRemittanceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payroll remittance item page data request is required")
	}
	if req.PayrollRemittanceId == "" {
		return nil, fmt.Errorf("payroll remittance ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.PayrollRemittanceId)
	if err != nil {
		return nil, fmt.Errorf("failed to read payroll remittance '%s': %w", req.PayrollRemittanceId, err)
	}

	postgresCore.ConvertMillisToDateStr(result, "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payrollRemittance := &payrollremittancepb.PayrollRemittance{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, payrollRemittance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &payrollremittancepb.GetPayrollRemittanceItemPageDataResponse{
		PayrollRemittance: payrollRemittance,
		Success:           true,
	}, nil
}

// NewPayrollRemittanceRepository creates a new PostgreSQL payroll remittance repository (old-style constructor)
func NewPayrollRemittanceRepository(db *sql.DB, tableName string) payrollremittancepb.PayrollRemittanceDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPayrollRemittanceRepository(dbOps, tableName)
}
