//go:build postgresql

// Package treasury — postgres adapter for the
// treasury_disbursement_supplier_billing_event junction entity.
//
// This is the buying-side mirror of treasury_collection_billing_event: it links
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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/treasury_disbursement_supplier_billing_event"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryDisbursementSupplierBillingEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres treasury_disbursement_supplier_billing_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTreasuryDisbursementSupplierBillingEventRepository(dbOps, tableName), nil
	})
}

// PostgresTreasuryDisbursementSupplierBillingEventRepository implements
// junction CRUD using PostgreSQL.
type PostgresTreasuryDisbursementSupplierBillingEventRepository struct {
	pb.UnimplementedTreasuryDisbursementSupplierBillingEventDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTreasuryDisbursementSupplierBillingEventRepository creates a new repository.
func NewPostgresTreasuryDisbursementSupplierBillingEventRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TreasuryDisbursementSupplierBillingEventDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TreasuryDisbursementSupplierBillingEvent
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresTreasuryDisbursementSupplierBillingEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateTreasuryDisbursementSupplierBillingEvent inserts a new junction row.
func (r *PostgresTreasuryDisbursementSupplierBillingEventRepository) CreateTreasuryDisbursementSupplierBillingEvent(ctx context.Context, req *pb.CreateTreasuryDisbursementSupplierBillingEventRequest) (*pb.CreateTreasuryDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("treasury_disbursement_supplier_billing_event data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_disbursement_supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_disbursement_supplier_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create treasury_disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.TreasuryDisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.CreateTreasuryDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryDisbursementSupplierBillingEvent{row},
	}, nil
}

// ReadTreasuryDisbursementSupplierBillingEvent retrieves a junction row by ID.
func (r *PostgresTreasuryDisbursementSupplierBillingEventRepository) ReadTreasuryDisbursementSupplierBillingEvent(ctx context.Context, req *pb.ReadTreasuryDisbursementSupplierBillingEventRequest) (*pb.ReadTreasuryDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_disbursement_supplier_billing_event ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read treasury_disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.TreasuryDisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.ReadTreasuryDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryDisbursementSupplierBillingEvent{row},
	}, nil
}

// UpdateTreasuryDisbursementSupplierBillingEvent updates a junction row.
func (r *PostgresTreasuryDisbursementSupplierBillingEventRepository) UpdateTreasuryDisbursementSupplierBillingEvent(ctx context.Context, req *pb.UpdateTreasuryDisbursementSupplierBillingEventRequest) (*pb.UpdateTreasuryDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_disbursement_supplier_billing_event ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_disbursement_supplier_billing_event to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_disbursement_supplier_billing_event JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update treasury_disbursement_supplier_billing_event: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treasury_disbursement_supplier_billing_event result: %w", err)
	}
	row := &pb.TreasuryDisbursementSupplierBillingEvent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal treasury_disbursement_supplier_billing_event proto: %w", err)
	}
	return &pb.UpdateTreasuryDisbursementSupplierBillingEventResponse{
		Success: true,
		Data:    []*pb.TreasuryDisbursementSupplierBillingEvent{row},
	}, nil
}

// DeleteTreasuryDisbursementSupplierBillingEvent soft-deletes a junction row.
func (r *PostgresTreasuryDisbursementSupplierBillingEventRepository) DeleteTreasuryDisbursementSupplierBillingEvent(ctx context.Context, req *pb.DeleteTreasuryDisbursementSupplierBillingEventRequest) (*pb.DeleteTreasuryDisbursementSupplierBillingEventResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("treasury_disbursement_supplier_billing_event ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete treasury_disbursement_supplier_billing_event: %w", err)
	}
	return &pb.DeleteTreasuryDisbursementSupplierBillingEventResponse{Success: true}, nil
}

// ListTreasuryDisbursementSupplierBillingEvents lists junction rows with optional filters.
func (r *PostgresTreasuryDisbursementSupplierBillingEventRepository) ListTreasuryDisbursementSupplierBillingEvents(ctx context.Context, req *pb.ListTreasuryDisbursementSupplierBillingEventsRequest) (*pb.ListTreasuryDisbursementSupplierBillingEventsResponse, error) {
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
		return nil, fmt.Errorf("failed to list treasury_disbursement_supplier_billing_events: %w", err)
	}
	rows := make([]*pb.TreasuryDisbursementSupplierBillingEvent, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			log.Printf("WARN: marshal treasury_disbursement_supplier_billing_event row: %v", err)
			continue
		}
		row := &pb.TreasuryDisbursementSupplierBillingEvent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, row); err != nil {
			log.Printf("WARN: unmarshal treasury_disbursement_supplier_billing_event row: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &pb.ListTreasuryDisbursementSupplierBillingEventsResponse{
		Success: true,
		Data:    rows,
	}, nil
}
