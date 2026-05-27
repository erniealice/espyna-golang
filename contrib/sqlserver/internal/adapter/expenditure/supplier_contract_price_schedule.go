//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SupplierContractPriceSchedule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver supplier_contract_price_schedule repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSupplierContractPriceScheduleRepository(db, dbOps, tableName), nil
	})
}

// SQLServerSupplierContractPriceScheduleRepository implements supplier contract price schedule
// CRUD using SQL Server.
type SQLServerSupplierContractPriceScheduleRepository struct {
	scpspb.UnimplementedSupplierContractPriceScheduleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerSupplierContractPriceScheduleRepository creates a new SQL Server repository.
func NewSQLServerSupplierContractPriceScheduleRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) scpspb.SupplierContractPriceScheduleDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_price_schedule"
	}
	return &SQLServerSupplierContractPriceScheduleRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerSupplierContractPriceScheduleRepository) CreateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.CreateSupplierContractPriceScheduleRequest) (*scpspb.CreateSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract price schedule data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	if v, ok := data["status"].(string); ok {
		if num, ok := scpspb.SupplierContractPriceScheduleStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_contract_price_schedule: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpspb.CreateSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleRepository) ReadSupplierContractPriceSchedule(ctx context.Context, req *scpspb.ReadSupplierContractPriceScheduleRequest) (*scpspb.ReadSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpspb.ReadSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleRepository) UpdateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.UpdateSupplierContractPriceScheduleRequest) (*scpspb.UpdateSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	if v, ok := data["status"].(string); ok {
		if num, ok := scpspb.SupplierContractPriceScheduleStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_contract_price_schedule: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpspb.UpdateSupplierContractPriceScheduleResponse{Success: true, Data: []*scpspb.SupplierContractPriceSchedule{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleRepository) DeleteSupplierContractPriceSchedule(ctx context.Context, req *scpspb.DeleteSupplierContractPriceScheduleRequest) (*scpspb.DeleteSupplierContractPriceScheduleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_price_schedule: %w", err)
	}
	return &scpspb.DeleteSupplierContractPriceScheduleResponse{Success: true}, nil
}

func (r *SQLServerSupplierContractPriceScheduleRepository) ListSupplierContractPriceSchedules(ctx context.Context, req *scpspb.ListSupplierContractPriceSchedulesRequest) (*scpspb.ListSupplierContractPriceSchedulesResponse, error) {
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
		sqlserverCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// GetSupplierContractPriceScheduleListPageData returns a paginated list page.
func (r *SQLServerSupplierContractPriceScheduleRepository) GetSupplierContractPriceScheduleListPageData(ctx context.Context, req *scpspb.GetSupplierContractPriceScheduleListPageDataRequest) (*scpspb.GetSupplierContractPriceScheduleListPageDataResponse, error) {
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
func (r *SQLServerSupplierContractPriceScheduleRepository) GetSupplierContractPriceScheduleItemPageData(ctx context.Context, req *scpspb.GetSupplierContractPriceScheduleItemPageDataRequest) (*scpspb.GetSupplierContractPriceScheduleItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractPriceScheduleId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule item: %w", err)
	}
	sqlserverCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpspb.SupplierContractPriceSchedule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpspb.GetSupplierContractPriceScheduleItemPageDataResponse{SupplierContractPriceSchedule: row, Success: true}, nil
}

// ActivateSupplierContractPriceSchedule transitions a SCHEDULED row to ACTIVE.
func (r *SQLServerSupplierContractPriceScheduleRepository) ActivateSupplierContractPriceSchedule(ctx context.Context, req *scpspb.ActivateSupplierContractPriceScheduleRequest) (*scpspb.ActivateSupplierContractPriceScheduleResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	newStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE)
	if _, err := r.db.ExecContext(ctx,
		`UPDATE [supplier_contract_price_schedule]
		 SET [status] = @p1, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p2 AND [active] = 1`,
		newStatus, req.GetSupplierContractPriceScheduleId(),
	); err != nil {
		return nil, fmt.Errorf("failed to activate supplier_contract_price_schedule: %w", err)
	}
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
func (r *SQLServerSupplierContractPriceScheduleRepository) SupersedeSupplierContractPriceSchedule(ctx context.Context, req *scpspb.SupersedeSupplierContractPriceScheduleRequest) (*scpspb.SupersedeSupplierContractPriceScheduleResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule ID is required")
	}
	newStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_SUPERSEDED)
	if _, err := r.db.ExecContext(ctx,
		`UPDATE [supplier_contract_price_schedule]
		 SET [status] = @p1, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p2 AND [active] = 1`,
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

// GetActiveAsOf returns the schedule active for the given supplier_contract_id at asOf.
// SQL Server translation: uses @pN parameters; active = 1; ORDER BY (CASE WHEN … END) DESC.
func (r *SQLServerSupplierContractPriceScheduleRepository) GetActiveAsOf(ctx context.Context, supplierContractID string, asOf time.Time) (*scpspb.SupplierContractPriceSchedule, error) {
	if supplierContractID == "" {
		return nil, fmt.Errorf("supplier_contract_id is required")
	}
	cancelledStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED)
	activeStatus := int32(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE)

	query := `
		SELECT TOP 1 [id]
		FROM [supplier_contract_price_schedule]
		WHERE [supplier_contract_id] = @p1
		  AND [active] = 1
		  AND [status] <> @p2
		  AND [date_time_start] <= @p3
		  AND (@p3 < [date_time_end] OR [date_time_end] IS NULL)
		ORDER BY (CASE WHEN [status] = @p4 THEN 1 ELSE 0 END) DESC, [date_time_start] DESC
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
