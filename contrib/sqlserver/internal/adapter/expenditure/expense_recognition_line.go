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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenseRecognitionLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expense_recognition_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenseRecognitionLineRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenseRecognitionLineRepository implements expense recognition line
// CRUD using SQL Server.
type SQLServerExpenseRecognitionLineRepository struct {
	expenserecognitionlinepb.UnimplementedExpenseRecognitionLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerExpenseRecognitionLineRepository creates a new SQL Server expense
// recognition line repository.
func NewSQLServerExpenseRecognitionLineRepository(dbOps interfaces.DatabaseOperation, tableName string) expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer {
	if tableName == "" {
		tableName = "expense_recognition_line"
	}
	return &SQLServerExpenseRecognitionLineRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateExpenseRecognitionLine creates a new recognition-line row.
func (r *SQLServerExpenseRecognitionLineRepository) CreateExpenseRecognitionLine(ctx context.Context, req *expenserecognitionlinepb.CreateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.CreateExpenseRecognitionLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expense recognition line data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionlinepb.ExpenseRecognitionLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionlinepb.CreateExpenseRecognitionLineResponse{Success: true, Data: []*expenserecognitionlinepb.ExpenseRecognitionLine{row}}, nil
}

// ReadExpenseRecognitionLine retrieves a recognition-line by ID.
func (r *SQLServerExpenseRecognitionLineRepository) ReadExpenseRecognitionLine(ctx context.Context, req *expenserecognitionlinepb.ReadExpenseRecognitionLineRequest) (*expenserecognitionlinepb.ReadExpenseRecognitionLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionlinepb.ExpenseRecognitionLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionlinepb.ReadExpenseRecognitionLineResponse{Success: true, Data: []*expenserecognitionlinepb.ExpenseRecognitionLine{row}}, nil
}

// UpdateExpenseRecognitionLine updates a recognition-line row.
func (r *SQLServerExpenseRecognitionLineRepository) UpdateExpenseRecognitionLine(ctx context.Context, req *expenserecognitionlinepb.UpdateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.UpdateExpenseRecognitionLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition line ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expense_recognition_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionlinepb.ExpenseRecognitionLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionlinepb.UpdateExpenseRecognitionLineResponse{Success: true, Data: []*expenserecognitionlinepb.ExpenseRecognitionLine{row}}, nil
}

// DeleteExpenseRecognitionLine soft-deletes a recognition-line row.
func (r *SQLServerExpenseRecognitionLineRepository) DeleteExpenseRecognitionLine(ctx context.Context, req *expenserecognitionlinepb.DeleteExpenseRecognitionLineRequest) (*expenserecognitionlinepb.DeleteExpenseRecognitionLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense recognition line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expense_recognition_line: %w", err)
	}
	return &expenserecognitionlinepb.DeleteExpenseRecognitionLineResponse{Success: true}, nil
}

// ListExpenseRecognitionLines lists recognition-lines with optional filters.
func (r *SQLServerExpenseRecognitionLineRepository) ListExpenseRecognitionLines(ctx context.Context, req *expenserecognitionlinepb.ListExpenseRecognitionLinesRequest) (*expenserecognitionlinepb.ListExpenseRecognitionLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense_recognition_lines: %w", err)
	}
	var rows []*expenserecognitionlinepb.ExpenseRecognitionLine
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expense_recognition_line row: %v", err)
			continue
		}
		row := &expenserecognitionlinepb.ExpenseRecognitionLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
			log.Printf("WARN: protojson unmarshal expense_recognition_line: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &expenserecognitionlinepb.ListExpenseRecognitionLinesResponse{Success: true, Data: rows}, nil
}

// GetExpenseRecognitionLineListPageData returns a paginated list page.
func (r *SQLServerExpenseRecognitionLineRepository) GetExpenseRecognitionLineListPageData(ctx context.Context, req *expenserecognitionlinepb.GetExpenseRecognitionLineListPageDataRequest) (*expenserecognitionlinepb.GetExpenseRecognitionLineListPageDataResponse, error) {
	listResp, err := r.ListExpenseRecognitionLines(ctx, &expenserecognitionlinepb.ListExpenseRecognitionLinesRequest{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
		Sort:       req.GetSort(),
	})
	if err != nil {
		return nil, err
	}
	return &expenserecognitionlinepb.GetExpenseRecognitionLineListPageDataResponse{
		ExpenseRecognitionLineList: listResp.Data,
		Success:                    true,
	}, nil
}

// GetExpenseRecognitionLineItemPageData returns a single recognition-line.
func (r *SQLServerExpenseRecognitionLineRepository) GetExpenseRecognitionLineItemPageData(ctx context.Context, req *expenserecognitionlinepb.GetExpenseRecognitionLineItemPageDataRequest) (*expenserecognitionlinepb.GetExpenseRecognitionLineItemPageDataResponse, error) {
	if req == nil || req.GetExpenseRecognitionLineId() == "" {
		return nil, fmt.Errorf("expense recognition line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetExpenseRecognitionLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition_line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	row := &expenserecognitionlinepb.ExpenseRecognitionLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, row); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &expenserecognitionlinepb.GetExpenseRecognitionLineItemPageDataResponse{
		ExpenseRecognitionLine: row,
		Success:                true,
	}, nil
}
