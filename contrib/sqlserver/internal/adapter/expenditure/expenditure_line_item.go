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
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenditureLineItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expenditure_line_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenditureLineItemRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenditureLineItemRepository implements expenditure_line_item CRUD using SQL Server.
type SQLServerExpenditureLineItemRepository struct {
	expenditurelineitempb.UnimplementedExpenditureLineItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewSQLServerExpenditureLineItemRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurelineitempb.ExpenditureLineItemDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_line_item"
	}
	return &SQLServerExpenditureLineItemRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerExpenditureLineItemRepository) CreateExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.CreateExpenditureLineItemRequest) (*expenditurelineitempb.CreateExpenditureLineItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure line item data is required")
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
		return nil, fmt.Errorf("failed to create expenditure line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	lineItem := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurelineitempb.CreateExpenditureLineItemResponse{Data: []*expenditurelineitempb.ExpenditureLineItem{lineItem}}, nil
}

func (r *SQLServerExpenditureLineItemRepository) ReadExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.ReadExpenditureLineItemRequest) (*expenditurelineitempb.ReadExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	lineItem := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurelineitempb.ReadExpenditureLineItemResponse{Data: []*expenditurelineitempb.ExpenditureLineItem{lineItem}}, nil
}

func (r *SQLServerExpenditureLineItemRepository) UpdateExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.UpdateExpenditureLineItemRequest) (*expenditurelineitempb.UpdateExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
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
		return nil, fmt.Errorf("failed to update expenditure line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	lineItem := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurelineitempb.UpdateExpenditureLineItemResponse{Data: []*expenditurelineitempb.ExpenditureLineItem{lineItem}}, nil
}

func (r *SQLServerExpenditureLineItemRepository) DeleteExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.DeleteExpenditureLineItemRequest) (*expenditurelineitempb.DeleteExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure line item: %w", err)
	}
	return &expenditurelineitempb.DeleteExpenditureLineItemResponse{Success: true}, nil
}

func (r *SQLServerExpenditureLineItemRepository) ListExpenditureLineItems(ctx context.Context, req *expenditurelineitempb.ListExpenditureLineItemsRequest) (*expenditurelineitempb.ListExpenditureLineItemsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure line items: %w", err)
	}
	var lineItems []*expenditurelineitempb.ExpenditureLineItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure_line_item row: %v", err)
			continue
		}
		lineItem := &expenditurelineitempb.ExpenditureLineItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure_line_item: %v", err)
			continue
		}
		lineItems = append(lineItems, lineItem)
	}
	return &expenditurelineitempb.ListExpenditureLineItemsResponse{Data: lineItems}, nil
}
