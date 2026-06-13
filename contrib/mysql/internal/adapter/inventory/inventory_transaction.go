//go:build mysql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventorytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.InventoryTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql inventory_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLInventoryTransactionRepository(dbOps, tableName), nil
	})
}

// MySQLInventoryTransactionRepository implements inventory_transaction CRUD operations using MySQL 8.0+.
type MySQLInventoryTransactionRepository struct {
	inventorytransactionpb.UnimplementedInventoryTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLInventoryTransactionRepository creates a new MySQL inventory transaction repository.
func NewMySQLInventoryTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorytransactionpb.InventoryTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_transaction"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLInventoryTransactionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryTransaction creates a new inventory transaction.
func (r *MySQLInventoryTransactionRepository) CreateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.CreateInventoryTransactionRequest) (*inventorytransactionpb.CreateInventoryTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory transaction data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory transaction: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// ReadInventoryTransaction retrieves an inventory transaction.
func (r *MySQLInventoryTransactionRepository) ReadInventoryTransaction(ctx context.Context, req *inventorytransactionpb.ReadInventoryTransactionRequest) (*inventorytransactionpb.ReadInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory transaction: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// UpdateInventoryTransaction updates an inventory transaction.
func (r *MySQLInventoryTransactionRepository) UpdateInventoryTransaction(ctx context.Context, req *inventorytransactionpb.UpdateInventoryTransactionRequest) (*inventorytransactionpb.UpdateInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory transaction: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteInventoryTransaction deletes an inventory transaction (soft delete).
func (r *MySQLInventoryTransactionRepository) DeleteInventoryTransaction(ctx context.Context, req *inventorytransactionpb.DeleteInventoryTransactionRequest) (*inventorytransactionpb.DeleteInventoryTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory transaction: %w", err)
	}

	return &inventorytransactionpb.DeleteInventoryTransactionResponse{
		Success: true,
	}, nil
}

// ListInventoryTransactions lists inventory transactions.
func (r *MySQLInventoryTransactionRepository) ListInventoryTransactions(ctx context.Context, req *inventorytransactionpb.ListInventoryTransactionsRequest) (*inventorytransactionpb.ListInventoryTransactionsResponse, error) {
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

	unmarshalOpts := protojson.UnmarshalOptions{DiscardUnknown: true}
	var inventoryTransactions []*inventorytransactionpb.InventoryTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// GetInventoryTransactionListPageData retrieves inventory transactions with advanced
// filtering, sorting, searching, and pagination.
//
// Dialect translation from postgres gold standard:
//   - $1,$2,$3 → ? (positional, same left-to-right order)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1
//   - COUNT(*) OVER() + CTE stays — MySQL 8.0+ window functions
func (r *MySQLInventoryTransactionRepository) GetInventoryTransactionListPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryTransactionListPageDataRequest,
) (*inventorytransactionpb.GetInventoryTransactionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory transaction list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	sortField := "it.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Dialect: active = true → active = 1; ILIKE → LIKE; $N → ?
	query := fmt.Sprintf(`
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
			FROM inventory_transaction it
			LEFT JOIN inventory_item ii ON it.inventory_item_id = ii.id AND ii.active = 1
			WHERE it.active = 1
			  AND (? = '' OR
			       it.transaction_type LIKE ? OR
			       it.status LIKE ? OR
			       ii.name LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query,
		searchPattern, searchPattern, searchPattern, searchPattern,
		limit, offset)
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

		inventoryTransactions = append(inventoryTransactions, inventoryTransaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory transaction rows: %w", err)
	}

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

// GetInventoryTransactionItemPageData retrieves a single inventory transaction
// with enhanced item page data.
//
// Dialect: $1 → ?, active = true → active = 1.
func (r *MySQLInventoryTransactionRepository) GetInventoryTransactionItemPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryTransactionItemPageDataRequest,
) (*inventorytransactionpb.GetInventoryTransactionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory transaction item page data request is required")
	}
	if req.InventoryTransactionId == "" {
		return nil, fmt.Errorf("inventory transaction ID is required")
	}

	// Dialect: $1 → ?, active = true → active = 1
	query := `
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
			FROM inventory_transaction it
			LEFT JOIN inventory_item ii ON it.inventory_item_id = ii.id AND ii.active = 1
			WHERE it.id = ? AND it.active = 1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventoryTransactionId)

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

	err := row.Scan(
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

	_ = inventoryItemName
	_ = inventoryItemSku

	return &inventorytransactionpb.GetInventoryTransactionItemPageDataResponse{
		InventoryTransaction: inventoryTransaction,
		Success:              true,
	}, nil
}

// GetInventoryMovementsListPageData retrieves inventory movements with joined
// product/variant/item data.
//
// Dialect translation from postgres gold standard:
//   - TO_CHAR(... AT TIME ZONE 'UTC', 'YYYY-MM-DD') → DATE_FORMAT(CONVERT_TZ(it.transaction_date, @@session.time_zone, 'UTC'), '%Y-%m-%d')
//   - $1 ... $6 → ? (positional, same left-to-right order)
//   - $3::date + interval '1 day' → DATE_ADD(?, INTERVAL 1 DAY)
//   - $2::timestamptz → ? (MySQL driver handles datetime comparison)
//   - ILIKE → LIKE (MySQL ci collation)
//
// CRITICAL: workspace_id isolation via inventory_item.workspace_id.
func (r *MySQLInventoryTransactionRepository) GetInventoryMovementsListPageData(
	ctx context.Context,
	req *inventorytransactionpb.GetInventoryMovementsListPageDataRequest,
) (*inventorytransactionpb.GetInventoryMovementsListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory movements list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	dateFrom := req.GetDateFrom()
	dateTo := req.GetDateTo()
	locationID := req.GetLocationId()
	txType := req.GetTransactionType()
	search := req.GetSearch()

	// Dialect: TO_CHAR(... AT TIME ZONE 'UTC', ...) → DATE_FORMAT(CONVERT_TZ(...), ...)
	// $3::date + interval '1 day' → DATE_ADD(?, INTERVAL 1 DAY)
	// ILIKE → LIKE; $N → ?
	query := `
		SELECT it.id,
		       COALESCE(DATE_FORMAT(CONVERT_TZ(it.transaction_date, @@session.time_zone, 'UTC'), '%Y-%m-%d'), '') AS transaction_date,
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
		FROM inventory_transaction it
		LEFT JOIN inventory_item ii ON it.inventory_item_id = ii.id
		LEFT JOIN product_variant pv ON ii.product_variant_id = pv.id
		LEFT JOIN product p ON pv.product_id = p.id
		WHERE it.active = 1
		  AND (? = '' OR ii.workspace_id = ?)
		  AND (? = '' OR it.transaction_date >= ?)
		  AND (? = '' OR it.transaction_date < DATE_ADD(?, INTERVAL 1 DAY))
		  AND (? = '' OR ii.location_id = ?)
		  AND (? = '' OR it.transaction_type = ?)
		  AND (? = '' OR (
		       p.name LIKE CONCAT('%', ?, '%')
		    OR pv.sku LIKE CONCAT('%', ?, '%')
		    OR ii.sku LIKE CONCAT('%', ?, '%')
		    OR ii.name LIKE CONCAT('%', ?, '%')
		  ))
		ORDER BY it.transaction_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query,
		workspaceID, workspaceID,
		dateFrom, dateFrom,
		dateTo, dateTo,
		locationID, locationID,
		txType, txType,
		search, search, search, search, search,
	)
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

// NewInventoryTransactionRepository creates a new MySQL inventory transaction repository (old-style constructor).
func NewInventoryTransactionRepository(db *sql.DB, tableName string) inventorytransactionpb.InventoryTransactionDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLInventoryTransactionRepository(dbOps, tableName)
}
