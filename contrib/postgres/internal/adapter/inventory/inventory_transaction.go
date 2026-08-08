//go:build postgresql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventoryTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresInventoryTransactionRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryTransactionRepository implements inventory_transaction CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_transaction_active ON inventory_transaction(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_transaction_inventory_item_id ON inventory_transaction(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_transaction_transaction_type ON inventory_transaction(transaction_type) - Filter by type
//   - CREATE INDEX idx_inventory_transaction_status ON inventory_transaction(status) - Filter by status
//   - CREATE INDEX idx_inventory_transaction_date_created ON inventory_transaction(date_created DESC) - Default sorting
type PostgresInventoryTransactionRepository struct {
	inventorytransactionpb.UnimplementedInventoryTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// directWorkspaceResolver is the capability required by raw adapter queries.
// It proves a selected workspace and that the joined owner table has the
// workspace_id column before the query is allowed to run.
type directWorkspaceResolver interface {
	RequireDirectWorkspace(context.Context, string) (string, error)
}

func requireInventoryWorkspace(ctx context.Context, dbOps interfaces.DatabaseOperation) (string, error) {
	resolver, ok := dbOps.(directWorkspaceResolver)
	if !ok {
		return "", fmt.Errorf("inventory repository requires direct workspace capability")
	}
	return resolver.RequireDirectWorkspace(ctx, entityid.InventoryItem)
}

// NewPostgresInventoryTransactionRepository creates a new PostgreSQL inventory transaction repository
func NewPostgresInventoryTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorytransactionpb.InventoryTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_transaction" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventoryTransactionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryTransaction creates a new inventory transaction using common PostgreSQL operations
func (r *PostgresInventoryTransactionRepository) CreateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory transaction data is required")
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
		return nil, fmt.Errorf("failed to create inventory transaction: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryTransaction := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorytransactionpb.CreateInventoryTransactionResponse{
		Data: []*inventorytransactionpb.InventoryTransaction{inventoryTransaction},
	}, nil
}

// ReadInventoryTransaction retrieves an inventory transaction using common PostgreSQL operations
func (r *PostgresInventoryTransactionRepository) ReadInventoryTransaction(ctx context.Context, req *inventorytransactionpb.ReadInventoryTransactionRequest) (*inventorytransactionpb.ReadInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory transaction: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryTransaction := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorytransactionpb.ReadInventoryTransactionResponse{
		Data: []*inventorytransactionpb.InventoryTransaction{inventoryTransaction},
	}, nil
}

// UpdateInventoryTransaction updates an inventory transaction using common PostgreSQL operations
func (r *PostgresInventoryTransactionRepository) UpdateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
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
		return nil, fmt.Errorf("failed to update inventory transaction: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryTransaction := &inventorytransactionpb.InventoryTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorytransactionpb.UpdateInventoryTransactionResponse{
		Data: []*inventorytransactionpb.InventoryTransaction{inventoryTransaction},
	}, nil
}

// DeleteInventoryTransaction deletes an inventory transaction using common PostgreSQL operations
func (r *PostgresInventoryTransactionRepository) DeleteInventoryTransaction(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory transaction: %w", err)
	}

	return &inventorytransactionpb.DeleteInventoryTransactionResponse{
		Success: true,
	}, nil
}

// ListInventoryTransactions lists inventory transactions using common PostgreSQL operations
func (r *PostgresInventoryTransactionRepository) ListInventoryTransactions(ctx context.Context, req *inventorytransactionpb.ListInventoryTransactionsRequest) (*inventorytransactionpb.ListInventoryTransactionsResponse, error) {
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
		return nil, fmt.Errorf("failed to list inventory transactions: %w", err)
	}

	// Convert results to protobuf slice using protojson
	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	var inventoryTransactions []*inventorytransactionpb.InventoryTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		inventoryTransaction := &inventorytransactionpb.InventoryTransaction{}
		if err := unmarshalOpts.Unmarshal(resultJSON, inventoryTransaction); err != nil {
			continue
		}
		inventoryTransactions = append(inventoryTransactions, inventoryTransaction)
	}

	return &inventorytransactionpb.ListInventoryTransactionsResponse{
		Data: inventoryTransactions,
	}, nil
}

// inventoryTransactionSortableSQLCols is the fail-closed sort whitelist for
// GetInventoryTransactionListPageData. Only columns/aliases projected by the CTE
// SELECT are included so ORDER BY can never reference an unprojected/injected
// identifier.
var inventoryTransactionSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "inventory_item_id",
	"transaction_type", "quantity", "status", "reference_type", "reference_id",
	"from_location_id", "to_location_id", "notes", "serial_number",
	"performed_by", "inventory_item_name",
}

