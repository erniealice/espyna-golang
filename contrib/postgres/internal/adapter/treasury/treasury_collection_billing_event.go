//go:build postgresql

// Package treasury — postgres adapter for the treasury_collection_billing_event
// junction entity.
//
// treasury_collection_billing_event links one MILESTONE TreasuryCollection
// (advance_kind = MILESTONE) to one or more BillingEvent rows; the tranche
// amount on this junction is the portion of the advance assigned to that
// milestone. `revenue_id` is set when the junction is consumed by the
// `recognize_milestone_advance` use case (Phase 7 of the
// 20260517-advance-cash-events plan).
//
// Schema constraint reminder (application-layer, not DB):
//   SUM(tranche_amount) over all junctions for a given treasury_collection_id
//   must be <= treasury_collection.amount. The adapter does not enforce this —
//   the use-case layer is responsible.
package treasury

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/treasury_collection_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryCollectionBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres treasury_collection_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTreasuryCollectionBillingEventRepository(dbOps, tableName), nil
	})
}

// PostgresTreasuryCollectionBillingEventRepository implements junction CRUD using PostgreSQL.
type PostgresTreasuryCollectionBillingEventRepository struct {
	pb.UnimplementedTreasuryCollectionBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTreasuryCollectionBillingEventRepository creates a new repository.
func NewPostgresTreasuryCollectionBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TreasuryCollectionBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TreasuryCollectionBillingEvent
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresTreasuryCollectionBillingEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateTreasuryCollectionBillingEvent inserts a new junction row.
func (r *PostgresTreasuryCollectionBillingEventRepository) CreateTreasuryCollectionBillingEvent(ctx context.Context, req *pb.CreateTreasuryCollectionBillingEventRequest) (*pb.CreateTreasuryCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("treasury_collection_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_collection_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_collection_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create treasury_collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_collection_billing_event result: %w", err)
	}
	row := &pb.TreasuryCollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_collection_billing_event proto: %w", err)
	}
	return &pb.CreateTreasuryCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryCollectionBillingEvent{row},
	}, nil
}

// ReadTreasuryCollectionBillingEvent retrieves a junction row by ID.
func (r *PostgresTreasuryCollectionBillingEventRepository) ReadTreasuryCollectionBillingEvent(ctx context.Context, req *pb.ReadTreasuryCollectionBillingEventRequest) (*pb.ReadTreasuryCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_collection_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read treasury_collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_collection_billing_event result: %w", err)
	}
	row := &pb.TreasuryCollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_collection_billing_event proto: %w", err)
	}
	return &pb.ReadTreasuryCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryCollectionBillingEvent{row},
	}, nil
}

// UpdateTreasuryCollectionBillingEvent updates a junction row.
func (r *PostgresTreasuryCollectionBillingEventRepository) UpdateTreasuryCollectionBillingEvent(ctx context.Context, req *pb.UpdateTreasuryCollectionBillingEventRequest) (*pb.UpdateTreasuryCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_collection_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_collection_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_collection_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update treasury_collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_collection_billing_event result: %w", err)
	}
	row := &pb.TreasuryCollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_collection_billing_event proto: %w", err)
	}
	return &pb.UpdateTreasuryCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryCollectionBillingEvent{row},
	}, nil
}

// DeleteTreasuryCollectionBillingEvent soft-deletes a junction row.
func (r *PostgresTreasuryCollectionBillingEventRepository) DeleteTreasuryCollectionBillingEvent(ctx context.Context, req *pb.DeleteTreasuryCollectionBillingEventRequest) (*pb.DeleteTreasuryCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_collection_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete treasury_collection_billing_event: %w", err)
	}
	return &pb.DeleteTreasuryCollectionBillingEventResponse{Success: true}, nil
}

// ListTreasuryCollectionBillingEvents lists junction rows with optional filters.
func (r *PostgresTreasuryCollectionBillingEventRepository) ListTreasuryCollectionBillingEvents(ctx context.Context, req *pb.ListTreasuryCollectionBillingEventsRequest) (*pb.ListTreasuryCollectionBillingEventsResponse, error) {
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
		return nil, fmt.Errorf("failed to list treasury_collection_billing_events: %w", err)
	}
	rows := make([]*pb.TreasuryCollectionBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			log.Printf("WARN: marshal treasury_collection_billing_event row: %v", err)
			continue
		}
		row := &pb.TreasuryCollectionBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, row); err != nil {
			log.Printf("WARN: unmarshal treasury_collection_billing_event row: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &pb.ListTreasuryCollectionBillingEventsResponse{
		Success: true,
		Data:    rows,
	}, nil
}
