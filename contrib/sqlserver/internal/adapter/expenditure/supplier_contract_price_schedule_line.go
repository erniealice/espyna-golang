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
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SupplierContractPriceScheduleLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver supplier_contract_price_schedule_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSupplierContractPriceScheduleLineRepository(db, dbOps, tableName), nil
	})
}

// SQLServerSupplierContractPriceScheduleLineRepository implements per-line pricing CRUD using SQL Server.
type SQLServerSupplierContractPriceScheduleLineRepository struct {
	scpslpb.UnimplementedSupplierContractPriceScheduleLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerSupplierContractPriceScheduleLineRepository creates a new SQL Server repository.
func NewSQLServerSupplierContractPriceScheduleLineRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) scpslpb.SupplierContractPriceScheduleLineDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_price_schedule_line"
	}
	return &SQLServerSupplierContractPriceScheduleLineRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerSupplierContractPriceScheduleLineRepository) CreateSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.CreateSupplierContractPriceScheduleLineRequest) (*scpslpb.CreateSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract price schedule line data is required")
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpslpb.CreateSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleLineRepository) ReadSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.ReadSupplierContractPriceScheduleLineRequest) (*scpslpb.ReadSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpslpb.ReadSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleLineRepository) UpdateSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.UpdateSupplierContractPriceScheduleLineRequest) (*scpslpb.UpdateSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier_contract_price_schedule_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpslpb.UpdateSupplierContractPriceScheduleLineResponse{Success: true, Data: []*scpslpb.SupplierContractPriceScheduleLine{row}}, nil
}

func (r *SQLServerSupplierContractPriceScheduleLineRepository) DeleteSupplierContractPriceScheduleLine(ctx context.Context, req *scpslpb.DeleteSupplierContractPriceScheduleLineRequest) (*scpslpb.DeleteSupplierContractPriceScheduleLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_price_schedule_line: %w", err)
	}
	return &scpslpb.DeleteSupplierContractPriceScheduleLineResponse{Success: true}, nil
}

func (r *SQLServerSupplierContractPriceScheduleLineRepository) ListSupplierContractPriceScheduleLines(ctx context.Context, req *scpslpb.ListSupplierContractPriceScheduleLinesRequest) (*scpslpb.ListSupplierContractPriceScheduleLinesResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// GetSupplierContractPriceScheduleLineListPageData returns a paginated list page.
func (r *SQLServerSupplierContractPriceScheduleLineRepository) GetSupplierContractPriceScheduleLineListPageData(ctx context.Context, req *scpslpb.GetSupplierContractPriceScheduleLineListPageDataRequest) (*scpslpb.GetSupplierContractPriceScheduleLineListPageDataResponse, error) {
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
func (r *SQLServerSupplierContractPriceScheduleLineRepository) GetSupplierContractPriceScheduleLineItemPageData(ctx context.Context, req *scpslpb.GetSupplierContractPriceScheduleLineItemPageDataRequest) (*scpslpb.GetSupplierContractPriceScheduleLineItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractPriceScheduleLineId() == "" {
		return nil, fmt.Errorf("supplier contract price schedule line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractPriceScheduleLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_price_schedule_line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	row := &scpslpb.SupplierContractPriceScheduleLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &scpslpb.GetSupplierContractPriceScheduleLineItemPageDataResponse{SupplierContractPriceScheduleLine: row, Success: true}, nil
}

// ResolveActiveScheduleLine returns the schedule-line active for the given supplierContractLineID at asOf.
// SQL Server translation: @pN params, active = 1, ORDER BY CASE WHEN.
func (r *SQLServerSupplierContractPriceScheduleLineRepository) ResolveActiveScheduleLine(ctx context.Context, supplierContractLineID string, asOf time.Time) (*scpslpb.SupplierContractPriceScheduleLine, error) {
	if supplierContractLineID == "" {
		return nil, fmt.Errorf("supplier_contract_line_id is required")
	}
	const cancelledStatus = 4 // SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_CANCELLED
	const activeStatus = 2    // SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE

	query := `
		SELECT TOP 1 [scpsl].[id]
		FROM [supplier_contract_price_schedule_line] [scpsl]
		JOIN [supplier_contract_price_schedule] [scps]
		  ON [scps].[id] = [scpsl].[supplier_contract_price_schedule_id]
		 AND [scps].[active] = 1
		 AND [scps].[status] <> @p1
		 AND [scps].[date_time_start] <= @p2
		 AND (@p2 < [scps].[date_time_end] OR [scps].[date_time_end] IS NULL)
		WHERE [scpsl].[supplier_contract_line_id] = @p3
		  AND [scpsl].[active] = 1
		ORDER BY (CASE WHEN [scps].[status] = @p4 THEN 1 ELSE 0 END) DESC, [scps].[date_time_start] DESC
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