// GetInventoryTransactionListPageData retrieves inventory transactions with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the inventory_item table to include the parent item name
// Supports search on transaction_type, status, and inventory item name
func (r *PostgresInventoryTransactionRepository) GetInventoryTransactionListPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryTransactionListPageDataRequest,
) (*inventorytransactionpb.GetInventoryTransactionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory transaction list page data request is required")
	}

	// Build search condition
	searchPattern, searchErr := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if searchErr != nil {
		return nil, fmt.Errorf("bounded search: %w", searchErr)
	}

	// Default pagination values
	limit, offset, page, paginationErr := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if paginationErr != nil {
		return nil, fmt.Errorf("bounded pagination: %w", paginationErr)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The default
	// references the outer enriched projection (date_created) since the page rows
	// are selected via "SELECT e.* FROM enriched e". An unknown sort column now
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(inventoryTransactionSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	workspaceID, err := requireInventoryWorkspace(ctx, r.dbOps)
	if err != nil {
		return nil, err
	}

	// CTE Query - Single round-trip with inventory_item join
	query := inventoryTransactionListPageDataSQL(orderByClause)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transaction list page data: %w", err)
	}
	defer rows.Close()

	var inventoryTransactions []*inventorytransactionpb.InventoryTransaction
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			inventoryItemID   string
			transactionType   string
			quantity          float64
			status            string
			referenceType     *string
			referenceID       *string
			fromLocationID    *string
			toLocationID      *string
			notes             *string
			serialNumber      *string
			performedBy       *string
			inventoryItemName string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&inventoryItemID,
			&transactionType,
			&quantity,
			&status,
			&referenceType,
			&referenceID,
			&fromLocationID,
			&toLocationID,
			&notes,
			&serialNumber,
			&performedBy,
			&inventoryItemName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory transaction row: %w", err)
		}

		totalCount = total

		inventoryTransaction := &inventorytransactionpb.InventoryTransaction{
			Id:              id,
			Active:          active,
			InventoryItemId: inventoryItemID,
			TransactionType: transactionType,
			Quantity:        quantity,
			Status:          status,
		}

		// Handle nullable fields
		if referenceType != nil {
			inventoryTransaction.ReferenceType = referenceType
		}
		if referenceID != nil {
			inventoryTransaction.ReferenceId = referenceID
		}
		if fromLocationID != nil {
			inventoryTransaction.FromLocationId = fromLocationID
		}
		if toLocationID != nil {
			inventoryTransaction.ToLocationId = toLocationID
		}
		if notes != nil {
			inventoryTransaction.Notes = notes
		}
		if serialNumber != nil {
			inventoryTransaction.SerialNumber = serialNumber
		}
		if performedBy != nil {
			inventoryTransaction.PerformedBy = performedBy
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			inventoryTransaction.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			inventoryTransaction.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			inventoryTransaction.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			inventoryTransaction.DateModifiedString = &dmStr
		}

		// Note: inventoryItemName is available from the join but not directly mapped
		// to the InventoryTransaction protobuf. Could be populated via the
		// InventoryItem field if needed for frontend display.

		inventoryTransactions = append(inventoryTransactions, inventoryTransaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventorytransactionpb.GetInventoryTransactionListPageDataResponse{
		InventoryTransactionList: inventoryTransactions,
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

func inventoryTransactionListPageDataSQL(orderByClause string) string {
	return `
		WITH enriched AS (
			SELECT
				it.id,
				it.date_created,
				it.date_modified,
				it.active,
				it.inventory_item_id,
				it.transaction_type,
				it.quantity,
				it.status,
				it.reference_type,
				it.reference_id,
				it.from_location_id,
				it.to_location_id,
				it.notes,
				it.serial_number,
				it.performed_by,
				COALESCE(ii.name, '') as inventory_item_name
			FROM ` + entityid.InventoryTransaction + ` it
			LEFT JOIN ` + entityid.InventoryItem + ` ii ON it.inventory_item_id = ii.id AND ii.active = true
			WHERE it.active = true
			  AND ii.workspace_id = $4
			  AND ($1::text IS NULL OR $1::text = '' OR
			       it.transaction_type ILIKE $1 OR
			       it.status ILIKE $1 OR
			       ii.name ILIKE $1)
		)
		-- A3 (Q-PAGE-COUNT default tier): COUNT(*) OVER () computes the total in the
		-- same scan as the page rows (the prior counted CTE forced a second scan).
		SELECT
			e.*,
			COUNT(*) OVER () AS total
		FROM enriched e
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`
}

// GetInventoryTransactionItemPageData retrieves a single inventory transaction with enhanced item page data using CTE
// This method joins with the inventory_item table for the parent item reference
func (r *PostgresInventoryTransactionRepository) GetInventoryTransactionItemPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryTransactionItemPageDataRequest,
) (*inventorytransactionpb.GetInventoryTransactionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory transaction item page data request is required")
	}
	if req.InventoryTransactionId == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	workspaceID, err := requireInventoryWorkspace(ctx, r.dbOps)
	if err != nil {
		return nil, err
	}

	// CTE Query - Single round-trip with inventory_item join
	query := inventoryTransactionItemPageDataSQL()

	row := r.db.QueryRowContext(ctx, query, req.InventoryTransactionId, workspaceID)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		inventoryItemID   string
		transactionType   string
		quantity          float64
		status            string
		referenceType     *string
		referenceID       *string
		fromLocationID    *string
		toLocationID      *string
		notes             *string
		serialNumber      *string
		performedBy       *string
		inventoryItemName string
		inventoryItemSku  string
	)

	err = row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&inventoryItemID,
		&transactionType,
		&quantity,
		&status,
		&referenceType,
		&referenceID,
		&fromLocationID,
		&toLocationID,
		&notes,
		&serialNumber,
		&performedBy,
		&inventoryItemName,
		&inventoryItemSku,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory transaction with ID '%s' not found", req.InventoryTransactionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory transaction item page data: %w", err)
	}

	inventoryTransaction := &inventorytransactionpb.InventoryTransaction{
		Id:              id,
		Active:          active,
		InventoryItemId: inventoryItemID,
		TransactionType: transactionType,
		Quantity:        quantity,
		Status:          status,
	}

	// Handle nullable fields
	if referenceType != nil {
		inventoryTransaction.ReferenceType = referenceType
	}
	if referenceID != nil {
		inventoryTransaction.ReferenceId = referenceID
	}
	if fromLocationID != nil {
		inventoryTransaction.FromLocationId = fromLocationID
	}
	if toLocationID != nil {
		inventoryTransaction.ToLocationId = toLocationID
	}
	if notes != nil {
		inventoryTransaction.Notes = notes
	}
	if serialNumber != nil {
		inventoryTransaction.SerialNumber = serialNumber
	}
	if performedBy != nil {
		inventoryTransaction.PerformedBy = performedBy
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		inventoryTransaction.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		inventoryTransaction.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		inventoryTransaction.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		inventoryTransaction.DateModifiedString = &dmStr
	}

	// Note: inventoryItemName and inventoryItemSku are available from the join
	// but not directly mapped to the InventoryTransaction protobuf. These could be
	// returned via the InventoryItem field or processed separately.

	return &inventorytransactionpb.GetInventoryTransactionItemPageDataResponse{
		InventoryTransaction: inventoryTransaction,
		Success:              true,
	}, nil
}

