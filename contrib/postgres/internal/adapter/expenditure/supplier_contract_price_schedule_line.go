//go:build postgresql

package expenditure

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
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierContractPriceScheduleLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_contract_price_schedule_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierContractPriceScheduleLineRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierContractPriceScheduleLineRepository implements per-line per-window
// pricing CRUD using PostgreSQL.
type PostgresSupplierContractPriceScheduleLineRepository struct {
	scpslpb.UnimplementedSupplierContractPriceScheduleLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierContractPriceScheduleLineRepository creates a new PostgreSQL
// supplier contract price schedule line repository.
func NewPostgresSupplierContractPriceScheduleLineRepository(dbOps interfaces.DatabaseOperation, tableName string) scpslpb.SupplierContractPriceScheduleLineDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_price_schedule_line"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierContractPriceScheduleLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSupplierContractPriceScheduleLine creates a new schedule-line row.
func (r *PostgresSupplierContractPriceScheduleLineRepository) CreateSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.CreateSupplierContractPriceScheduleLineRequest) (*scpslpb.CreateSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract price schedule line data is required")
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
		return nil, fmt.Errorf("failed to create supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpslpb.CreateSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

// ReadSupplierContractPriceScheduleLine retrieves a schedule-line by ID.
func (r *PostgresSupplierContractPriceScheduleLineRepository) ReadSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.ReadSupplierContractPriceScheduleLineRequest) (*scpslpb.ReadSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpslpb.ReadSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

// UpdateSupplierContractPriceScheduleLine updates a schedule-line row.
func (r *PostgresSupplierContractPriceScheduleLineRepository) UpdateSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.UpdateSupplierContractPriceScheduleLineRequest) (*scpslpb.UpdateSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpslpb.UpdateSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

// DeleteSupplierContractPriceScheduleLine soft-deletes a schedule-line row.
func (r *PostgresSupplierContractPriceScheduleLineRepository) DeleteSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.DeleteSupplierContractPriceScheduleLineRequest) (*scpslpb.DeleteSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_price_schedule_line: %w", err)
	}
	return &scpslpb.DeleteSupplierContractPriceScheduleLineResponse{Success: true}, nil
}

// ListSupplierContractPriceScheduleLines lists schedule-lines with optional filters.
func (r *PostgresSupplierContractPriceScheduleLineRepository) ListSupplierContractPriceScheduleLines(ctx context.Context, req *scpslpb.ListSupplierContractPriceScheduleLinesRequest) (*scpslpb.ListSupplierContractPriceScheduleLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_contract_price_schedule_lines: %w", err)
	}
	var rows []*scpslpb.SupplierContractPriceScheduleLine
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_contract_price_schedule_line row: %v", err)
			continue
		}
		row := &scpslpb.SupplierContractPriceScheduleLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_contract_price_schedule_line: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &scpslpb.ListSupplierContractPriceScheduleLinesResponse{Success: true, Data: rows}, nil
}

// GetSupplierContractPriceScheduleLineListPageData returns a paginated list page (basic CRUD form).
func (r *PostgresSupplierContractPriceScheduleLineRepository) GetSupplierContractPriceScheduleLineListPageData(ctx context.Context, req *scpslpb.GetSupplierContractPriceScheduleLineListPageDataRequest) (*scpslpb.GetSupplierContractPriceScheduleLineListPageDataResponse, error) {
	listResp, err := r.ListSupplierContractPriceScheduleLines(ctx, &scpslpb.ListSupplierContractPriceScheduleLinesRequest{Filters: req.GetFilters(), Pagination: req.GetPagination(), Sort: req.GetSort()})
	if err != nil {
		return nil, err
	}
	return &scpslpb.GetSupplierContractPriceScheduleLineListPageDataResponse{
		SupplierContractPriceScheduleLineList: listResp.Data,
		Success:                               true,
	}, nil
}

// GetSupplierContractPriceScheduleLineItemPageData returns a single schedule-line.
func (r *PostgresSupplierContractPriceScheduleLineRepository) GetSupplierContractPriceScheduleLineItemPageData(ctx context.Context, req *scpslpb.GetSupplierContractPriceScheduleLineItemPageDataRequest) (*scpslpb.GetSupplierContractPriceScheduleLineItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleLineId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractPriceScheduleLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule_line item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpslpb.GetSupplierContractPriceScheduleLineItemPageDataResponse{
		SupplierContractPriceScheduleLine: row,
		Success:                           true,
	}, nil
}

// ResolveActiveScheduleLine returns the unit_price (in centavos) and full schedule-line
// row matching the given supplier_contract_line_id at the supplied asOf time.
//
// Algorithm:
//
//  1. Find the active schedule for the contract via a JOIN onto
//     supplier_contract_price_schedule (using the half-open window window
//     [date_time_start, date_time_end) and excluding cancelled rows).
//  2. JOIN that schedule to supplier_contract_price_schedule_line on
//     (schedule_id, supplier_contract_line_id).
//  3. If a row exists, return it; otherwise return nil (caller falls back
//     to SupplierContractLine.unit_price per Model D precedence).
//
// This is the cross-callable resolver from F12 — used by the recurrence engine
// (P5'), procurement spawn helpers, and the expense recognition use case.
func (r *PostgresSupplierContractPriceScheduleLineRepository) ResolveActiveScheduleLine(ctx context.Context, supplierContractLineID string, asOf time.Time) (*scpslpb.SupplierContractPriceScheduleLine, error) {
	if supplierContractLineID == "" {
		return nil, fmt.Errorf("supplier_contract_line_id is required")
	}
	const cancelledStatus = 4 // SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED
	const activeStatus = 2    // SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE

	query := `
		SELECT scpsl.id
		FROM supplier_contract_price_schedule_line scpsl
		JOIN supplier_contract_price_schedule scps
		  ON scps.id = scpsl.supplier_contract_price_schedule_id
		 AND scps.active = true
		 AND scps.status <> $1
		 AND scps.date_time_start <= $2
		 AND ($2 < scps.date_time_end OR scps.date_time_end IS NULL)
		WHERE scpsl.supplier_contract_line_id = $3
		  AND scpsl.active = true
		ORDER BY (scps.status = $4) DESC, scps.date_time_start DESC
		LIMIT 1
	`
	var id string
	if err := r.db.QueryRowContext(ctx, query, cancelledStatus, asOf, supplierContractLineID, activeStatus).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve active schedule line: %w", err)
	}
	resp, err := r.ReadSupplierContractPriceScheduleLine(ctx, &scpslpb.ReadSupplierContractPriceScheduleLineRequest{
		Data: &scpslpb.SupplierContractPriceScheduleLine{Id: id},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return resp.Data[0], nil
}
