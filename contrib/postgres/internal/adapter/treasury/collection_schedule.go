//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CollectionSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCollectionScheduleRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionScheduleRepository implements collection schedule CRUD operations using PostgreSQL
type PostgresCollectionScheduleRepository struct {
	collectionschedulepb.UnimplementedCollectionScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionScheduleRepository creates a new PostgreSQL collection schedule repository
func NewPostgresCollectionScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionschedulepb.CollectionScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "collection_schedule"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollectionSchedule creates a new collection schedule record
func (r *PostgresCollectionScheduleRepository) CreateCollectionSchedule(ctx context.Context, req *collectionschedulepb.CreateCollectionScheduleRequest) (*collectionschedulepb.CreateCollectionScheduleResponse, error) {
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

	postgresCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// ReadCollectionSchedule retrieves a collection schedule record by ID
func (r *PostgresCollectionScheduleRepository) ReadCollectionSchedule(ctx context.Context, req *collectionschedulepb.ReadCollectionScheduleRequest) (*collectionschedulepb.ReadCollectionScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection schedule: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// UpdateCollectionSchedule updates a collection schedule record
func (r *PostgresCollectionScheduleRepository) UpdateCollectionSchedule(ctx context.Context, req *collectionschedulepb.UpdateCollectionScheduleRequest) (*collectionschedulepb.UpdateCollectionScheduleResponse, error) {
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

	postgresCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// DeleteCollectionSchedule deletes a collection schedule record (soft delete)
func (r *PostgresCollectionScheduleRepository) DeleteCollectionSchedule(ctx context.Context, req *collectionschedulepb.DeleteCollectionScheduleRequest) (*collectionschedulepb.DeleteCollectionScheduleResponse, error) {
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

// ListCollectionSchedules lists collection schedule records with optional filters
func (r *PostgresCollectionScheduleRepository) ListCollectionSchedules(ctx context.Context, req *collectionschedulepb.ListCollectionSchedulesRequest) (*collectionschedulepb.ListCollectionSchedulesResponse, error) {
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
		postgresCore.ConvertMillisToDateStr(result, "due_date", "paid_date")
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
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

// GetCollectionScheduleListPageData retrieves collection schedules with pagination, filtering, sorting, and search using CTE
func (r *PostgresCollectionScheduleRepository) GetCollectionScheduleListPageData(
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

	sortField := "cs.date_created"
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
			WHERE cs.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       cs.status ILIKE $1)
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

		err := rows.Scan(
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collection schedule row: %w", err)
		}

		totalCount = total

		cs := buildCollectionScheduleFromScan(
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

// GetCollectionScheduleItemPageData retrieves a single collection schedule by ID using CTE
func (r *PostgresCollectionScheduleRepository) GetCollectionScheduleItemPageData(
	ctx context.Context,
	req *collectionschedulepb.GetCollectionScheduleItemPageDataRequest,
) (*collectionschedulepb.GetCollectionScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection schedule item page data request is required")
	}
	if req.CollectionScheduleId == "" {
		return nil, fmt.Errorf("collection schedule ID is required")
	}

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
			WHERE cs.id = $1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionScheduleId)

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

	cs := buildCollectionScheduleFromScan(
		id, dateCreated, dateModified, active,
		revenueID, sequence, amount, dueDate, dueDateString, status,
		paidAmount, paidDate, collectionID, paymentTermID,
	)

	return &collectionschedulepb.GetCollectionScheduleItemPageDataResponse{
		CollectionSchedule: cs,
		Success:            true,
	}, nil
}

// buildCollectionScheduleFromScan constructs a CollectionSchedule protobuf from scanned SQL fields
func buildCollectionScheduleFromScan(
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

// NewCollectionScheduleRepository creates a new PostgreSQL collection schedule repository (old-style constructor)
func NewCollectionScheduleRepository(db *sql.DB, tableName string) collectionschedulepb.CollectionScheduleDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionScheduleRepository(dbOps, tableName)
}