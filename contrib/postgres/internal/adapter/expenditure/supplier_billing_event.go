//go:build postgresql

// Package expenditure — postgres adapter for the supplier_billing_event
// domain service.
//
// supplier_billing_event is the buying-side mirror of the
// subscription/billing_event entity introduced by the advance-cash-events plan
// (20260517-advance-cash-events). Schema differences from BillingEvent:
//   - status + trigger are stored as INTEGER columns (per migration
//     20260517150000_advance_cash_events.sql), not TEXT enum strings.
//     The adapter therefore folds protojson's enum-name strings into the
//     numeric `_value` map on write, and protojson's DiscardUnknown read
//     restores them via the generated `_name` map automatically once the
//     row comes back as an integer.
//
// All other behaviour follows the billing_event adapter convention.
package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierBillingEventRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierBillingEventRepository implements supplier_billing_event CRUD using PostgreSQL.
type PostgresSupplierBillingEventRepository struct {
	pb.UnimplementedSupplierBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierBillingEventRepository creates a new PostgreSQL supplier_billing_event repository.
func NewPostgresSupplierBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SupplierBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.SupplierBillingEvent
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierBillingEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// foldEnumStringsToInt collapses the protojson enum-name strings on the
// `status` + `trigger` keys to their numeric proto wire values, because the
// underlying postgres columns are typed INTEGER (not TEXT).
//
// protojson writes enum fields as their wire-name strings (e.g.
// "SUPPLIER_BILLING_EVENT_STATUS_READY"); the WorkspaceAwareOperations layer
// passes the map straight through to a parameterised INSERT/UPDATE, so the
// driver would error with "invalid input syntax for type integer" without
// this fold. Reads are the inverse — protojson's UnmarshalOptions handles
// integer → enum just fine without any helper.
func foldEnumStringsToInt(data map[string]any) {
	if v, ok := data["status"].(string); ok {
		if num, ok := pb.SupplierBillingEventStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}
	if v, ok := data["trigger"].(string); ok {
		if num, ok := pb.SupplierBillingEventTrigger_value[v]; ok {
			data["trigger"] = int32(num)
		}
	}
}

// CreateSupplierBillingEvent inserts a new supplier_billing_event row.
func (r *PostgresSupplierBillingEventRepository) CreateSupplierBillingEvent(ctx context.Context, req *pb.CreateSupplierBillingEventRequest) (*pb.CreateSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("supplier_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event JSON to map: %w", err)
	}
	foldEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.CreateSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.SupplierBillingEvent{ev},
	}, nil
}

// ReadSupplierBillingEvent retrieves a supplier_billing_event by ID.
func (r *PostgresSupplierBillingEventRepository) ReadSupplierBillingEvent(ctx context.Context, req *pb.ReadSupplierBillingEventRequest) (*pb.ReadSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.ReadSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.SupplierBillingEvent{ev},
	}, nil
}

// UpdateSupplierBillingEvent updates a supplier_billing_event row.
func (r *PostgresSupplierBillingEventRepository) UpdateSupplierBillingEvent(ctx context.Context, req *pb.UpdateSupplierBillingEventRequest) (*pb.UpdateSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event JSON to map: %w", err)
	}
	foldEnumStringsToInt(data)

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event result: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event proto: %w", err)
	}
	return &pb.UpdateSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.SupplierBillingEvent{ev},
	}, nil
}

// DeleteSupplierBillingEvent soft-deletes a supplier_billing_event row.
func (r *PostgresSupplierBillingEventRepository) DeleteSupplierBillingEvent(ctx context.Context, req *pb.DeleteSupplierBillingEventRequest) (*pb.DeleteSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_billing_event: %w", err)
	}
	return &pb.DeleteSupplierBillingEventResponse{Success: true}, nil
}

// ListSupplierBillingEvents lists supplier_billing_event rows with optional
// filters, sort, search, and pagination.
func (r *PostgresSupplierBillingEventRepository) ListSupplierBillingEvents(ctx context.Context, req *pb.ListSupplierBillingEventsRequest) (*pb.ListSupplierBillingEventsResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_billing_events: %w", err)
	}
	events := make([]*pb.SupplierBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			log.Printf("WARN: marshal supplier_billing_event row: %v", err)
			continue
		}
		ev := &pb.SupplierBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, ev); err != nil {
			log.Printf("WARN: unmarshal supplier_billing_event row: %v", err)
			continue
		}
		events = append(events, ev)
	}
	return &pb.ListSupplierBillingEventsResponse{
		Success: true,
		Data:    events,
	}, nil
}

// GetSupplierBillingEventListPageData returns a paginated list view for the
// supplier_billing_event collection (delegates to dbOps for the heavy lifting
// since the table has no joins on the list page).
func (r *PostgresSupplierBillingEventRepository) GetSupplierBillingEventListPageData(
	ctx context.Context,
	req *pb.GetSupplierBillingEventListPageDataRequest,
) (*pb.GetSupplierBillingEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier_billing_event list page data request is required")
	}
	listResp, err := r.ListSupplierBillingEvents(ctx, &pb.ListSupplierBillingEventsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}
	// Pagination response is best-effort — dbOps.List returns a stable shape
	// when WorkspaceAwareOperations is wired with limit/offset; mirror what
	// the auto-generated adapter would surface.
	var page *commonpb.PaginationResponse
	if req.Pagination != nil && req.Pagination.GetOffset() != nil {
		currentPage := req.Pagination.GetOffset().Page
		totalItems := int32(len(listResp.Data))
		page = &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &currentPage,
		}
	}
	return &pb.GetSupplierBillingEventListPageDataResponse{
		SupplierBillingEventList: listResp.Data,
		Pagination:               page,
		Success:                  true,
	}, nil
}

// GetSupplierBillingEventItemPageData retrieves a single supplier_billing_event
// by ID for the detail page.
func (r *PostgresSupplierBillingEventRepository) GetSupplierBillingEventItemPageData(
	ctx context.Context,
	req *pb.GetSupplierBillingEventItemPageDataRequest,
) (*pb.GetSupplierBillingEventItemPageDataResponse, error) {
	if req == nil || req.SupplierBillingEventId == "" {
		return nil, fmt.Errorf("supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SupplierBillingEventId)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal supplier_billing_event row: %w", err)
	}
	ev := &pb.SupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal supplier_billing_event: %w", err)
	}
	return &pb.GetSupplierBillingEventItemPageDataResponse{
		SupplierBillingEvent: ev,
		Success:              true,
	}, nil
}
