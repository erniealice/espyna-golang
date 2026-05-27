//go:build mysql

// Package treasury — MySQL 8.0+ adapter for the disbursement_schedule entity.
//
// Dialect translation from postgres gold standard
// (docs/plan/20260527-multi-dialect-adapter-alignment/brief.md):
//   - $N → ? (positional, re-sequenced)
//   - "ident" → `ident` (backtick quoting)
//   - ILIKE → LIKE (MySQL ci collation)
//   - FILTER (WHERE c) → SUM(CASE WHEN c THEN expr END)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions)
//   - RETURNING → app-side UUID + SELECT after insert
//   - active = true → active = 1 (MySQL TINYINT(1) boolean)
package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.DisbursementSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql disbursement_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLDisbursementScheduleRepository(dbOps, tableName), nil
	})
}

// MySQLDisbursementScheduleRepository implements disbursement schedule CRUD operations using MySQL 8.0+.
type MySQLDisbursementScheduleRepository struct {
	disbursementschedulepb.UnimplementedDisbursementScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLDisbursementScheduleRepository creates a new MySQL disbursement schedule repository.
func NewMySQLDisbursementScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) disbursementschedulepb.DisbursementScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "disbursement_schedule"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLDisbursementScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDisbursementSchedule creates a new disbursement schedule record.
func (r *MySQLDisbursementScheduleRepository) CreateDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.CreateDisbursementScheduleRequest) (*disbursementschedulepb.CreateDisbursementScheduleResponse, error) {
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

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementschedulepb.CreateDisbursementScheduleResponse{
		Success: true,
		Data:    []*disbursementschedulepb.DisbursementSchedule{ds},
	}, nil
}

// ReadDisbursementSchedule retrieves a disbursement schedule record by ID.
func (r *MySQLDisbursementScheduleRepository) ReadDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.ReadDisbursementScheduleRequest) (*disbursementschedulepb.ReadDisbursementScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement schedule: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementschedulepb.ReadDisbursementScheduleResponse{
		Success: true,
		Data:    []*disbursementschedulepb.DisbursementSchedule{ds},
	}, nil
}

// UpdateDisbursementSchedule updates a disbursement schedule record.
func (r *MySQLDisbursementScheduleRepository) UpdateDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.UpdateDisbursementScheduleRequest) (*disbursementschedulepb.UpdateDisbursementScheduleResponse, error) {
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

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ds := &disbursementschedulepb.DisbursementSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementschedulepb.UpdateDisbursementScheduleResponse{
		Success: true,
		Data:    []*disbursementschedulepb.DisbursementSchedule{ds},
	}, nil
}

// DeleteDisbursementSchedule deletes a disbursement schedule record (soft delete).
func (r *MySQLDisbursementScheduleRepository) DeleteDisbursementSchedule(ctx context.Context, req *disbursementschedulepb.DeleteDisbursementScheduleRequest) (*disbursementschedulepb.DeleteDisbursementScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete disbursement schedule: %w", err)
	}

	return &disbursementschedulepb.DeleteDisbursementScheduleResponse{
		Success: true,
	}, nil
}

// ListDisbursementSchedules lists disbursement schedule records with optional filters.
func (r *MySQLDisbursementScheduleRepository) ListDisbursementSchedules(ctx context.Context, req *disbursementschedulepb.ListDisbursementSchedulesRequest) (*disbursementschedulepb.ListDisbursementSchedulesResponse, error) {
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
		mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		ds := &disbursementschedulepb.DisbursementSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ds); err != nil {
			continue
		}
		schedules = append(schedules, ds)
	}

	return &disbursementschedulepb.ListDisbursementSchedulesResponse{
		Success: true,
		Data:    schedules,
	}, nil
}

