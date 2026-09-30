//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ExpenditureLineItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expenditure_line_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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

	// Map proto totalPrice → DB column line_amount
	if v, ok := data["totalPrice"]; ok {
		data["line_amount"] = v
		delete(data, "totalPrice")
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure line item: %w", err)
	}

	normalizeLineItemRow(result)
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

	normalizeLineItemRow(result)
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

	// Same proto totalPrice → DB column line_amount mapping as Create.
	if v, ok := data["totalPrice"]; ok {
		data["line_amount"] = v
		delete(data, "totalPrice")
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expenditure line item: %w", err)
	}

	normalizeLineItemRow(result)
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

// ListExpenditureLineItems lists expenditure line item records with optional filters.
// Supports filtering by expenditure_id when req.ExpenditureId is set.
func (r *PostgresExpenditureLineItemRepository) ListExpenditureLineItems(ctx context.Context, req *expenditurelineitempb.ListExpenditureLineItemsRequest) (*expenditurelineitempb.ListExpenditureLineItemsResponse, error) {
	// Search/sort/pagination are forwarded like the sibling adapters (allocation_batch.go
	// ListAllocationBatches). The expenditure_id request field is pushed down as an AND-ed equality
	// filter so it is applied BEFORE paging (a post-page client-side filter would return short pages).
	filters := req.GetFilters()
	if id := req.GetExpenditureId(); id != "" {
		merged := &commonpb.FilterRequest{Logic: filters.GetLogic()}
		merged.Filters = append(merged.Filters, filters.GetFilters()...)
		merged.Filters = append(merged.Filters, &commonpb.TypedFilter{
			Field: "expenditure_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: id, Operator: commonpb.StringOperator_STRING_EQUALS,
			}},
		})
		filters = merged
	}
	params, err := postgresCore.ScopedListParams(req.GetSearch(), filters, req.GetSort(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure line items: %w", err)
	}

	var items []*expenditurelineitempb.ExpenditureLineItem
	for _, result := range listResult.Data {
		normalizeLineItemRow(result)

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

// normalizeLineItemRow reconciles the two storage columns of the proto field total_price on a row
// read back from the generic operations. The legacy column line_amount (NOT NULL, the one Create
// writes and the reports sum) is the source of truth; the descriptor-aligned total_price column
// (20260822213000, absent on older databases, NULL for rows written through Create) would decode
// to the SAME proto field, and protojson rejects a document carrying both spellings of one field
// ("duplicate field totalPrice"). Emit exactly one: totalPrice from line_amount, falling back to
// total_price when line_amount is absent.
func normalizeLineItemRow(row map[string]any) {
	total, hasTotal := row["total_price"]
	delete(row, "total_price")
	delete(row, "totalPrice")
	if v, ok := row["line_amount"]; ok && v != nil {
		row["totalPrice"] = v
	} else if hasTotal && total != nil {
		row["totalPrice"] = total
	}
}
