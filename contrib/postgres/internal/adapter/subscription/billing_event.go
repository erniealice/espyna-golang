//go:build postgresql

package subscription

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.BillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresBillingEventRepository(dbOps, tableName), nil
	})
}

// PostgresBillingEventRepository implements billing_event CRUD operations using PostgreSQL.
type PostgresBillingEventRepository struct {
	pb.UnimplementedBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresBillingEventRepository creates a new PostgreSQL billing_event repository.
func NewPostgresBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.BillingEventDomainServiceServer {
	if tableName == "" {
		tableName = "billing_event"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresBillingEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateBillingEvent inserts a new billing_event row.
func (r *PostgresBillingEventRepository) CreateBillingEvent(ctx context.Context, req *pb.CreateBillingEventRequest) (*pb.CreateBillingEventResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("billing event data is required")
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
		return nil, fmt.Errorf("failed to create billing event: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ev := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateBillingEventResponse{
		Success: true,
		Data:    []*pb.BillingEvent{ev},
	}, nil
}

// ReadBillingEvent retrieves a billing_event by ID.
func (r *PostgresBillingEventRepository) ReadBillingEvent(ctx context.Context, req *pb.ReadBillingEventRequest) (*pb.ReadBillingEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billing event ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read billing event: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ev := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadBillingEventResponse{
		Success: true,
		Data:    []*pb.BillingEvent{ev},
	}, nil
}

// UpdateBillingEvent updates a billing_event row.
func (r *PostgresBillingEventRepository) UpdateBillingEvent(ctx context.Context, req *pb.UpdateBillingEventRequest) (*pb.UpdateBillingEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billing event ID is required")
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
		return nil, fmt.Errorf("failed to update billing event: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ev := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateBillingEventResponse{
		Success: true,
		Data:    []*pb.BillingEvent{ev},
	}, nil
}

// DeleteBillingEvent soft-deletes a billing_event row.
func (r *PostgresBillingEventRepository) DeleteBillingEvent(ctx context.Context, req *pb.DeleteBillingEventRequest) (*pb.DeleteBillingEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billing event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete billing event: %w", err)
	}
	return &pb.DeleteBillingEventResponse{Success: true}, nil
}

// ListBillingEvents lists billing_event rows with optional filters.
func (r *PostgresBillingEventRepository) ListBillingEvents(ctx context.Context, req *pb.ListBillingEventsRequest) (*pb.ListBillingEventsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list billing events: %w", err)
	}

	var events []*pb.BillingEvent
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal billing_event row: %v", err)
			continue
		}
		ev := &pb.BillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
			log.Printf("WARN: protojson unmarshal billing_event: %v", err)
			continue
		}
		events = append(events, ev)
	}

	return &pb.ListBillingEventsResponse{
		Success: true,
		Data:    events,
	}, nil
}

