//go:build postgresql

// Package treasury — postgres adapter for the
// disbursement_supplier_billing_event junction entity.
//
// This is the buying-side mirror of collection_billing_event: it links
// one MILESTONE TreasuryDisbursement (advance_kind = MILESTONE) to one or more
// SupplierBillingEvent rows. `expense_recognition_id` is set when the junction
// is consumed by the `recognize_milestone_advance` use case (buying-side
// counterpart, Phase 7 of the 20260517-advance-cash-events plan).
//
// Schema constraint reminder (application-layer, not DB):
//   SUM(tranche_amount) over all junctions for a given treasury_disbursement_id
//   must be <= treasury_disbursement.amount.
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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_supplier_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DisbursementSupplierBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres disbursement_supplier_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresDisbursementSupplierBillingEventRepository(dbOps, tableName), nil
	})
}

// PostgresDisbursementSupplierBillingEventRepository implements
// junction CRUD using PostgreSQL.
type PostgresDisbursementSupplierBillingEventRepository struct {
	pb.UnimplementedDisbursementSupplierBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresDisbursementSupplierBillingEventRepository creates a new repository.
func NewPostgresDisbursementSupplierBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.DisbursementSupplierBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.DisbursementSupplierBillingEvent
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresDisbursementSupplierBillingEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDisbursementSupplierBillingEvent inserts a new junction row.
func (r *PostgresDisbursementSupplierBillingEventRepository) CreateDisbursementSupplierBillingEvent(ctx context.Context, req *pb.CreateDisbursementSupplierBillingEventRequest) (*pb.CreateDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("disbursement_supplier_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_supplier_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.DisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.CreateDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.DisbursementSupplierBillingEvent{row},
	}, nil
}

// ReadDisbursementSupplierBillingEvent retrieves a junction row by ID.
func (r *PostgresDisbursementSupplierBillingEventRepository) ReadDisbursementSupplierBillingEvent(ctx context.Context, req *pb.ReadDisbursementSupplierBillingEventRequest) (*pb.ReadDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.DisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.ReadDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.DisbursementSupplierBillingEvent{row},
	}, nil
}

// UpdateDisbursementSupplierBillingEvent updates a junction row.
func (r *PostgresDisbursementSupplierBillingEventRepository) UpdateDisbursementSupplierBillingEvent(ctx context.Context, req *pb.UpdateDisbursementSupplierBillingEventRequest) (*pb.UpdateDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_supplier_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_supplier_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.DisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.UpdateDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.DisbursementSupplierBillingEvent{row},
	}, nil
}

// DeleteDisbursementSupplierBillingEvent soft-deletes a junction row.
func (r *PostgresDisbursementSupplierBillingEventRepository) DeleteDisbursementSupplierBillingEvent(ctx context.Context, req *pb.DeleteDisbursementSupplierBillingEventRequest) (*pb.DeleteDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement_supplier_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete disbursement_supplier_billing_event: %w", err)
	}
	return &pb.DeleteDisbursementSupplierBillingEventResponse{Success: true}, nil
}

// ListDisbursementSupplierBillingEvents lists junction rows with optional filters.
func (r *PostgresDisbursementSupplierBillingEventRepository) ListDisbursementSupplierBillingEvents(ctx context.Context, req *pb.ListDisbursementSupplierBillingEventsRequest) (*pb.ListDisbursementSupplierBillingEventsResponse, error) {
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
		return nil, fmt.Errorf("failed to list disbursement_supplier_billing_events: %w", err)
	}
	rows := make([]*pb.DisbursementSupplierBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			log.Printf("WARN: marshal disbursement_supplier_billing_event row: %v", err)
			continue
		}
		row := &pb.DisbursementSupplierBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, row); err != nil {
			log.Printf("WARN: unmarshal disbursement_supplier_billing_event row: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &pb.ListDisbursementSupplierBillingEventsResponse{
		Success: true,
		Data:    rows,
	}, nil
}
