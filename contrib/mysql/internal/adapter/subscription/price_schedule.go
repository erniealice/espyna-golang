//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MySQLPriceScheduleRepository implements price_schedule CRUD operations using MySQL 8.0+.
type MySQLPriceScheduleRepository struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PriceSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql price_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPriceScheduleRepository(dbOps, tableName), nil
	})
}

// NewMySQLPriceScheduleRepository creates a new MySQL price schedule repository.
func NewMySQLPriceScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "price_schedule"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLPriceScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePriceSchedule creates a new price schedule using common MySQL operations.
func (r *MySQLPriceScheduleRepository) CreatePriceSchedule(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price schedule data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Empty optional FKs ("" from the inactive scope picker) must arrive as SQL NULL.
	// Keys are camelCase here because protojson emits camelCase by default;
	// normalizeKeys() inside dbOps.Create later converts them to snake_case.
	if v, ok := data["clientId"].(string); ok && v == "" {
		data["clientId"] = nil
	}
	if v, ok := data["locationId"].(string); ok && v == "" {
		data["locationId"] = nil
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price schedule: %w", err)
	}

	// date_time_start / date_time_end come back as int64 unix-millis from
	// normalizeValue; convert to RFC3339 so protojson can decode the Timestamp.
	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceSchedule := &priceschedulepb.PriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, priceSchedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceschedulepb.CreatePriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{priceSchedule},
	}, nil
}

// ReadPriceSchedule retrieves a price schedule using common MySQL operations.
func (r *MySQLPriceScheduleRepository) ReadPriceSchedule(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule: %w", err)
	}

	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceSchedule := &priceschedulepb.PriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, priceSchedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceschedulepb.ReadPriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{priceSchedule},
	}, nil
}

// UpdatePriceSchedule updates a price schedule using common MySQL operations.
func (r *MySQLPriceScheduleRepository) UpdatePriceSchedule(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// EmitDefaultValues ensures active:false is included.
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Empty optional FK values must reach the column as SQL NULL.
	if v, ok := data["clientId"].(string); ok && v == "" {
		data["clientId"] = nil
	}
	if v, ok := data["locationId"].(string); ok && v == "" {
		data["locationId"] = nil
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price schedule: %w", err)
	}

	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	priceSchedule := &priceschedulepb.PriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, priceSchedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceschedulepb.UpdatePriceScheduleResponse{
		Data: []*priceschedulepb.PriceSchedule{priceSchedule},
	}, nil
}

// DeletePriceSchedule permanently deletes a price schedule (hard delete).
// Soft delete (active=false) is handled separately by the set-status action.
func (r *MySQLPriceScheduleRepository) DeletePriceSchedule(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete price schedule: %w", err)
	}

	return &priceschedulepb.DeletePriceScheduleResponse{
		Success: true,
	}, nil
}

var priceScheduleSortableSQLCols = []string{
	"id", "active", "name", "description", "location_id", "client_id",
	"date_time_start", "date_time_end", "date_created", "date_modified",
}

var priceScheduleSortSpec = espynahttp.SortSpec{AllowedCols: priceScheduleSortableSQLCols}

// ListPriceSchedules lists price schedules using common MySQL operations.
func (r *MySQLPriceScheduleRepository) ListPriceSchedules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	if err := espynahttp.ValidateSortColumns(priceScheduleSortSpec, req.GetSort(), "price_schedule"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price schedules: %w", err)
	}

	var priceSchedules []*priceschedulepb.PriceSchedule
	for _, result := range listResult.Data {
		mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		priceSchedule := &priceschedulepb.PriceSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, priceSchedule); err != nil {
			continue
		}
		priceSchedules = append(priceSchedules, priceSchedule)
	}

	return &priceschedulepb.ListPriceSchedulesResponse{
		Data: priceSchedules,
	}, nil
}