// GetDisbursementScheduleListPageData retrieves disbursement schedules with pagination,
// filtering, sorting, and search.
//
// Dialect translation from postgres gold standard:
//   - $1/$2/$3 → ? (positional, args: searchPattern, limit, offset)
//   - active = true → active = 1 (MySQL TINYINT(1))
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions supported)
//   - Sort field interpolated after whitelist validation — no $N needed
func (r *MySQLDisbursementScheduleRepository) GetDisbursementScheduleListPageData(
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

	// Sort — validated against whitelist (A2 guard). mysqlCore.BuildOrderBy
	// uses backtick quoting for MySQL column names.
	disbursementScheduleSortableSQLCols := []string{
		"ds.date_created", "ds.date_modified", "ds.amount",
		"ds.due_date", "ds.status", "ds.sequence",
	}
	sortClause, err := mysqlCore.BuildOrderBy(disbursementScheduleSortableSQLCols, req.GetSort(), "ds.date_created DESC")
	if err != nil {
		return nil, err
	}

	// Dialect change: active = true → active = 1; ILIKE → LIKE; $1/$2/$3 → ?
	// searchPattern is "" when no search — WHERE clause handles empty string as no-op.
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
			WHERE ds.active = 1
			  AND (? = '' OR ds.status LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + sortClause + `
		LIMIT ? OFFSET ?;
	`

	// Args: searchPattern (twice for the two ? in the LIKE clause), limit, offset.
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, searchPattern, limit, offset)
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
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&expenditureID,
			&sequence,
			&amount,
			&dueDate,
			&dueDateString,
			&status,
			&paidAmount,
			&paidDate,
			&disbursementID,
			&paymentTermID,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan disbursement schedule row: %w", err)
		}

		totalCount = total

		ds := buildMySQLDisbursementScheduleFromScan(
			id, dateCreated, dateModified, active,
			expenditureID, sequence, amount, dueDate, dueDateString, status,
			paidAmount, paidDate, disbursementID, paymentTermID,
		)

		schedules = append(schedules, ds)
	}

	if err = rows.Err(); err != nil {
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
// Dialect translation: $1 → ?; active = true → active = 1 not needed here (lookup by ID).
func (r *MySQLDisbursementScheduleRepository) GetDisbursementScheduleItemPageData(
	ctx context.Context,
	req *disbursementschedulepb.GetDisbursementScheduleItemPageDataRequest,
) (*disbursementschedulepb.GetDisbursementScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement schedule item page data request is required")
	}
	if req.DisbursementScheduleId == "" {
		return nil, fmt.Errorf("disbursement schedule ID is required")
	}

	// Dialect change: $1 → ?
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
			WHERE ds.id = ?
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.DisbursementScheduleId)

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
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&expenditureID,
		&sequence,
		&amount,
		&dueDate,
		&dueDateString,
		&status,
		&paidAmount,
		&paidDate,
		&disbursementID,
		&paymentTermID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("disbursement schedule with ID '%s' not found", req.DisbursementScheduleId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement schedule item page data: %w", err)
	}

	ds := buildMySQLDisbursementScheduleFromScan(
		id, dateCreated, dateModified, active,
		expenditureID, sequence, amount, dueDate, dueDateString, status,
		paidAmount, paidDate, disbursementID, paymentTermID,
	)

	return &disbursementschedulepb.GetDisbursementScheduleItemPageDataResponse{
		DisbursementSchedule: ds,
		Success:              true,
	}, nil
}

// buildMySQLDisbursementScheduleFromScan constructs a DisbursementSchedule protobuf from scanned SQL fields.
// Scan order and column set are preserved exactly from the postgres gold standard
// so the Go-side response mapping is dialect-agnostic.
func buildMySQLDisbursementScheduleFromScan(
	id string, dateCreated time.Time, dateModified time.Time, active bool,
	expenditureID string, sequence int32, amount int64, dueDate int64, dueDateString *string, status string,
	paidAmount *int64, paidDate *int64, disbursementID *string, paymentTermID *string,
) *disbursementschedulepb.DisbursementSchedule {
	ds := &disbursementschedulepb.DisbursementSchedule{
		Id:            id,
		Active:        active,
		ExpenditureId: expenditureID,
		Sequence:      sequence,
		Amount:        amount,
		DueDate:       time.UnixMilli(dueDate).UTC().Format("2006-01-02"),
		Status:        status,
	}

	ds.DueDate = time.UnixMilli(dueDate).UTC().Format("2006-01-02")
	ds.PaidAmount = paidAmount
	ds.PaidDate = paidDate
	ds.DisbursementId = disbursementID
	ds.PaymentTermId = paymentTermID

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

// NewDisbursementScheduleRepository creates a new MySQL disbursement schedule repository (old-style constructor).
func NewDisbursementScheduleRepository(db *sql.DB, tableName string) disbursementschedulepb.DisbursementScheduleDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLDisbursementScheduleRepository(dbOps, tableName)
}
