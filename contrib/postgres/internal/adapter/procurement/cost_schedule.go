//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PostgresCostScheduleRepository implements cost_schedule CRUD operations using PostgreSQL.
type PostgresCostScheduleRepository struct {
	costschedulepb.UnimplementedCostScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CostSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres cost_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCostScheduleRepository(dbOps, tableName), nil
	})
}

// NewPostgresCostScheduleRepository creates a new PostgreSQL cost schedule repository.
func NewPostgresCostScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) costschedulepb.CostScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "cost_schedule"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresCostScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *PostgresCostScheduleRepository) CreateCostSchedule(ctx context.Context, req *costschedulepb.CreateCostScheduleRequest) (*costschedulepb.CreateCostScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("cost schedule data is required")
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
		return nil, fmt.Errorf("failed to create cost schedule: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cs := &costschedulepb.CostSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costschedulepb.CreateCostScheduleResponse{Data: []*costschedulepb.CostSchedule{cs}}, nil
}

func (r *PostgresCostScheduleRepository) ReadCostSchedule(ctx context.Context, req *costschedulepb.ReadCostScheduleRequest) (*costschedulepb.ReadCostScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read cost schedule: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cs := &costschedulepb.CostSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costschedulepb.ReadCostScheduleResponse{Data: []*costschedulepb.CostSchedule{cs}}, nil
}

func (r *PostgresCostScheduleRepository) UpdateCostSchedule(ctx context.Context, req *costschedulepb.UpdateCostScheduleRequest) (*costschedulepb.UpdateCostScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost schedule ID is required")
	}
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update cost schedule: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cs := &costschedulepb.CostSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costschedulepb.UpdateCostScheduleResponse{Data: []*costschedulepb.CostSchedule{cs}}, nil
}

func (r *PostgresCostScheduleRepository) DeleteCostSchedule(ctx context.Context, req *costschedulepb.DeleteCostScheduleRequest) (*costschedulepb.DeleteCostScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost schedule ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete cost schedule: %w", err)
	}
	return &costschedulepb.DeleteCostScheduleResponse{Success: true}, nil
}

func (r *PostgresCostScheduleRepository) ListCostSchedules(ctx context.Context, req *costschedulepb.ListCostSchedulesRequest) (*costschedulepb.ListCostSchedulesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list cost schedules: %w", err)
	}
	var items []*costschedulepb.CostSchedule
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		cs := &costschedulepb.CostSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cs); err != nil {
			continue
		}
		items = append(items, cs)
	}
	return &costschedulepb.ListCostSchedulesResponse{Data: items}, nil
}

// costScheduleSortableSQLCols is the fail-closed sort whitelist for
// GetCostScheduleListPageData. Only columns projected by the SELECT are included
// so ORDER BY can never reference an unprojected/injected identifier.
var costScheduleSortableSQLCols = []string{
	"id", "name", "description", "active", "date_created", "date_modified",
	"date_time_start", "date_time_end",
}

func (r *PostgresCostScheduleRepository) GetCostScheduleListPageData(ctx context.Context, req *costschedulepb.GetCostScheduleListPageDataRequest) (*costschedulepb.GetCostScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset := int32(50), int32(0)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if op := req.Pagination.GetOffset(); op != nil && op.Page > 0 {
			offset = (op.Page - 1) * limit
		}
	}
	// Sort — fail-closed against the per-entity whitelist (A2 guard). An unknown
	// sort column now errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(costScheduleSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}
	query := `SELECT id, name, description, active, date_created, date_modified, date_time_start, date_time_end
	          FROM cost_schedule
	          WHERE active = true
	            AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1 OR description ILIKE $1)
	          ` + orderByClause + ` LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*costschedulepb.CostSchedule
	for rows.Next() {
		var id, name string
		var description sql.NullString
		var active bool
		var dateCreated, dateModified time.Time
		var dateTimeStart, dateTimeEnd sql.NullTime
		if err := rows.Scan(&id, &name, &description, &active, &dateCreated, &dateModified, &dateTimeStart, &dateTimeEnd); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		cs := &costschedulepb.CostSchedule{Id: id, Name: name, Active: active}
		if description.Valid {
			cs.Description = &description.String
		}
		if dateTimeStart.Valid {
			cs.DateTimeStart = timestamppb.New(dateTimeStart.Time)
		}
		if dateTimeEnd.Valid {
			cs.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
		}
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
		items = append(items, cs)
	}
	return &costschedulepb.GetCostScheduleListPageDataResponse{CostScheduleList: items, Success: true}, nil
}

func (r *PostgresCostScheduleRepository) GetCostScheduleItemPageData(ctx context.Context, req *costschedulepb.GetCostScheduleItemPageDataRequest) (*costschedulepb.GetCostScheduleItemPageDataResponse, error) {
	if req == nil || req.CostScheduleId == "" {
		return nil, fmt.Errorf("cost schedule ID required")
	}
	query := `SELECT id, name, description, active, date_created, date_modified, date_time_start, date_time_end
	          FROM cost_schedule WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, req.CostScheduleId)
	var id, name string
	var description sql.NullString
	var active bool
	var dateCreated, dateModified time.Time
	var dateTimeStart, dateTimeEnd sql.NullTime
	if err := row.Scan(&id, &name, &description, &active, &dateCreated, &dateModified, &dateTimeStart, &dateTimeEnd); err == sql.ErrNoRows {
		return nil, fmt.Errorf("cost schedule not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	cs := &costschedulepb.CostSchedule{Id: id, Name: name, Active: active}
	if description.Valid {
		cs.Description = &description.String
	}
	if dateTimeStart.Valid {
		cs.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if dateTimeEnd.Valid {
		cs.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
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
	return &costschedulepb.GetCostScheduleItemPageDataResponse{CostSchedule: cs, Success: true}, nil
}

// FindApplicableCostSchedule finds the active cost schedule covering the given location and date.
// Returns the most recently started schedule that covers the date; found=false with no error when none match.
func (r *PostgresCostScheduleRepository) FindApplicableCostSchedule(ctx context.Context, req *costschedulepb.FindApplicableCostScheduleRequest) (*costschedulepb.FindApplicableCostScheduleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.LocationId == "" {
		return nil, fmt.Errorf("location_id is required")
	}
	if req.Date == "" {
		return nil, fmt.Errorf("date is required")
	}

	manila, _ := time.LoadLocation("Asia/Manila")
	reqTime, err := time.ParseInLocation("2006-01-02", req.Date, manila)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}

	query := `
		SELECT id, name, description, active, date_time_start, date_time_end, date_created, date_modified
		FROM cost_schedule
		WHERE active = true
		  AND location_id = $1
		  AND date_time_start <= $2
		  AND (date_time_end >= $2 OR date_time_end IS NULL)
		ORDER BY date_time_start DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.LocationId, reqTime)
	var id, name string
	var description sql.NullString
	var active bool
	var dateTimeStart, dateTimeEnd sql.NullTime
	var dateCreated, dateModified time.Time

	err = row.Scan(&id, &name, &description, &active, &dateTimeStart, &dateTimeEnd, &dateCreated, &dateModified)
	if err == sql.ErrNoRows {
		return &costschedulepb.FindApplicableCostScheduleResponse{Found: false, Success: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	cs := &costschedulepb.CostSchedule{Id: id, Name: name, Active: active}
	if description.Valid {
		cs.Description = &description.String
	}
	if dateTimeStart.Valid {
		cs.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if dateTimeEnd.Valid {
		cs.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
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
	return &costschedulepb.FindApplicableCostScheduleResponse{CostSchedule: cs, Found: true, Success: true}, nil
}
