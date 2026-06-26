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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierContractPriceSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_contract_price_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierContractPriceScheduleRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierContractPriceScheduleRepository implements supplier contract price schedule
// CRUD operations using PostgreSQL.
//
// Window discipline: schedules are stored as half-open `[date_time_start, date_time_end)`
// ranges and DB-enforced via the GIST exclusion constraint added in migration
// 20260430140000. Use-case-level overlap validation is a defense-in-depth layer.
//
// At most one row per supplier_contract_id may carry status=ACTIVE
// (partial unique index `supplier_contract_price_schedule_one_active_per_contract`).
type PostgresSupplierContractPriceScheduleRepository struct {
	scpspb.UnimplementedSupplierContractPriceScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierContractPriceScheduleRepository creates a new PostgreSQL
// supplier contract price schedule repository.
func NewPostgresSupplierContractPriceScheduleRepository(dbOps interfaces.DatabaseOperation, tableName string) scpspb.SupplierContractPriceScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_price_schedule"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierContractPriceScheduleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSupplierContractPriceSchedule creates a new schedule row.
func (r *PostgresSupplierContractPriceScheduleRepository) CreateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.CreateSupplierContractPriceScheduleRequest) (*scpspb.CreateSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract price schedule data is required")
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
	// status is an INTEGER column; protojson serialised the enum as its string
	// name. Convert back to the proto enum int so the INSERT typechecks.
	if v, ok := data["status"].(string); ok {
		if num, ok := scpspb.SupplierContractPriceScheduleStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_contract_price_schedule: %w", err)
	}
	// TIMESTAMPTZ columns come back as int64 millis from scanRowToMap; protojson
	// expects RFC3339 strings for google.protobuf.Timestamp.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpspb.CreateSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

// ReadSupplierContractPriceSchedule retrieves a schedule by ID.
func (r *PostgresSupplierContractPriceScheduleRepository) ReadSupplierContractPriceSchedule(ctx context.Context, req *scpspb.ReadSupplierContractPriceScheduleRequest) (*scpspb.ReadSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpspb.ReadSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

// UpdateSupplierContractPriceSchedule updates a schedule row.
func (r *PostgresSupplierContractPriceScheduleRepository) UpdateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.UpdateSupplierContractPriceScheduleRequest) (*scpspb.UpdateSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
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
	// status: protojson emits the enum as a string; the column is INTEGER.
	if v, ok := data["status"].(string); ok {
		if num, ok := scpspb.SupplierContractPriceScheduleStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_contract_price_schedule: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpspb.UpdateSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

// DeleteSupplierContractPriceSchedule soft-deletes a schedule row.
func (r *PostgresSupplierContractPriceScheduleRepository) DeleteSupplierContractPriceSchedule(ctx context.Context, req *scpspb.DeleteSupplierContractPriceScheduleRequest) (*scpspb.DeleteSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_price_schedule: %w", err)
	}
	return &scpspb.DeleteSupplierContractPriceScheduleResponse{Success: true}, nil
}

// ListSupplierContractPriceSchedules lists schedules with optional filters.
func (r *PostgresSupplierContractPriceScheduleRepository) ListSupplierContractPriceSchedules(ctx context.Context, req *scpspb.ListSupplierContractPriceSchedulesRequest) (*scpspb.ListSupplierContractPriceSchedulesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_contract_price_schedules: %w", err)
	}
	var rows []*scpspb.SupplierContractPriceSchedule
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_contract_price_schedule row: %v", err)
			continue
		}
		row := &scpspb.SupplierContractPriceSchedule{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_contract_price_schedule: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &scpspb.ListSupplierContractPriceSchedulesResponse{Success: true, Data: rows}, nil
}

// GetSupplierContractPriceScheduleListPageData returns a paginated list page (basic CRUD form).
func (r *PostgresSupplierContractPriceScheduleRepository) GetSupplierContractPriceScheduleListPageData(ctx context.Context, req *scpspb.GetSupplierContractPriceScheduleListPageDataRequest) (*scpspb.GetSupplierContractPriceScheduleListPageDataResponse, error) {
	listResp, err := r.ListSupplierContractPriceSchedules(ctx, &scpspb.ListSupplierContractPriceSchedulesRequest{Filters: req.GetFilters(), Pagination: req.GetPagination(), Sort: req.GetSort(), Search: req.GetSearch()})
	if err != nil {
		return nil, err
	}
	return &scpspb.GetSupplierContractPriceScheduleListPageDataResponse{
		SupplierContractPriceScheduleList: listResp.Data,
		Success:                           true,
	}, nil
}

// GetSupplierContractPriceScheduleItemPageData returns a single schedule for the detail page.
func (r *PostgresSupplierContractPriceScheduleRepository) GetSupplierContractPriceScheduleItemPageData(ctx context.Context, req *scpspb.GetSupplierContractPriceScheduleItemPageDataRequest) (*scpspb.GetSupplierContractPriceScheduleItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractPriceScheduleId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule item: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &scpspb.GetSupplierContractPriceScheduleItemPageDataResponse{
		SupplierContractPriceSchedule: row,
		Success:                       true,
	}, nil
}

// ActivateSupplierContractPriceSchedule transitions a SCHEDULED row to ACTIVE.
// Caller is expected to ensure the contract has at most one ACTIVE schedule
// at activation time (use cases auto-supersede prior ACTIVE rows in the same
// transaction; the partial unique index serves as the DB-side defense).
func (r *PostgresSupplierContractPriceScheduleRepository) ActivateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.ActivateSupplierContractPriceScheduleRequest) (*scpspb.ActivateSupplierContractPriceScheduleResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	newStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE)
	if _, err := r.db.ExecContext(ctx,
		`UPDATE supplier_contract_price_schedule
		 SET status = $1, date_modified = NOW()
		 WHERE id = $2 AND active = true`,
		newStatus, req.GetSupplierContractPriceScheduleId(),
	); err != nil {
		return nil, fmt.Errorf("failed to activate supplier_contract_price_schedule: %w", err)
	}
	// Fetch updated row to return.
	readResp, err := r.ReadSupplierContractPriceSchedule(ctx, &scpspb.ReadSupplierContractPriceScheduleRequest{Data: &scpspb.SupplierContractPriceSchedule{Id: req.GetSupplierContractPriceScheduleId()}})
	if err != nil {
		return nil, err
	}
	var data *scpspb.SupplierContractPriceSchedule
	if len(readResp.Data) > 0 {
		data = readResp.Data[0]
	}
	return &scpspb.ActivateSupplierContractPriceScheduleResponse{Success: true, Data: data}, nil
}

