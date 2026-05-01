//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PostgresPriceScheduleRepository implements price_schedule CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_price_schedule_active ON price_schedule(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_price_schedule_date_time_start ON price_schedule(date_time_start) - Filter by start timestamp
//   - CREATE INDEX idx_price_schedule_date_time_end ON price_schedule(date_time_end) - Filter by end timestamp
//   - CREATE INDEX idx_price_schedule_date_created ON price_schedule(date_created DESC) - Default sorting
type PostgresPriceScheduleRepository struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PriceSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPriceScheduleRepository(dbOps, tableName), nil
	})
}

// NewPostgresPriceScheduleRepository creates a new PostgreSQL price schedule repository
func NewPostgresPriceScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "price_schedule" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPriceScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePriceSchedule creates a new price schedule using common PostgreSQL operations
func (r *PostgresPriceScheduleRepository) CreatePriceSchedule(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price schedule data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Empty optional FKs ("" from the inactive scope picker) must arrive at
	// postgres as SQL NULL — the Scope radio (2026-04-28) clears the inactive
	// FK on save, and an empty literal would trip the FK constraint.
	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}
	if v, ok := data["location_id"].(string); ok && v == "" {
		data["location_id"] = nil
	}

	// Create document using common operations.
	// date_time_start / date_time_end arrive as RFC3339 strings (protojson
	// representation of google.protobuf.Timestamp). PostgreSQL TIMESTAMPTZ
	// accepts that directly, so no manual conversion is needed.
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price schedule: %w", err)
	}

	// date_time_start / date_time_end come back as int64 unix-millis from
	// normalizeValue; convert to RFC3339 so protojson can decode the Timestamp.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result back to protobuf using protojson
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

// ReadPriceSchedule retrieves a price schedule using common PostgreSQL operations
func (r *PostgresPriceScheduleRepository) ReadPriceSchedule(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule: %w", err)
	}

	// Same date_time_start/date_time_end conversion as CreatePriceSchedule — see comment there.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result to protobuf using protojson
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

// UpdatePriceSchedule updates a price schedule using common PostgreSQL operations
func (r *PostgresPriceScheduleRepository) UpdatePriceSchedule(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	// Convert protobuf to map using protojson (EmitDefaultValues ensures active:false is included)
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Empty optional FK values must reach the column as SQL NULL — the
	// scope radio (2026-04-28) flips between location-scoped and client-scoped
	// schedules, clearing the inactive FK each time. EmitDefaultValues serialises
	// nil/&"" as the empty string which would trip the FK constraint.
	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}
	if v, ok := data["location_id"].(string); ok && v == "" {
		data["location_id"] = nil
	}

	// Update document using common operations.
	// date_time_start / date_time_end arrive as RFC3339 strings (protojson
	// representation of google.protobuf.Timestamp). PostgreSQL TIMESTAMPTZ
	// accepts that directly, so no manual conversion is needed.
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price schedule: %w", err)
	}

	// Same date_time_start/date_time_end conversion as CreatePriceSchedule — see comment there.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result back to protobuf using protojson
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

// DeletePriceSchedule permanently deletes a price schedule.
// Soft delete (active=false) is handled separately by the set-status action,
// so Delete is always a hard delete here.
func (r *PostgresPriceScheduleRepository) DeletePriceSchedule(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id)
	if err != nil {
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

// ListPriceSchedules lists price schedules using common PostgreSQL operations
func (r *PostgresPriceScheduleRepository) ListPriceSchedules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
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

	// Convert results to protobuf slice using protojson
	var priceSchedules []*priceschedulepb.PriceSchedule
	for _, result := range listResult.Data {
		// Same date_time_start/date_time_end conversion as CreatePriceSchedule — see comment there.
		postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		priceSchedule := &priceschedulepb.PriceSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, priceSchedule); err != nil {
			// Log error and continue with next item
			continue
		}
		priceSchedules = append(priceSchedules, priceSchedule)
	}

	return &priceschedulepb.ListPriceSchedulesResponse{
		Data: priceSchedules,
	}, nil
}

// GetPriceScheduleListPageData retrieves paginated price schedule list data with CTE
func (r *PostgresPriceScheduleRepository) GetPriceScheduleListPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
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
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end FROM price_schedule WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1 OR description ILIKE $1) ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var priceSchedules []*priceschedulepb.PriceSchedule
	var totalCount int64
	for rows.Next() {
		var id, name, description string
		var active bool
		var dateCreated, dateModified time.Time
		var locationId sql.NullString
		var dateTimeStart, dateTimeEnd sql.NullTime
		if err := rows.Scan(&id, &name, &description, &active, &dateCreated, &dateModified, &locationId, &dateTimeStart, &dateTimeEnd); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount++
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

// GetPriceScheduleItemPageData retrieves price schedule item page data
func (r *PostgresPriceScheduleRepository) GetPriceScheduleItemPageData(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	if req == nil || req.PriceScheduleId == "" {
		return nil, fmt.Errorf("price schedule ID required")
	}
	query := `SELECT id, name, description, active, date_created, date_modified, location_id, date_time_start, date_time_end FROM price_schedule WHERE id = $1 AND active = true`
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
// If no match is found, returns found=false with no error (not an error condition).
// If multiple rows match, the one with the latest date_time_start wins (most specific/recent wins).
func (r *PostgresPriceScheduleRepository) FindApplicablePriceSchedule(ctx context.Context, req *priceschedulepb.FindApplicablePriceScheduleRequest) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
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
		SELECT id, name, description, active, date_time_start, date_time_end, location_id, date_created, date_modified
		FROM price_schedule
		WHERE active = true
		  AND location_id = $1
		  AND date_time_start <= $2
		  AND (date_time_end >= $2 OR date_time_end IS NULL)
		ORDER BY date_time_start DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.LocationId, reqTime)

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

// NewPriceScheduleRepository creates a new PostgreSQL price_schedule repository (old-style constructor)
func NewPriceScheduleRepository(db *sql.DB, tableName string) priceschedulepb.PriceScheduleDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPriceScheduleRepository(dbOps, tableName)
}