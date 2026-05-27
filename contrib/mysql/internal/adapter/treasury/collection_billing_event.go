//go:build mysql

// Package treasury — MySQL 8.0+ adapter for the collection_billing_event
// junction entity.
//
// collection_billing_event links one MILESTONE TreasuryCollection
// (advance_kind = MILESTONE) to one or more BillingEvent rows; the tranche
// amount on this junction is the portion of the advance assigned to that
// milestone. `revenue_id` is set when the junction is consumed by the
// `recognize_milestone_advance` use case (Phase 7 of the
// 20260517-advance-cash-events plan).
//
// Dialect translation from postgres gold standard
// (docs/plan/20260527-multi-dialect-adapter-alignment/brief.md):
//   - $N → ? (positional, re-sequenced)
//   - "ident" → `ident` (backtick quoting)
//   - ILIKE → LIKE (MySQL ci collation)
//   - FILTER (WHERE c) → SUM(CASE WHEN c THEN expr END)
//   - COUNT(*) OVER () stays (MySQL 8.0+ window functions)
//   - RETURNING → app-side UUID + SELECT after insert
//
// Schema constraint reminder (application-layer, not DB):
//
//	SUM(tranche_amount) over all junctions for a given treasury_collection_id
//	must be <= treasury_collection.amount. The adapter does not enforce this —
//	the use-case layer is responsible.
package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.CollectionBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql collection_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLCollectionBillingEventRepository(dbOps, tableName), nil
	})
}

// MySQLCollectionBillingEventRepository implements junction CRUD using MySQL 8.0+.
type MySQLCollectionBillingEventRepository struct {
	pb.UnimplementedCollectionBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLCollectionBillingEventRepository creates a new repository.
func NewMySQLCollectionBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.CollectionBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.CollectionBillingEvent
	}
	return &MySQLCollectionBillingEventRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateCollectionBillingEvent inserts a new junction row.
func (r *MySQLCollectionBillingEventRepository) CreateCollectionBillingEvent(ctx context.Context, req *pb.CreateCollectionBillingEventRequest) (*pb.CreateCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("collection_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_billing_event result: %w", err)
	}
	row := &pb.CollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_billing_event proto: %w", err)
	}
	return &pb.CreateCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.CollectionBillingEvent{row},
	}, nil
}

// ReadCollectionBillingEvent retrieves a junction row by ID.
func (r *MySQLCollectionBillingEventRepository) ReadCollectionBillingEvent(ctx context.Context, req *pb.ReadCollectionBillingEventRequest) (*pb.ReadCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_billing_event result: %w", err)
	}
	row := &pb.CollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_billing_event proto: %w", err)
	}
	return &pb.ReadCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.CollectionBillingEvent{row},
	}, nil
}

// UpdateCollectionBillingEvent updates a junction row.
func (r *MySQLCollectionBillingEventRepository) UpdateCollectionBillingEvent(ctx context.Context, req *pb.UpdateCollectionBillingEventRequest) (*pb.UpdateCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_billing_event result: %w", err)
	}
	row := &pb.CollectionBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_billing_event proto: %w", err)
	}
	return &pb.UpdateCollectionBillingEventResponse{
		Success: true,
		Data:    []*pb.CollectionBillingEvent{row},
	}, nil
}

// DeleteCollectionBillingEvent soft-deletes a junction row.
func (r *MySQLCollectionBillingEventRepository) DeleteCollectionBillingEvent(ctx context.Context, req *pb.DeleteCollectionBillingEventRequest) (*pb.DeleteCollectionBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection_billing_event: %w", err)
	}
	return &pb.DeleteCollectionBillingEventResponse{Success: true}, nil
}

// ListCollectionBillingEvents lists junction rows with optional filters.
func (r *MySQLCollectionBillingEventRepository) ListCollectionBillingEvents(ctx context.Context, req *pb.ListCollectionBillingEventsRequest) (*pb.ListCollectionBillingEventsResponse, error) {
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
		return nil, fmt.Errorf("failed to list collection_billing_events: %w", err)
	}
	rows := make([]*pb.CollectionBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			log.Printf("WARN: marshal collection_billing_event row: %v", err)
			continue
		}
		row := &pb.CollectionBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, row); err != nil {
			log.Printf("WARN: unmarshal collection_billing_event row: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &pb.ListCollectionBillingEventsResponse{
		Success: true,
		Data:    rows,
	}, nil
}
