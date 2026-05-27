//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.AccruedExpenseSettlement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver accrued_expense_settlement repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAccruedExpenseSettlementRepository(dbOps, tableName), nil
	})
}

// SQLServerAccruedExpenseSettlementRepository implements accrued_expense_settlement CRUD using SQL Server.
type SQLServerAccruedExpenseSettlementRepository struct {
	accruedexpensepb.UnimplementedAccruedExpenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewSQLServerAccruedExpenseSettlementRepository(dbOps interfaces.DatabaseOperation, tableName string) accruedexpensepb.AccruedExpenseDomainServiceServer {
	if tableName == "" {
		tableName = "accrued_expense_settlement"
	}
	return &SQLServerAccruedExpenseSettlementRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerAccruedExpenseSettlementRepository) CreateAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseSettlementRequest) (*accruedexpensepb.CreateAccruedExpenseSettlementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("accrued expense settlement data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create accrued expense settlement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	s := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.CreateAccruedExpenseSettlementResponse{Data: []*accruedexpensepb.AccruedExpenseSettlement{s}}, nil
}

func (r *SQLServerAccruedExpenseSettlementRepository) ReadAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseSettlementRequest) (*accruedexpensepb.ReadAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued expense settlement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	s := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.ReadAccruedExpenseSettlementResponse{Data: []*accruedexpensepb.AccruedExpenseSettlement{s}}, nil
}

func (r *SQLServerAccruedExpenseSettlementRepository) UpdateAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseSettlementRequest) (*accruedexpensepb.UpdateAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update accrued expense settlement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	s := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.UpdateAccruedExpenseSettlementResponse{Data: []*accruedexpensepb.AccruedExpenseSettlement{s}}, nil
}

func (r *SQLServerAccruedExpenseSettlementRepository) DeleteAccruedExpenseSettlement(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseSettlementRequest) (*accruedexpensepb.DeleteAccruedExpenseSettlementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete accrued expense settlement: %w", err)
	}
	return &accruedexpensepb.DeleteAccruedExpenseSettlementResponse{Success: true}, nil
}

func (r *SQLServerAccruedExpenseSettlementRepository) ListAccruedExpenseSettlements(ctx context.Context, req *accruedexpensepb.ListAccruedExpenseSettlementsRequest) (*accruedexpensepb.ListAccruedExpenseSettlementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accrued expense settlements: %w", err)
	}
	var items []*accruedexpensepb.AccruedExpenseSettlement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal accrued_expense_settlement row: %v", err)
			continue
		}
		s := &accruedexpensepb.AccruedExpenseSettlement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
			log.Printf("WARN: protojson unmarshal accrued_expense_settlement: %v", err)
			continue
		}
		items = append(items, s)
	}
	return &accruedexpensepb.ListAccruedExpenseSettlementsResponse{Data: items}, nil
}

// GetAccruedExpenseSettlementListPageData — TODO: translate CTE query from postgres gold standard.
func (r *SQLServerAccruedExpenseSettlementRepository) GetAccruedExpenseSettlementListPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseSettlementListPageDataRequest) (*accruedexpensepb.GetAccruedExpenseSettlementListPageDataResponse, error) {
	listResp, err := r.ListAccruedExpenseSettlements(ctx, &accruedexpensepb.ListAccruedExpenseSettlementsRequest{})
	if err != nil {
		return nil, err
	}
	return &accruedexpensepb.GetAccruedExpenseSettlementListPageDataResponse{AccruedExpenseSettlementList: listResp.Data, Success: true}, nil
}

// GetAccruedExpenseSettlementItemPageData — TODO: translate CTE query from postgres gold standard.
func (r *SQLServerAccruedExpenseSettlementRepository) GetAccruedExpenseSettlementItemPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseSettlementItemPageDataRequest) (*accruedexpensepb.GetAccruedExpenseSettlementItemPageDataResponse, error) {
	if req == nil || req.AccruedExpenseSettlementId == "" {
		return nil, fmt.Errorf("accrued expense settlement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.AccruedExpenseSettlementId)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued expense settlement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	s := &accruedexpensepb.AccruedExpenseSettlement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.GetAccruedExpenseSettlementItemPageDataResponse{AccruedExpenseSettlement: s, Success: true}, nil
}