// GetPriceScheduleListPageData retrieves paginated price schedule list data.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1
//   - WHERE workspace_id = ? added for multi-tenancy (price_schedule has own workspace_id)
//   - mysqlCore.BuildOrderBy used for safe sort interpolation
func (r *MySQLPriceScheduleRepository) GetPriceScheduleListPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	orderBy, err := mysqlCore.BuildOrderBy(priceScheduleSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for price schedule list: %w", err)
	}

	// A1: price_schedule has its own workspace_id column; scope directly.
	// Dialect: $N → ?, ILIKE → LIKE, active = true → active = 1,
	// WHERE workspace_id = ? added (postgres gold was missing this — added here per brief).
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end FROM price_schedule WHERE active = 1 AND (? = '' OR workspace_id = ?) AND (? IS NULL OR ? = '' OR name LIKE ? OR description LIKE ?) ` + orderBy + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, wsID, wsID, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var priceSchedules []*priceschedulepb.PriceSchedule
	for rows.Next() {
		var id, name, description string
		var active bool
		var dateCreated, dateModified time.Time
		var locationId sql.NullString
		var dateTimeStart, dateTimeEnd sql.NullTime
		if err := rows.Scan(&id, &name, &description, &active, &dateCreated, &dateModified, &locationId, &dateTimeStart, &dateTimeEnd); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		priceSchedule := &priceschedulepb.PriceSchedule{Id: id, Name: name, Description: &description, Active: active}
		if locationId.Valid && locationId.String != "" {
			priceSchedule.LocationId = &locationId.String
		}
		if dateTimeStart.Valid {
			priceSchedule.DateTimeStart = timestamppb.New(dateTimeStart.Time)
		}
		if dateTimeEnd.Valid {
			priceSchedule.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			priceSchedule.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			priceSchedule.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			priceSchedule.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			priceSchedule.DateModifiedString = &dmStr
		}
		priceSchedules = append(priceSchedules, priceSchedule)
	}
	return &priceschedulepb.GetPriceScheduleListPageDataResponse{PriceScheduleList: priceSchedules, Success: true}, nil
}

// GetPriceScheduleItemPageData retrieves price schedule item page data.
//
// Dialect: $1 → ?, active = true → active = 1.
func (r *MySQLPriceScheduleRepository) GetPriceScheduleItemPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	if req == nil || req.PriceScheduleId == "" {
		return nil, fmt.Errorf("price schedule ID required")
	}
	query := `SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end FROM price_schedule WHERE id = ? AND active = 1`
	row := r.db.QueryRowContext(ctx, query, req.PriceScheduleId)
	var id, name, description string
	var active bool
	var dateCreated, dateModified time.Time
	var locationId sql.NullString
	var dateTimeStart, dateTimeEnd sql.NullTime
	if err := row.Scan(&id, &name, &description, &active, &dateCreated, &dateModified, &locationId, &dateTimeStart, &dateTimeEnd); err == sql.ErrNoRows {
		return nil, fmt.Errorf("price schedule not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	priceSchedule := &priceschedulepb.PriceSchedule{Id: id, Name: name, Description: &description, Active: active}
	if locationId.Valid && locationId.String != "" {
		priceSchedule.LocationId = &locationId.String
	}
	if dateTimeStart.Valid {
		priceSchedule.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if dateTimeEnd.Valid {
		priceSchedule.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		priceSchedule.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		priceSchedule.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		priceSchedule.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		priceSchedule.DateModifiedString = &dmStr
	}
	return &priceschedulepb.GetPriceScheduleItemPageDataResponse{PriceSchedule: priceSchedule, Success: true}, nil
}

// FindApplicablePriceSchedule finds the active price schedule for a given location and date.
// Returns the most recently started price schedule that covers the given date.
//
// Dialect: $N → ?, active = true → active = 1.
func (r *MySQLPriceScheduleRepository) FindApplicablePriceSchedule(ctx context.Context, req *priceschedulepb.FindApplicablePriceScheduleRequest) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
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

	// Dialect: $N → ?, active = true → active = 1, no date_time_end IS NULL
	// (MySQL NULL comparison is identical to postgres for IS NULL).
	query := `
		SELECT id, name, description, active, date_time_start, date_time_end, location_id, date_created, date_modified
		FROM price_schedule
		WHERE active = 1
		  AND location_id = ?
		  AND date_time_start <= ?
		  AND (date_time_end >= ? OR date_time_end IS NULL)
		ORDER BY date_time_start DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.LocationId, reqTime, reqTime)

	var id, name string
	var description, locationId sql.NullString
	var dateTimeStart, dateTimeEnd sql.NullTime
	var active bool
	var dateCreated, dateModified time.Time

	err = row.Scan(&id, &name, &description, &active, &dateTimeStart, &dateTimeEnd, &locationId, &dateCreated, &dateModified)
	if err == sql.ErrNoRows {
		return &priceschedulepb.FindApplicablePriceScheduleResponse{
			Found:   false,
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	priceSchedule := &priceschedulepb.PriceSchedule{Id: id, Name: name, Active: active}
	if description.Valid {
		priceSchedule.Description = &description.String
	}
	if dateTimeStart.Valid {
		priceSchedule.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if locationId.Valid {
		priceSchedule.LocationId = &locationId.String
	}
	if dateTimeEnd.Valid {
		priceSchedule.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		priceSchedule.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		priceSchedule.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		priceSchedule.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		priceSchedule.DateModifiedString = &dmStr
	}

	return &priceschedulepb.FindApplicablePriceScheduleResponse{
		PriceSchedule: priceSchedule,
		Found:         true,
		Success:       true,
	}, nil
}

// NewPriceScheduleRepository creates a new MySQL price_schedule repository (old-style constructor).
func NewPriceScheduleRepository(db *sql.DB, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLPriceScheduleRepository(dbOps, tableName)
}
