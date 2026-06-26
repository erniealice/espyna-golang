//go:build sqlserver

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.DisbursementSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver disbursement_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerDisbursementScheduleRepository(dbOps, tableName), nil
	})
}

// SQLServerDisbursementScheduleRepository implements disbursement schedule CRUD using SQL Server.
type SQLServerDisbursementScheduleRepository struct {
	disbursementschedulepb.UnimplementedDisbursementScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerDisbursementScheduleRepository creates a new SQL Server disbursement schedule repository.
func NewSQLServerDisbursementScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) disbursementschedulepb.DisbursementScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "disbursement_schedule"
	}
	var db *sql.DB
	if ep, ok := dbOps.(executorProvider); ok {
		if rawDB, ok2 := ep.GetExecutor(context.Background()).(*sql.DB); ok2 {
			db = rawDB
		}
	}
	return &SQLServerDisbursementScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDisbursementSchedule creates a new disbursement schedule record.
func (r *SQLServerDisbursementScheduleRepository) CreateDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.CreateDisbursementScheduleRequest) (*disbursementschedulepb.CreateDisbursementScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("disbursement schedule data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement schedule: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &disbursementschedulepb.CreateDisbursementScheduleResponse{Success: true, Data: []*disbursementschedulepb.DisbursementSchedule{ds}}, nil
}

// ReadDisbursementSchedule retrieves a disbursement schedule record by ID.
func (r *SQLServerDisbursementScheduleRepository) ReadDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.ReadDisbursementScheduleRequest) (*disbursementschedulepb.ReadDisbursementScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement schedule: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &disbursementschedulepb.ReadDisbursementScheduleResponse{Success: true, Data: []*disbursementschedulepb.DisbursementSchedule{ds}}, nil
}

// UpdateDisbursementSchedule updates a disbursement schedule record.
func (r *SQLServerDisbursementScheduleRepository) UpdateDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.UpdateDisbursementScheduleRequest) (*disbursementschedulepb.UpdateDisbursementScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update disbursement schedule: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &disbursementschedulepb.UpdateDisbursementScheduleResponse{Success: true, Data: []*disbursementschedulepb.DisbursementSchedule{ds}}, nil
}

// DeleteDisbursementSchedule soft-deletes a disbursement schedule record.
func (r *SQLServerDisbursementScheduleRepository) DeleteDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.DeleteDisbursementScheduleRequest) (*disbursementschedulepb.DeleteDisbursementScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete disbursement schedule: %w", err)
	}
	return &disbursementschedulepb.DeleteDisbursementScheduleResponse{Success: true}, nil
}

// ListDisbursementSchedules lists disbursement schedule records with optional filters.
func (r *SQLServerDisbursementScheduleRepository) ListDisbursementSchedules(ctx context.Context, req *disbursementschedulepb.ListDisbursementSchedulesRequest) (*disbursementschedulepb.ListDisbursementSchedulesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list disbursement schedules: %w", err)
	}
	var schedules []*disbursementschedulepb.DisbursementSchedule
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		ds := &disbursementschedulepb.DisbursementSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
			continue
		}
		schedules = append(schedules, ds)
	}
	return &disbursementschedulepb.ListDisbursementSchedulesResponse{Success: true, Data: schedules}, nil
}