// SupersedeSupplierContractPriceSchedule transitions an ACTIVE row to SUPERSEDED.
func (r *PostgresSupplierContractPriceScheduleRepository) SupersedeSupplierContractPriceSchedule(ctx context.Context, req *scpspb.SupersedeSupplierContractPriceScheduleRequest) (*scpspb.SupersedeSupplierContractPriceScheduleResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	newStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_SUPERSEDED)
	if _, err := r.db.ExecContext(ctx,
		`UPDATE supplier_contract_price_schedule
		 SET status = $1, date_modified = NOW()
		 WHERE id = $2 AND active = true`,
		newStatus, req.GetSupplierContractPriceScheduleId(),
	); err != nil {
		return nil, fmt.Errorf("failed to supersede supplier_contract_price_schedule: %w", err)
	}
	readResp, err := r.ReadSupplierContractPriceSchedule(ctx, &scpspb.ReadSupplierContractPriceScheduleRequest{Data: &scpspb.SupplierContractPriceSchedule{Id: req.GetSupplierContractPriceScheduleId()}})
	if err != nil {
		return nil, err
	}
	var data *scpspb.SupplierContractPriceSchedule
	if len(readResp.Data) > 0 {
		data = readResp.Data[0]
	}
	return &scpspb.SupersedeSupplierContractPriceScheduleResponse{Success: true, Data: data}, nil
}

// GetActiveAsOf returns the schedule that is ACTIVE for the given supplier_contract_id
// at the supplied asOf timestamp.
//
// Uses the half-open window [date_time_start, date_time_end). Falls back to the
// open-ended row (date_time_end IS NULL) where applicable. Excludes cancelled rows.
//
// This is a custom (non-proto) cross-callable resolver used by the recurrence
// engine (P5'), the schedule-line resolver, and reporting queries.
func (r *PostgresSupplierContractPriceScheduleRepository) GetActiveAsOf(ctx context.Context, supplierContractID string, asOf time.Time) (*scpspb.SupplierContractPriceSchedule, error) {
	if supplierContractID == "" {
		return nil, fmt.Errorf("supplier_contract_id is required")
	}
	activeStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE)
	cancelledStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED)

	// Prefer status=ACTIVE; fall back to any non-cancelled row whose window
	// contains asOf. The partial unique index guarantees at most one ACTIVE row.
	query := `
		SELECT id
		FROM supplier_contract_price_schedule
		WHERE supplier_contract_id = $1
		  AND active = true
		  AND status <> $2
		  AND date_time_start <= $3
		  AND ($3 < date_time_end OR date_time_end IS NULL)
		ORDER BY (status = $4) DESC, date_time_start DESC
		LIMIT 1
	`
	var id string
	if err := r.db.QueryRowContext(ctx, query, supplierContractID, cancelledStatus, asOf, activeStatus).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve active schedule: %w", err)
	}
	resp, err := r.ReadSupplierContractPriceSchedule(ctx, &scpspb.ReadSupplierContractPriceScheduleRequest{
		Data: &scpspb.SupplierContractPriceSchedule{Id: id},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return resp.Data[0], nil
}