func inventoryTransactionItemPageDataSQL() string {
	return `
		WITH enriched AS (
			SELECT
				it.id,
				it.date_created,
				it.date_modified,
				it.active,
				it.inventory_item_id,
				it.transaction_type,
				it.quantity,
				it.status,
				it.reference_type,
				it.reference_id,
				it.from_location_id,
				it.to_location_id,
				it.notes,
				it.serial_number,
				it.performed_by,
				COALESCE(ii.name, '') as inventory_item_name,
				COALESCE(ii.sku, '') as inventory_item_sku
			FROM ` + entityid.InventoryTransaction + ` it
			LEFT JOIN ` + entityid.InventoryItem + ` ii ON it.inventory_item_id = ii.id AND ii.active = true
			WHERE it.id = $1
			  AND it.active = true
			  AND ii.workspace_id = $2
		)
		SELECT * FROM enriched LIMIT 1;
	`
}

// GetInventoryMovementsListPageData retrieves inventory movements with joined product/variant/item
// data and supports dateFrom, dateTo, location_id, transaction_type, and full-text search filters.
// CRITICAL: Always filters by workspace_id for multi-tenancy (via inventory_item.workspace_id).
func (r *PostgresInventoryTransactionRepository) GetInventoryMovementsListPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryMovementsListPageDataRequest,
) (*inventorytransactionpb.GetInventoryMovementsListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory movements list page data request is required")
	}

	workspaceID, err := requireInventoryWorkspace(ctx, r.dbOps)
	if err != nil {
		return nil, err
	}

	dateFrom := req.GetDateFrom()
	dateTo := req.GetDateTo()
	locationID := req.GetLocationId()
	txType := req.GetTransactionType()
	searchPattern, err := inventoryMovementsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("bounded search: %w", err)
	}

	// A12: Apply LIMIT/OFFSET for pagination safety. The proto contract for this
	// request/response has no Pagination field, so we apply a large but finite
	// cap (10 000 rows) to prevent unbounded full-table scans. Callers that
	// previously received the full set continue to work correctly because
	// real-world inventory_transaction tables are well under this threshold.
	const (
		inventoryMovementsLimit  = int32(10000)
		inventoryMovementsOffset = int32(0)
	)

	query := inventoryMovementsListPageDataSQL()

	rows, err := r.db.QueryContext(ctx, query, workspaceID, dateFrom, dateTo, locationID, txType, searchPattern, inventoryMovementsLimit, inventoryMovementsOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory movements: %w", err)
	}
	defer rows.Close()

	var result []*inventorytransactionpb.InventoryMovementRow
	for rows.Next() {
		var (
			id              string
			transactionDate string
			transactionType string
			quantity        float64
			itemName        string
			locationIDVal   string
			itemSKU         string
			variantSKU      string
			productName     string
			serialNumber    sql.NullString
			referenceType   sql.NullString
			referenceID     sql.NullString
			performedBy     sql.NullString
		)
		if err := rows.Scan(
			&id, &transactionDate, &transactionType, &quantity,
			&itemName, &locationIDVal, &itemSKU, &variantSKU, &productName,
			&serialNumber, &referenceType, &referenceID, &performedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan inventory movement row: %w", err)
		}

		row := &inventorytransactionpb.InventoryMovementRow{
			Id:              id,
			TransactionDate: transactionDate,
			TransactionType: transactionType,
			Quantity:        quantity,
			ItemName:        itemName,
			LocationId:      locationIDVal,
			ItemSku:         itemSKU,
			VariantSku:      variantSKU,
			ProductName:     productName,
		}
		if serialNumber.Valid {
			row.SerialNumber = &serialNumber.String
		}
		if referenceType.Valid {
			row.ReferenceType = &referenceType.String
		}
		if referenceID.Valid {
			row.ReferenceId = &referenceID.String
		}
		if performedBy.Valid {
			row.PerformedBy = &performedBy.String
		}
		result = append(result, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory movements rows: %w", err)
	}

	return &inventorytransactionpb.GetInventoryMovementsListPageDataResponse{
		Data:    result,
		Success: true,
	}, nil
}