// GetDisbursementScheduleListPageData retrieves disbursement schedules with pagination, filtering, and search.
//
// SQL Server: active = 1; @pN placeholders; LIKE; OFFSET/FETCH pagination;
// COUNT(*) OVER() for total; no workspace_id filter (mirrors postgres behaviour).
func (r *SQLServerDisbursementScheduleRepository) GetDisbursementScheduleListPageData(
	ctx context.Context,
	req *disbursementschedulepb.GetDisbursementScheduleListPageDataRequest,
) (*disbursementschedulepb.GetDisbursementScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement schedule list page data request is required")
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

	sortDir := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortDir = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				ds.id,
				ds.date_created,
				ds.date_modified,
				ds.active,
				ds.expenditure_id,
				ds.sequence,
				ds.amount,
				ds.due_date,
				ds.due_date_string,
				ds.status,
				ds.paid_amount,
				ds.paid_date,
				ds.disbursement_id,
				ds.payment_term_id,
				COUNT(*) OVER() AS total
			FROM ` + r.tableName + ` ds
			WHERE ds.active = 1
			  AND (@p1 = '' OR ds.status LIKE @p1)
		)
		SELECT * FROM enriched
		ORDER BY date_created ` + sortDir + `
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement schedule list page data: %w", err)
	}
	defer rows.Close()

	var schedules []*disbursementschedulepb.DisbursementSchedule
	var totalCount int64

	for rows.Next() {
		var (
			id             string
			dateCreated    time.Time
			dateModified   time.Time
			active         bool
			expenditureID  string
			sequence       int32
			amount         int64
			dueDate        int64
			dueDateString  *string
			status         string
			paidAmount     *int64
			paidDate       *int64
			disbursementID *string
			paymentTermID  *string
			total          int64
		)
		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active,
			&expenditureID, &sequence, &amount, &dueDate, &dueDateString, &status,
			&paidAmount, &paidDate, &disbursementID, &paymentTermID,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan disbursement schedule row: %w", err)
		}
		totalCount = total
		schedules = append(schedules, buildDisbursementScheduleFromScan(
			id, dateCreated, dateModified, active,
			expenditureID, sequence, amount, dueDate, dueDateString, status,
			paidAmount, paidDate, disbursementID, paymentTermID,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating disbursement schedule rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &disbursementschedulepb.GetDisbursementScheduleListPageDataResponse{
		DisbursementScheduleList: schedules,
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

// GetDisbursementScheduleItemPageData retrieves a single disbursement schedule by ID.
//
// SQL Server: TOP 1 instead of LIMIT 1; @pN placeholder; no workspace filter.
func (r *SQLServerDisbursementScheduleRepository) GetDisbursementScheduleItemPageData(
	ctx context.Context,
	req *disbursementschedulepb.GetDisbursementScheduleItemPageDataRequest,
) (*disbursementschedulepb.GetDisbursementScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement schedule item page data request is required")
	}
	if req.DisbursementScheduleId == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				ds.id,
				ds.date_created,
				ds.date_modified,
				ds.active,
				ds.expenditure_id,
				ds.sequence,
				ds.amount,
				ds.due_date,
				ds.due_date_string,
				ds.status,
				ds.paid_amount,
				ds.paid_date,
				ds.disbursement_id,
				ds.payment_term_id
			FROM ` + r.tableName + ` ds
			WHERE ds.id = @p1
		)
		SELECT TOP 1 * FROM enriched`

	row := r.db.QueryRowContext(ctx, query, req.DisbursementScheduleId)

	var (
		id             string
		dateCreated    time.Time
		dateModified   time.Time
		active         bool
		expenditureID  string
		sequence       int32
		amount         int64
		dueDate        int64
		dueDateString  *string
		status         string
		paidAmount     *int64
		paidDate       *int64
		disbursementID *string
		paymentTermID  *string
	)

	err := row.Scan(
		&id, &dateCreated, &dateModified, &active,
		&expenditureID, &sequence, &amount, &dueDate, &dueDateString, &status,
		&paidAmount, &paidDate, &disbursementID, &paymentTermID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("disbursement schedule with ID '%s' not found", req.DisbursementScheduleId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement schedule item page data: %w", err)
	}

	ds := buildDisbursementScheduleFromScan(
		id, dateCreated, dateModified, active,
		expenditureID, sequence, amount, dueDate, dueDateString, status,
		paidAmount, paidDate, disbursementID, paymentTermID,
	)

	return &disbursementschedulepb.GetDisbursementScheduleItemPageDataResponse{
		DisbursementSchedule: ds,
		Success:              true,
	}, nil
}

// buildDisbursementScheduleFromScan constructs a DisbursementSchedule proto from scanned fields.
func buildDisbursementScheduleFromScan(
	id string, dateCreated time.Time, dateModified time.Time, active bool,
	expenditureID string, sequence int32, amount int64, dueDate int64, dueDateString *string, status string,
	paidAmount *int64, paidDate *int64, disbursementID *string, paymentTermID *string,
) *disbursementschedulepb.DisbursementSchedule {
	ds := &disbursementschedulepb.DisbursementSchedule{
		Id:             id,
		Active:         active,
		ExpenditureId:  expenditureID,
		Sequence:       sequence,
		Amount:         amount,
		DueDate:        time.UnixMilli(dueDate).UTC().Format("2006-01-02"),
		Status:         status,
		PaidAmount:     paidAmount,
		PaidDate:       paidDate,
		DisbursementId: disbursementID,
		PaymentTermId:  paymentTermID,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		ds.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		ds.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		ds.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		ds.DateModifiedString = &dmStr
	}
	return ds
}

// NewDisbursementScheduleRepository creates a new SQL Server disbursement schedule repository (old-style constructor).
func NewDisbursementScheduleRepository(db *sql.DB, tableName string) disbursementschedulepb.DisbursementScheduleDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerDisbursementScheduleRepository(dbOps, tableName)
}
