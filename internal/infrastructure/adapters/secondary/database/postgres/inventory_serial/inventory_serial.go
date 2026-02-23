//go:build postgresql

package inventory_serial

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryserialpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_serial"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "inventory_serial", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_serial repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventorySerialRepository(dbOps, tableName), nil
	})
}

// PostgresInventorySerialRepository implements inventory_serial CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_serial_active ON inventory_serial(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_serial_inventory_item_id ON inventory_serial(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_serial_number ON inventory_serial(serial_number) - Search on serial_number
//   - CREATE INDEX idx_inventory_serial_imei ON inventory_serial(imei) - Search on imei
//   - CREATE INDEX idx_inventory_serial_status ON inventory_serial(status) - Filter by status
//   - CREATE INDEX idx_inventory_serial_date_created ON inventory_serial(date_created DESC) - Default sorting
type PostgresInventorySerialRepository struct {
	inventoryserialpb.UnimplementedInventorySerialDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresInventorySerialRepository creates a new PostgreSQL inventory serial repository
func NewPostgresInventorySerialRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryserialpb.InventorySerialDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventorySerialRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventorySerial creates a new inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) CreateInventorySerial(ctx context.Context, req *inventoryserialpb.CreateInventorySerialRequest) (*inventoryserialpb.CreateInventorySerialResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory serial data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory serial: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := protojson.Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.CreateInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// ReadInventorySerial retrieves an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) ReadInventorySerial(ctx context.Context, req *inventoryserialpb.ReadInventorySerialRequest) (*inventoryserialpb.ReadInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory serial: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := protojson.Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.ReadInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// UpdateInventorySerial updates an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) UpdateInventorySerial(ctx context.Context, req *inventoryserialpb.UpdateInventorySerialRequest) (*inventoryserialpb.UpdateInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory serial: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{}
	if err := protojson.Unmarshal(resultJSON, inventorySerial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryserialpb.UpdateInventorySerialResponse{
		Data: []*inventoryserialpb.InventorySerial{inventorySerial},
	}, nil
}

// DeleteInventorySerial deletes an inventory serial using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) DeleteInventorySerial(ctx context.Context, req *inventoryserialpb.DeleteInventorySerialRequest) (*inventoryserialpb.DeleteInventorySerialResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory serial: %w", err)
	}

	return &inventoryserialpb.DeleteInventorySerialResponse{
		Success: true,
	}, nil
}

// ListInventorySerials lists inventory serials using common PostgreSQL operations
func (r *PostgresInventorySerialRepository) ListInventorySerials(ctx context.Context, req *inventoryserialpb.ListInventorySerialsRequest) (*inventoryserialpb.ListInventorySerialsResponse, error) {
	// Build filter params, honoring entity-specific InventoryItemId field
	var params *interfaces.ListParams
	if req != nil {
		filters := req.Filters
		if itemID := req.GetInventoryItemId(); itemID != "" {
			itemFilter := &commonpb.TypedFilter{
				Field: "inventory_item_id",
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value:    itemID,
						Operator: commonpb.StringOperator_STRING_EQUALS,
					},
				},
			}
			if filters == nil {
				filters = &commonpb.FilterRequest{}
			}
			filters.Filters = append(filters.Filters, itemFilter)
		}
		if filters != nil {
			params = &interfaces.ListParams{Filters: filters}
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory serials: %w", err)
	}

	// Convert results to protobuf slice using protojson
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	var inventorySerials []*inventoryserialpb.InventorySerial
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		inventorySerial := &inventoryserialpb.InventorySerial{}
		if err := unmarshalOpts.Unmarshal(resultJSON, inventorySerial); err != nil {
			continue
		}
		inventorySerials = append(inventorySerials, inventorySerial)
	}

	return &inventoryserialpb.ListInventorySerialsResponse{
		Data: inventorySerials,
	}, nil
}

