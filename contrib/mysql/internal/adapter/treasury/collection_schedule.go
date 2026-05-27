//go:build mysql

// Package treasury — MySQL 8.0+ adapter for the collection_schedule entity.
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
	collectionschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.CollectionSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql collection_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLCollectionScheduleRepository(dbOps, tableName), nil
	})
}

// MySQLCollectionScheduleRepository implements collection schedule CRUD operations using MySQL 8.0+.
type MySQLCollectionScheduleRepository struct {
	collectionschedulepb.UnimplementedCollectionScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLCollectionScheduleRepository creates a new MySQL collection schedule repository.
func NewMySQLCollectionScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionschedulepb.CollectionScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "collection_schedule"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLCollectionScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollectionSchedule creates a new collection schedule record.
func (r *MySQLCollectionScheduleRepository) CreateCollectionSchedule(ctx context.Context, req *collectionschedulepb.CreateCollectionScheduleRequest) (*collectionschedulepb.CreateCollectionScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection schedule data is required")
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
		return nil, fmt.Errorf("failed to create collection schedule: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cs := &collectionschedulepb.CollectionSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionschedulepb.CreateCollectionScheduleResponse{
		Success: true,
		Data:    []*collectionschedulepb.CollectionSchedule{cs},
	}, nil
}

// ReadCollectionSchedule retrieves a collection schedule record by ID.
func (r *MySQLCollectionScheduleRepository) ReadCollectionSchedule(ctx context.Context, req *collectionschedulepb.ReadCollectionScheduleRequest) (*collectionschedulepb.ReadCollectionScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection schedule: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cs := &collectionschedulepb.CollectionSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionschedulepb.ReadCollectionScheduleResponse{
		Success: true,
		Data:    []*collectionschedulepb.CollectionSchedule{cs},
	}, nil
}

// UpdateCollectionSchedule updates a collection schedule record.
func (r *MySQLCollectionScheduleRepository) UpdateCollectionSchedule(ctx context.Context, req *collectionschedulepb.UpdateCollectionScheduleRequest) (*collectionschedulepb.UpdateCollectionScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
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
		return nil, fmt.Errorf("failed to update collection schedule: %w", err)
	}

	mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	cs := &collectionschedulepb.CollectionSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionschedulepb.UpdateCollectionScheduleResponse{
		Success: true,
		Data:    []*collectionschedulepb.CollectionSchedule{cs},
	}, nil
}

// DeleteCollectionSchedule deletes a collection schedule record (soft delete).
func (r *MySQLCollectionScheduleRepository) DeleteCollectionSchedule(ctx context.Context, req *collectionschedulepb.DeleteCollectionScheduleRequest) (*collectionschedulepb.DeleteCollectionScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection schedule: %w", err)
	}

	return &collectionschedulepb.DeleteCollectionScheduleResponse{
		Success: true,
	}, nil
}

// ListCollectionSchedules lists collection schedule records with optional filters.
func (r *MySQLCollectionScheduleRepository) ListCollectionSchedules(ctx context.Context, req *collectionschedulepb.ListCollectionSchedulesRequest) (*collectionschedulepb.ListCollectionSchedulesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection schedules: %w", err)
	}

	var schedules []*collectionschedulepb.CollectionSchedule
	for _, result := range listResult.Data {
		mysqlCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		cs := &collectionschedulepb.CollectionSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
			continue
		}
		schedules = append(schedules, cs)
	}

	return &collectionschedulepb.ListCollectionSchedulesResponse{
		Success: true,
		Data:    schedules,
	}, nil
}