// GetBillingEventListPageData returns a paginated list view for the billing_event collection.
func (r *PostgresBillingEventRepository) GetBillingEventListPageData(
	ctx context.Context,
	req *pb.GetBillingEventListPageDataRequest,
) (*pb.GetBillingEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get billing event list page data request is required")
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH base AS (SELECT * FROM billing_event WHERE active = true), counted AS (SELECT COUNT(*) AS total FROM base) SELECT b.*, c.total FROM base b, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $1 OFFSET $2;`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing event list page data: %w", err)
	}
	defer rows.Close()

	var events []*pb.BillingEvent
	var totalCount int64
	for rows.Next() {
		raw := map[string]any{}
		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("rows.Columns: %w", err)
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan billing_event row: %w", err)
		}
		for i, c := range cols {
			raw[c] = vals[i]
		}
		if t, ok := raw["total"].(int64); ok {
			totalCount = t
		}
		delete(raw, "total")

		// Convert to camelCase keys for protojson.
		dataJSON, _ := json.Marshal(postgresCore.DenormalizeKeys(raw))
		ev := &pb.BillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, ev); err == nil {
			events = append(events, ev)
		}
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetBillingEventListPageDataResponse{
		BillingEventList: events,
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

// GetBillingEventItemPageData retrieves a single billing_event by ID for the detail page.
func (r *PostgresBillingEventRepository) GetBillingEventItemPageData(
	ctx context.Context,
	req *pb.GetBillingEventItemPageDataRequest,
) (*pb.GetBillingEventItemPageDataResponse, error) {
	if req == nil || req.BillingEventId == "" {
		return nil, fmt.Errorf("billing event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.BillingEventId)
	if err != nil {
		return nil, fmt.Errorf("failed to read billing event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal billing_event row: %w", err)
	}
	ev := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal billing_event: %w", err)
	}
	return &pb.GetBillingEventItemPageDataResponse{
		BillingEvent: ev,
		Success:      true,
	}, nil
}

// ListBySubscription returns all billing_event rows linked to a subscription.
func (r *PostgresBillingEventRepository) ListBySubscription(
	ctx context.Context,
	req *pb.ListBillingEventsBySubscriptionRequest,
) (*pb.ListBillingEventsBySubscriptionResponse, error) {
	if req == nil || req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	events, err := r.listByColumn(ctx, "subscription_id", req.SubscriptionId)
	if err != nil {
		return nil, err
	}

	return &pb.ListBillingEventsBySubscriptionResponse{
		BillingEvents: events,
		Success:       true,
	}, nil
}

// ListByJobPhase returns all billing_event rows linked to a job phase.
func (r *PostgresBillingEventRepository) ListByJobPhase(
	ctx context.Context,
	req *pb.ListBillingEventsByJobPhaseRequest,
) (*pb.ListBillingEventsByJobPhaseResponse, error) {
	if req == nil || req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	events, err := r.listByColumn(ctx, "job_phase_id", req.JobPhaseId)
	if err != nil {
		return nil, err
	}

	return &pb.ListBillingEventsByJobPhaseResponse{
		BillingEvents: events,
		Success:       true,
	}, nil
}

// listByColumn is the shared list-with-equality helper used by ListBy* RPCs.
func (r *PostgresBillingEventRepository) listByColumn(
	ctx context.Context, column, value string,
) ([]*pb.BillingEvent, error) {
	if r.db == nil {
		return nil, fmt.Errorf("billing_event repository missing *sql.DB")
	}
	// Validate column to avoid SQL injection — only allowlisted names.
	switch column {
	case "subscription_id", "job_phase_id", "job_id", "job_template_phase_id":
		// ok
	default:
		return nil, fmt.Errorf("unsupported list column: %s", column)
	}

	query := `SELECT * FROM ` + r.tableName + ` WHERE ` + column + ` = $1 AND active = true ORDER BY date_created ASC`
	rows, err := r.db.QueryContext(ctx, query, value)
	if err != nil {
		return nil, fmt.Errorf("billing_event query (%s=%s): %w", column, value, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("rows.Columns: %w", err)
	}

	var out []*pb.BillingEvent
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan billing_event row: %w", err)
		}
		raw := map[string]any{}
		for i, c := range cols {
			raw[c] = normalizeScanValue(vals[i])
		}
		dataJSON, err := json.Marshal(postgresCore.DenormalizeKeys(raw))
		if err != nil {
			log.Printf("WARN: marshal billing_event row: %v", err)
			continue
		}
		ev := &pb.BillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, ev); err != nil {
			log.Printf("WARN: unmarshal billing_event row: %v", err)
			continue
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing_event row iter: %w", err)
	}
	return out, nil
}

// SetStatus mutates only the status/trigger/reason/timestamps of a billing_event.
// All other fields stay untouched; the row's date_modified is bumped.
func (r *PostgresBillingEventRepository) SetStatus(
	ctx context.Context,
	req *pb.SetBillingEventStatusRequest,
) (*pb.SetBillingEventStatusResponse, error) {
	if req == nil || req.BillingEventId == "" {
		return nil, fmt.Errorf("billing event ID is required")
	}

	// Read first to fold the new status into the existing row.
	read, err := r.dbOps.Read(ctx, r.tableName, req.BillingEventId)
	if err != nil {
		return nil, fmt.Errorf("read billing_event: %w", err)
	}

	// Convert raw map back to proto.
	readJSON, _ := json.Marshal(read)
	current := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(readJSON, current); err != nil {
		return nil, fmt.Errorf("unmarshal billing_event: %w", err)
	}

	current.Status = req.Status
	current.Trigger = req.Trigger
	if req.Reason != nil {
		current.Reason = req.Reason
	}
	now := time.Now().UnixMilli()
	switch req.Status {
	case pb.BillingEventStatus_BILLING_EVENT_STATUS_READY:
		current.TriggeredAt = &now
	case pb.BillingEventStatus_BILLING_EVENT_STATUS_BILLED:
		current.BilledAt = &now
	}
	current.DateModified = &now
	dms := time.Now().Format(time.RFC3339)
	current.DateModifiedString = &dms

	updateJSON, _ := protojson.Marshal(current)
	var updateMap map[string]any
	if err := json.Unmarshal(updateJSON, &updateMap); err != nil {
		return nil, fmt.Errorf("billing_event update marshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, current.Id, updateMap)
	if err != nil {
		return nil, fmt.Errorf("update billing_event: %w", err)
	}

	resultJSON, _ := json.Marshal(result)
	out := &pb.BillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, out); err != nil {
		return nil, fmt.Errorf("unmarshal updated billing_event: %w", err)
	}

	return &pb.SetBillingEventStatusResponse{
		Data:    out,
		Success: true,
	}, nil
}

// normalizeScanValue handles the standard set of database/sql.Scan return types
// so map -> JSON serialization yields canonical proto-friendly values.
func normalizeScanValue(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.UnixMilli()
	default:
		return t
	}
}
