//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "expenditure_line_item", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expenditure_line_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresExpenditureLineItemRepository(dbOps, tableName), nil
	})
}

// PostgresExpenditureLineItemRepository implements expenditure line item CRUD operations using PostgreSQL
type PostgresExpenditureLineItemRepository struct {
	expenditurelineitempb.UnimplementedExpenditureLineItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresExpenditureLineItemRepository creates a new PostgreSQL expenditure line item repository
func NewPostgresExpenditureLineItemRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurelineitempb.ExpenditureLineItemDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_line_item"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresExpenditureLineItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenditureLineItem creates a new expenditure line item record
func (r *PostgresExpenditureLineItemRepository) CreateExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.CreateExpenditureLineItemRequest) (*expenditurelineitempb.CreateExpenditureLineItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure line item data is required")
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
		return nil, fmt.Errorf("failed to create expenditure line item: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurelineitempb.CreateExpenditureLineItemResponse{
		Success: true,
		Data:    []*expenditurelineitempb.ExpenditureLineItem{item},
	}, nil
}

// ReadExpenditureLineItem retrieves an expenditure line item record by ID
func (r *PostgresExpenditureLineItemRepository) ReadExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.ReadExpenditureLineItemRequest) (*expenditurelineitempb.ReadExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure line item: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurelineitempb.ReadExpenditureLineItemResponse{
		Success: true,
		Data:    []*expenditurelineitempb.ExpenditureLineItem{item},
	}, nil
}

// UpdateExpenditureLineItem updates an expenditure line item record
func (r *PostgresExpenditureLineItemRepository) UpdateExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.UpdateExpenditureLineItemRequest) (*expenditurelineitempb.UpdateExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
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
		return nil, fmt.Errorf("failed to update expenditure line item: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &expenditurelineitempb.ExpenditureLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurelineitempb.UpdateExpenditureLineItemResponse{
		Success: true,
		Data:    []*expenditurelineitempb.ExpenditureLineItem{item},
	}, nil
}

// DeleteExpenditureLineItem deletes an expenditure line item record (soft delete)
func (r *PostgresExpenditureLineItemRepository) DeleteExpenditureLineItem(ctx context.Context, req *expenditurelineitempb.DeleteExpenditureLineItemRequest) (*expenditurelineitempb.DeleteExpenditureLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure line item ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expenditure line item: %w", err)
	}

	return &expenditurelineitempb.DeleteExpenditureLineItemResponse{
		Success: true,
	}, nil
}

// ListExpenditureLineItems lists expenditure line item records with optional filters
func (r *PostgresExpenditureLineItemRepository) ListExpenditureLineItems(ctx context.Context, req *expenditurelineitempb.ListExpenditureLineItemsRequest) (*expenditurelineitempb.ListExpenditureLineItemsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure line items: %w", err)
	}

	var items []*expenditurelineitempb.ExpenditureLineItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure line item row: %v", err)
			continue
		}

		item := &expenditurelineitempb.ExpenditureLineItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure line item: %v", err)
			continue
		}
		items = append(items, item)
	}

	return &expenditurelineitempb.ListExpenditureLineItemsResponse{
		Success: true,
		Data:    items,
	}, nil
}