// GetCollectionScheduleListPageData retrieves collection schedules with pagination,
// filtering, sorting, and search.
//
// Dialect translation from postgres gold standard:
//   - $1/$2/$3 → ? (positional, args: searchPattern, limit, offset)
//   - active = true → active = 1 (MySQL TINYINT(1))
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions supported)
//   - Sort field interpolated after whitelist validation — no $N needed
func (r *MySQLCollectionScheduleRepository) GetCollectionScheduleListPageData(
	ctx context.Context,
	req *collectionschedulepb.GetCollectionScheduleListPageDataRequest,
) (*collectionschedulepb.GetCollectionScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection schedule list page data request is required")
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
	collectionScheduleSortableSQLCols := []string{
		"cs.date_created", "cs.date_modified", "cs.amount",
		"cs.due_date", "cs.status", "cs.sequence",
	}
	sortClause, err := mysqlCore.BuildOrderBy(collectionScheduleSortableSQLCols, req.GetSort(), "cs.date_created DESC")
	if err != nil {
		return nil, err
	}

	// Dialect change: active = true → active = 1; ILIKE → LIKE; $1/$2/$3 → ?
	// searchPattern is "" when no search — WHERE clause handles empty string as no-op.
	query := `
		WITH enriched AS (
			SELECT
				cs.id,
				cs.date_created,
				cs.date_modified,
				cs.active,
				cs.revenue_id,
				cs.sequence,
				cs.amount,
				cs.due_date,
				cs.due_date_string,
				cs.status,
				cs.paid_amount,
				cs.paid_date,
				cs.collection_id,
				cs.payment_term_id
			FROM ` + r.tableName + ` cs
			WHERE cs.active = 1
			  AND (? = '' OR cs.status LIKE ?)
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
		return nil, fmt.Errorf("failed to query collection schedule list page data: %w", err)
	}
	defer rows.Close()

	var schedules []*collectionschedulepb.CollectionSchedule
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			revenueID     string
			sequence      int32
			amount        int64
			dueDate       int64
			dueDateString *string
			status        string
			paidAmount    *int64
			paidDate      *int64
			collectionID  *string
			paymentTermID *string
			total         int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&revenueID,
			&sequence,
			&amount,
			&dueDate,
			&dueDateString,
			&status,
			&paidAmount,
			&paidDate,
			&collectionID,
			&paymentTermID,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection schedule row: %w", err)
		}

		totalCount = total

		cs := buildMySQLCollectionScheduleFromScan(
			id, dateCreated, dateModified, active,
			revenueID, sequence, amount, dueDate, dueDateString, status,
			paidAmount, paidDate, collectionID, paymentTermID,
		)

		schedules = append(schedules, cs)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection schedule rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionschedulepb.GetCollectionScheduleListPageDataResponse{
		CollectionScheduleList: schedules,
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

// GetCollectionScheduleItemPageData retrieves a single collection schedule by ID.
//
// Dialect translation: $1 → ?; active = true → active = 1 not needed here (lookup by ID).
func (r *MySQLCollectionScheduleRepository) GetCollectionScheduleItemPageData(
	ctx context.Context,
	req *collectionschedulepb.GetCollectionScheduleItemPageDataRequest,
) (*collectionschedulepb.GetCollectionScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection schedule item page data request is required")
	}
	if req.CollectionScheduleId == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
	}

	// Dialect change: $1 → ?
	query := `
		WITH enriched AS (
			SELECT
				cs.id,
				cs.date_created,
				cs.date_modified,
				cs.active,
				cs.revenue_id,
				cs.sequence,
				cs.amount,
				cs.due_date,
				cs.due_date_string,
				cs.status,
				cs.paid_amount,
				cs.paid_date,
				cs.collection_id,
				cs.payment_term_id
			FROM ` + r.tableName + ` cs
			WHERE cs.id = ?
		)
		SELECT * FROM enriched LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.CollectionScheduleId)

	var (
		id            string
		dateCreated   time.Time
		dateModified  time.Time
		active        bool
		revenueID     string
		sequence      int32
		amount        int64
		dueDate       int64
		dueDateString *string
		status        string
		paidAmount    *int64
		paidDate      *int64
		collectionID  *string
		paymentTermID *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&revenueID,
		&sequence,
		&amount,
		&dueDate,
		&dueDateString,
		&status,
		&paidAmount,
		&paidDate,
		&collectionID,
		&paymentTermID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection schedule with ID '%s' not found", req.CollectionScheduleId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collection schedule item page data: %w", err)
	}

	cs := buildMySQLCollectionScheduleFromScan(
		id, dateCreated, dateModified, active,
		revenueID, sequence, amount, dueDate, dueDateString, status,
		paidAmount, paidDate, collectionID, paymentTermID,
	)

	return &collectionschedulepb.GetCollectionScheduleItemPageDataResponse{
		CollectionSchedule: cs,
		Success:            true,
	}, nil
}

// buildMySQLCollectionScheduleFromScan constructs a CollectionSchedule protobuf from scanned SQL fields.
// Scan order and column set are preserved exactly from the postgres gold standard
// so the Go-side response mapping is dialect-agnostic.
func buildMySQLCollectionScheduleFromScan(
	id string, dateCreated time.Time, dateModified time.Time, active bool,
	revenueID string, sequence int32, amount int64, dueDate int64, dueDateString *string, status string,
	paidAmount *int64, paidDate *int64, collectionID *string, paymentTermID *string,
) *collectionschedulepb.CollectionSchedule {
	cs := &collectionschedulepb.CollectionSchedule{
		Id:        id,
		Active:    active,
		RevenueId: revenueID,
		Sequence:  sequence,
		Amount:    amount,
		DueDate:   time.UnixMilli(dueDate).UTC().Format("2006-01-02"),
		Status:    status,
	}

	cs.DueDate = time.UnixMilli(dueDate).UTC().Format("2006-01-02")
	cs.PaidAmount = paidAmount
	cs.PaidDate = paidDate
	cs.CollectionId = collectionID
	cs.PaymentTermId = paymentTermID

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		cs.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		cs.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		cs.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		cs.DateModifiedString = &dmStr
	}

	return cs
}

// NewCollectionScheduleRepository creates a new MySQL collection schedule repository (old-style constructor).
func NewCollectionScheduleRepository(db *sql.DB, tableName string) collectionschedulepb.CollectionScheduleDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLCollectionScheduleRepository(dbOps, tableName)
}
