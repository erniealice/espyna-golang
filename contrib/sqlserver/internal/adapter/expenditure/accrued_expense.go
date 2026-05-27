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
	registry.RegisterRepositoryFactory("sqlserver", entityid.AccruedExpense, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver accrued_expense repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAccruedExpenseRepository(dbOps, tableName), nil
	})
}

// SQLServerAccruedExpenseRepository implements accrued_expense CRUD using SQL Server.
type SQLServerAccruedExpenseRepository struct {
	accruedexpensepb.UnimplementedAccruedExpenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewSQLServerAccruedExpenseRepository(dbOps interfaces.DatabaseOperation, tableName string) accruedexpensepb.AccruedExpenseDomainServiceServer {
	if tableName == "" {
		tableName = "accrued_expense"
	}
	return &SQLServerAccruedExpenseRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerAccruedExpenseRepository) CreateAccruedExpense(ctx context.Context, req *accruedexpensepb.CreateAccruedExpenseRequest) (*accruedexpensepb.CreateAccruedExpenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("accrued expense data is required")
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
		return nil, fmt.Errorf("failed to create accrued expense: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	ae := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ae); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.CreateAccruedExpenseResponse{Data: []*accruedexpensepb.AccruedExpense{ae}}, nil
}

func (r *SQLServerAccruedExpenseRepository) ReadAccruedExpense(ctx context.Context, req *accruedexpensepb.ReadAccruedExpenseRequest) (*accruedexpensepb.ReadAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued expense: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	ae := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ae); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.ReadAccruedExpenseResponse{Data: []*accruedexpensepb.AccruedExpense{ae}}, nil
}

func (r *SQLServerAccruedExpenseRepository) UpdateAccruedExpense(ctx context.Context, req *accruedexpensepb.UpdateAccruedExpenseRequest) (*accruedexpensepb.UpdateAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
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
		return nil, fmt.Errorf("failed to update accrued expense: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	ae := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ae); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.UpdateAccruedExpenseResponse{Data: []*accruedexpensepb.AccruedExpense{ae}}, nil
}

func (r *SQLServerAccruedExpenseRepository) DeleteAccruedExpense(ctx context.Context, req *accruedexpensepb.DeleteAccruedExpenseRequest) (*accruedexpensepb.DeleteAccruedExpenseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete accrued expense: %w", err)
	}
	return &accruedexpensepb.DeleteAccruedExpenseResponse{Success: true}, nil
}

func (r *SQLServerAccruedExpenseRepository) ListAccruedExpenses(ctx context.Context, req *accruedexpensepb.ListAccruedExpensesRequest) (*accruedexpensepb.ListAccruedExpensesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accrued expenses: %w", err)
	}
	var items []*accruedexpensepb.AccruedExpense
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal accrued_expense row: %v", err)
			continue
		}
		ae := &accruedexpensepb.AccruedExpense{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ae); err != nil {
			log.Printf("WARN: protojson unmarshal accrued_expense: %v", err)
			continue
		}
		items = append(items, ae)
	}
	return &accruedexpensepb.ListAccruedExpensesResponse{Data: items}, nil
}

// GetAccruedExpenseListPageData — TODO: translate CTE query from postgres gold standard.
func (r *SQLServerAccruedExpenseRepository) GetAccruedExpenseListPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseListPageDataRequest) (*accruedexpensepb.GetAccruedExpenseListPageDataResponse, error) {
	// TODO: SQL Server CTE translation — delegate to ListAccruedExpenses for now.
	listResp, err := r.ListAccruedExpenses(ctx, &accruedexpensepb.ListAccruedExpensesRequest{})
	if err != nil {
		return nil, err
	}
	return &accruedexpensepb.GetAccruedExpenseListPageDataResponse{AccruedExpenseList: listResp.Data, Success: true}, nil
}

// GetAccruedExpenseItemPageData — TODO: translate CTE query from postgres gold standard.
func (r *SQLServerAccruedExpenseRepository) GetAccruedExpenseItemPageData(ctx context.Context, req *accruedexpensepb.GetAccruedExpenseItemPageDataRequest) (*accruedexpensepb.GetAccruedExpenseItemPageDataResponse, error) {
	if req == nil || req.AccruedExpenseId == "" {
		return nil, fmt.Errorf("accrued expense ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.AccruedExpenseId)
	if err != nil {
		return nil, fmt.Errorf("failed to read accrued expense: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	ae := &accruedexpensepb.AccruedExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ae); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &accruedexpensepb.GetAccruedExpenseItemPageDataResponse{AccruedExpense: ae, Success: true}, nil
}

// ListOutstanding returns outstanding accrued expenses for a workspace.
// TODO: translate raw SQL from postgres gold standard (uses workspace_id predicate).
func (r *SQLServerAccruedExpenseRepository) ListOutstanding(ctx context.Context, workspaceID string) ([]*accruedexpensepb.AccruedExpense, error) {
	// TODO: implement direct SQL with @p1 workspace predicate.
	_ = workspaceID
	return nil, fmt.Errorf("ListOutstanding: TODO — translate raw SQL for SQL Server")
}
