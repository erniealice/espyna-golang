//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierlifecycleeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_lifecycle_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierLifecycleEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_lifecycle_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierLifecycleEventRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierLifecycleEventRepository implements supplier lifecycle event operations using PostgreSQL.
// This entity is APPEND-ONLY: only Create / Read / List are exposed; Update / Delete are intentionally
// omitted (the embedded UnimplementedSupplierLifecycleEventDomainServiceServer satisfies any future
// surface area added to the interface).
type PostgresSupplierLifecycleEventRepository struct {
	supplierlifecycleeventpb.UnimplementedSupplierLifecycleEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierLifecycleEventRepository creates a new PostgreSQL supplier lifecycle event repository.
func NewPostgresSupplierLifecycleEventRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierlifecycleeventpb.SupplierLifecycleEventDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_lifecycle_event"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierLifecycleEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSupplierLifecycleEvent appends a new supplier lifecycle event row.
func (r *PostgresSupplierLifecycleEventRepository) CreateSupplierLifecycleEvent(ctx context.Context, req *supplierlifecycleeventpb.CreateSupplierLifecycleEventRequest) (*supplierlifecycleeventpb.CreateSupplierLifecycleEventResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier lifecycle event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_lifecycle_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sle := &supplierlifecycleeventpb.SupplierLifecycleEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierlifecycleeventpb.CreateSupplierLifecycleEventResponse{Success: true, Data: []*supplierlifecycleeventpb.SupplierLifecycleEvent{sle}}, nil
}

// ReadSupplierLifecycleEvent retrieves a supplier lifecycle event by ID.
func (r *PostgresSupplierLifecycleEventRepository) ReadSupplierLifecycleEvent(ctx context.Context, req *supplierlifecycleeventpb.ReadSupplierLifecycleEventRequest) (*supplierlifecycleeventpb.ReadSupplierLifecycleEventResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier lifecycle event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_lifecycle_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sle := &supplierlifecycleeventpb.SupplierLifecycleEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierlifecycleeventpb.ReadSupplierLifecycleEventResponse{Success: true, Data: []*supplierlifecycleeventpb.SupplierLifecycleEvent{sle}}, nil
}

// ListSupplierLifecycleEvents lists supplier lifecycle event records with optional filters.
func (r *PostgresSupplierLifecycleEventRepository) ListSupplierLifecycleEvents(ctx context.Context, req *supplierlifecycleeventpb.ListSupplierLifecycleEventsRequest) (*supplierlifecycleeventpb.ListSupplierLifecycleEventsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_lifecycle_events: %w", err)
	}
	var items []*supplierlifecycleeventpb.SupplierLifecycleEvent
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_lifecycle_event row: %v", err)
			continue
		}
		sle := &supplierlifecycleeventpb.SupplierLifecycleEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sle); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_lifecycle_event: %v", err)
			continue
		}
		items = append(items, sle)
	}
	return &supplierlifecycleeventpb.ListSupplierLifecycleEventsResponse{Success: true, Data: items}, nil
}

// GetSupplierLifecycleEventListPageData retrieves supplier lifecycle events with pagination, filtering, sorting, and search.
func (r *PostgresSupplierLifecycleEventRepository) GetSupplierLifecycleEventListPageData(
	ctx context.Context,
	req *supplierlifecycleeventpb.GetSupplierLifecycleEventListPageDataRequest,
) (*supplierlifecycleeventpb.GetSupplierLifecycleEventListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier lifecycle event list page data request is required")
	}

	var params *interfaces.ListParams
	if req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
			}
		}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_lifecycle_event list page data: %w", err)
	}

	var items []*supplierlifecycleeventpb.SupplierLifecycleEvent
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_lifecycle_event row: %v", err)
			continue
		}
		sle := &supplierlifecycleeventpb.SupplierLifecycleEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sle); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_lifecycle_event: %v", err)
			continue
		}
		items = append(items, sle)
	}

	totalCount := int64(len(items))
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &supplierlifecycleeventpb.GetSupplierLifecycleEventListPageDataResponse{
		SupplierLifecycleEventList: items,
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

// GetSupplierLifecycleEventItemPageData retrieves a single supplier lifecycle event.
func (r *PostgresSupplierLifecycleEventRepository) GetSupplierLifecycleEventItemPageData(
	ctx context.Context,
	req *supplierlifecycleeventpb.GetSupplierLifecycleEventItemPageDataRequest,
) (*supplierlifecycleeventpb.GetSupplierLifecycleEventItemPageDataResponse, error) {
	if req == nil || req.GetSupplierLifecycleEventId() == "" {
		return nil, fmt.Errorf("supplier lifecycle event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierLifecycleEventId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_lifecycle_event item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sle := &supplierlifecycleeventpb.SupplierLifecycleEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierlifecycleeventpb.GetSupplierLifecycleEventItemPageDataResponse{
		SupplierLifecycleEvent: sle,
		Success:                true,
	}, nil
}
