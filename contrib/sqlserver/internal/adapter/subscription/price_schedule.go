//go:build sqlserver

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SQLServerPriceScheduleRepository implements price_schedule CRUD operations using SQL Server.
type SQLServerPriceScheduleRepository struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PriceSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver price_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPriceScheduleRepository(dbOps, tableName), nil
	})
}

// NewSQLServerPriceScheduleRepository creates a new SQL Server price schedule repository.
func NewSQLServerPriceScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "price_schedule"
	}
	return &SQLServerPriceScheduleRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePriceSchedule creates a new price schedule using common SQL Server operations.
func (r *SQLServerPriceScheduleRepository) CreatePriceSchedule(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
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

// ReadPriceSchedule retrieves a price schedule using common SQL Server operations.
func (r *SQLServerPriceScheduleRepository) ReadPriceSchedule(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule: %w", err)
	}

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

// UpdatePriceSchedule updates a price schedule using common SQL Server operations.
func (r *SQLServerPriceScheduleRepository) UpdatePriceSchedule(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

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
func (r *SQLServerPriceScheduleRepository) DeletePriceSchedule(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
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

// ListPriceSchedules lists price schedules using common SQL Server operations.
func (r *SQLServerPriceScheduleRepository) ListPriceSchedules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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
// SQL Server differences:
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - active = true → active = 1.
//   - ILIKE → LIKE.
//   - LIMIT/OFFSET → OFFSET/FETCH NEXT.
//   - sortField validated against allowlist.
func (r *SQLServerPriceScheduleRepository) GetPriceScheduleListPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
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
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			offset = (offsetPag.Page - 1) * limit
		}
	}
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sf := req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
		for _, col := range priceScheduleSortableSQLCols {
			if col == sf {
				sortField = sf
				break
			}
		}
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end
		FROM price_schedule
		WHERE active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR name LIKE @p1 OR description LIKE @p1)
		ORDER BY [%s] %s
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
	`, sortField, sortOrder)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, limit, offset)
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
// SQL Server differences: $1 → @p1; active = true → active = 1.
func (r *SQLServerPriceScheduleRepository) GetPriceScheduleItemPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	if req == nil || req.PriceScheduleId == "" {
		return nil, fmt.Errorf("price schedule ID required")
	}

	query := `SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end
		FROM price_schedule WHERE id = @p1 AND active = 1`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.PriceScheduleId)

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
//
// SQL Server differences:
//   - $1/$2 → @p1/@p2.
//   - LIMIT 1 → SELECT TOP 1.
//   - active = true → active = 1.
func (r *SQLServerPriceScheduleRepository) FindApplicablePriceSchedule(ctx context.Context, req *priceschedulepb.FindApplicablePriceScheduleRequest) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
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

	// SQL Server: SELECT TOP 1 instead of LIMIT 1; active = 1; @pN placeholders.
	query := `
		SELECT TOP 1 id, name, description, active, date_time_start, date_time_end, location_id, date_created, date_modified
		FROM price_schedule
		WHERE active = 1
		  AND location_id = @p1
		  AND date_time_start <= @p2
		  AND (date_time_end >= @p2 OR date_time_end IS NULL)
		ORDER BY date_time_start DESC`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.LocationId, reqTime)

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

// NewPriceScheduleRepository creates a new SQL Server price_schedule repository (old-style constructor).
func NewPriceScheduleRepository(db *sql.DB, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerPriceScheduleRepository(dbOps, tableName)
}