// GetInventorySerialListPageData retrieves inventory serials with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the inventory_item table to include the parent item name
// Supports search on serial_number and imei, and filtering by status
func (r *PostgresInventorySerialRepository) GetInventorySerialListPageData(
	ctx context.Context,
	req *inventoryserialpb.GetInventorySerialListPageDataRequest,
) (*inventoryserialpb.GetInventorySerialListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory serial list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "is2.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with inventory_item join
	query := `
		WITH enriched AS (
			SELECT
				is2.id,
				is2.date_created,
				is2.date_modified,
				is2.active,
				is2.inventory_item_id,
				is2.serial_number,
				is2.imei,
				is2.status,
				is2.warranty_start,
				is2.warranty_end,
				is2.purchase_order,
				is2.notes,
				COALESCE(ii.name, '') as inventory_item_name
			FROM inventory_serial is2
			LEFT JOIN inventory_item ii ON is2.inventory_item_id = ii.id AND ii.active = true
			WHERE is2.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       is2.serial_number ILIKE $1 OR
			       is2.imei ILIKE $1 OR
			       ii.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory serial list page data: %w", err)
	}
	defer rows.Close()

	var inventorySerials []*inventoryserialpb.InventorySerial
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			inventoryItemID   string
			serialNumber      string
			imei              *string
			status            string
			warrantyStart     *string
			warrantyEnd       *string
			purchaseOrder     *string
			notes             *string
			inventoryItemName string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&inventoryItemID,
			&serialNumber,
			&imei,
			&status,
			&warrantyStart,
			&warrantyEnd,
			&purchaseOrder,
			&notes,
			&inventoryItemName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory serial row: %w", err)
		}

		totalCount = total

		inventorySerial := &inventoryserialpb.InventorySerial{
			Id:              id,
			Active:          active,
			InventoryItemId: inventoryItemID,
			SerialNumber:    serialNumber,
			Status:          status,
		}

		// Handle nullable fields
		if imei != nil {
			inventorySerial.Imei = imei
		}
		if warrantyStart != nil {
			inventorySerial.WarrantyStart = warrantyStart
		}
		if warrantyEnd != nil {
			inventorySerial.WarrantyEnd = warrantyEnd
		}
		if purchaseOrder != nil {
			inventorySerial.PurchaseOrder = purchaseOrder
		}
if notes != nil {
			inventorySerial.Notes = notes
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			inventorySerial.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			inventorySerial.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			inventorySerial.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			inventorySerial.DateModifiedString = &dmStr
		}

		// Note: inventoryItemName is available from the join but not directly mapped
		// to the InventorySerial protobuf. Could be populated via the InventoryItem field
		// if needed for frontend display.

		inventorySerials = append(inventorySerials, inventorySerial)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory serial rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventoryserialpb.GetInventorySerialListPageDataResponse{
		InventorySerialList: inventorySerials,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetInventorySerialItemPageData retrieves a single inventory serial with enhanced item page data using CTE
// This method joins with the inventory_item table for the parent item reference
func (r *PostgresInventorySerialRepository) GetInventorySerialItemPageData(
	ctx context.Context,
	req *inventoryserialpb.GetInventorySerialItemPageDataRequest,
) (*inventoryserialpb.GetInventorySerialItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory serial item page data request is required")
	}
	if req.InventorySerialId == "" {
		return nil, fmt.Errorf("inventory serial ID is required")
	}

	// CTE Query - Single round-trip with inventory_item join
	query := `
		WITH enriched AS (
			SELECT
				is2.id,
				is2.date_created,
				is2.date_modified,
				is2.active,
				is2.inventory_item_id,
				is2.serial_number,
				is2.imei,
				is2.status,
				is2.warranty_start,
				is2.warranty_end,
				is2.purchase_order,
				is2.notes,
				COALESCE(ii.name, '') as inventory_item_name,
				COALESCE(ii.sku, '') as inventory_item_sku
			FROM inventory_serial is2
			LEFT JOIN inventory_item ii ON is2.inventory_item_id = ii.id AND ii.active = true
			WHERE is2.id = $1 AND is2.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventorySerialId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		inventoryItemID   string
		serialNumber      string
		imei              *string
		status            string
		warrantyStart     *string
		warrantyEnd       *string
		purchaseOrder     *string
		notes             *string
		inventoryItemName string
		inventoryItemSku  string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&inventoryItemID,
		&serialNumber,
		&imei,
		&status,
		&warrantyStart,
		&warrantyEnd,
		&purchaseOrder,
		&notes,
		&inventoryItemName,
		&inventoryItemSku,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory serial with ID '%s' not found", req.InventorySerialId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory serial item page data: %w", err)
	}

	inventorySerial := &inventoryserialpb.InventorySerial{
		Id:              id,
		Active:          active,
		InventoryItemId: inventoryItemID,
		SerialNumber:    serialNumber,
		Status:          status,
	}

	// Handle nullable fields
	if imei != nil {
		inventorySerial.Imei = imei
	}
	if warrantyStart != nil {
		inventorySerial.WarrantyStart = warrantyStart
	}
	if warrantyEnd != nil {
		inventorySerial.WarrantyEnd = warrantyEnd
	}
	if purchaseOrder != nil {
		inventorySerial.PurchaseOrder = purchaseOrder
	}
	if notes != nil {
		inventorySerial.Notes = notes
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		inventorySerial.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		inventorySerial.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		inventorySerial.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		inventorySerial.DateModifiedString = &dmStr
	}

	// Note: inventoryItemName and inventoryItemSku are available from the join
	// but not directly mapped to the InventorySerial protobuf. These could be
	// returned via the InventoryItem field or processed separately.

	return &inventoryserialpb.GetInventorySerialItemPageDataResponse{
		InventorySerial: inventorySerial,
		Success:         true,
	}, nil
}

// NewInventorySerialRepository creates a new PostgreSQL inventory serial repository (old-style constructor)
func NewInventorySerialRepository(db *sql.DB, tableName string) inventoryserialpb.InventorySerialDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInventorySerialRepository(dbOps, tableName)
}