func inventoryMovementsSearchPattern(search string) (string, error) {
	return postgresCore.BoundedContainsPattern(search)
}

func inventoryMovementsListPageDataSQL() string {
	return `
		SELECT it.id,
		       COALESCE(TO_CHAR(it.transaction_date AT TIME ZONE 'UTC', 'YYYY-MM-DD'), '') AS transaction_date,
		       it.transaction_type,
		       it.quantity,
		       COALESCE(ii.name, '')          AS item_name,
		       COALESCE(ii.location_id, '')   AS location_id,
		       COALESCE(ii.sku, '')            AS item_sku,
		       COALESCE(pv.sku, '')            AS variant_sku,
		       COALESCE(p.name, '')            AS product_name,
		       it.serial_number,
		       it.reference_type,
		       it.reference_id,
		       it.performed_by
		FROM ` + entityid.InventoryTransaction + ` it
		LEFT JOIN ` + entityid.InventoryItem + ` ii ON it.inventory_item_id = ii.id
		LEFT JOIN ` + entityid.ProductVariant + ` pv ON ii.product_variant_id = pv.id
		LEFT JOIN ` + entityid.Product + ` p ON pv.product_id = p.id
		WHERE it.active = true
		  AND ii.workspace_id = $1
		  AND ($2 = '' OR it.transaction_date >= $2::timestamptz)
		  AND ($3 = '' OR it.transaction_date <= ($3::date + interval '1 day')::timestamptz)
		  AND ($4 = '' OR ii.location_id = $4)
		  AND ($5 = '' OR it.transaction_type = $5)
		  AND ($6 = '' OR (
		       p.name ILIKE $6 ESCAPE '\'
		    OR pv.sku ILIKE $6 ESCAPE '\'
		    OR ii.sku ILIKE $6 ESCAPE '\'
		    OR ii.name ILIKE $6 ESCAPE '\'
		  ))
		ORDER BY it.transaction_date DESC
		LIMIT $7 OFFSET $8
	`
}

// NewInventoryTransactionRepository creates a new PostgreSQL inventory transaction repository (old-style constructor)
func NewInventoryTransactionRepository(db *sql.DB, tableName string) inventorytransactionpb.InventoryTransactionDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresInventoryTransactionRepository(dbOps, tableName)
}
