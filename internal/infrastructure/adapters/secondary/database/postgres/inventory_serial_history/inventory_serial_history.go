//go:build postgresql

package inventory_serial_history

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
	serialhistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/serial_history"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "inventory_serial_history", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_serial_history repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventorySerialHistoryRepository(dbOps, tableName), nil
	})
}

// PostgresInventorySerialHistoryRepository implements inventory_serial_history operations using PostgreSQL
// This is an IMMUTABLE audit trail — records are never updated, only appended.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_serial_history_inventory_serial_id ON inventory_serial_history(inventory_serial_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_history_inventory_item_id ON inventory_serial_history(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_serial_history_from_status ON inventory_serial_history(from_status) - Filter by from_status
//   - CREATE INDEX idx_inventory_serial_history_to_status ON inventory_serial_history(to_status) - Filter by to_status
//   - CREATE INDEX idx_inventory_serial_history_reference_type ON inventory_serial_history(reference_type) - Filter by reference_type
//   - CREATE INDEX idx_inventory_serial_history_date_created ON inventory_serial_history(date_created DESC) - Default sorting
type PostgresInventorySerialHistoryRepository struct {
	serialhistorypb.UnimplementedInventorySerialHistoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresInventorySerialHistoryRepository creates a new PostgreSQL inventory serial history repository
func NewPostgresInventorySerialHistoryRepository(dbOps interfaces.DatabaseOperation, tableName string) serialhistorypb.InventorySerialHistoryDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_serial_history" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventorySerialHistoryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventorySerialHistory creates a new inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) CreateInventorySerialHistory(ctx context.Context, req *serialhistorypb.CreateInventorySerialHistoryRequest) (*serialhistorypb.CreateInventorySerialHistoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory serial history data is required")
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
		return nil, fmt.Errorf("failed to create inventory serial history: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	serialHistory := &serialhistorypb.InventorySerialHistory{}
	if err := protojson.Unmarshal(resultJSON, serialHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &serialhistorypb.CreateInventorySerialHistoryResponse{
		Data: []*serialhistorypb.InventorySerialHistory{serialHistory},
	}, nil
}

// ReadInventorySerialHistory retrieves an inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) ReadInventorySerialHistory(ctx context.Context, req *serialhistorypb.ReadInventorySerialHistoryRequest) (*serialhistorypb.ReadInventorySerialHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory serial history: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	serialHistory := &serialhistorypb.InventorySerialHistory{}
	if err := protojson.Unmarshal(resultJSON, serialHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &serialhistorypb.ReadInventorySerialHistoryResponse{
		Data: []*serialhistorypb.InventorySerialHistory{serialHistory},
	}, nil
}

// NOTE: UpdateInventorySerialHistory is intentionally NOT implemented.
// This is an immutable audit trail — records are never updated, only appended.
// The Unimplemented method from the embedded server will return codes.Unimplemented.

// DeleteInventorySerialHistory deletes an inventory serial history record using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) DeleteInventorySerialHistory(ctx context.Context, req *serialhistorypb.DeleteInventorySerialHistoryRequest) (*serialhistorypb.DeleteInventorySerialHistoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory serial history: %w", err)
	}

	return &serialhistorypb.DeleteInventorySerialHistoryResponse{
		Success: true,
	}, nil
}

// ListInventorySerialHistory lists inventory serial history records using common PostgreSQL operations
func (r *PostgresInventorySerialHistoryRepository) ListInventorySerialHistory(ctx context.Context, req *serialhistorypb.ListInventorySerialHistoryRequest) (*serialhistorypb.ListInventorySerialHistoryResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory serial history: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var serialHistories []*serialhistorypb.InventorySerialHistory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		serialHistory := &serialhistorypb.InventorySerialHistory{}
		if err := protojson.Unmarshal(resultJSON, serialHistory); err != nil {
			// Log error and continue with next item
			continue
		}
		serialHistories = append(serialHistories, serialHistory)
	}

	return &serialhistorypb.ListInventorySerialHistoryResponse{
		Data: serialHistories,
	}, nil
}

// GetInventorySerialHistoryListPageData retrieves inventory serial history with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the inventory_serial table to include the serial number
// Supports search on from_status, to_status, reference_type, and notes
func (r *PostgresInventorySerialHistoryRepository) GetInventorySerialHistoryListPageData(
	ctx context.Context,
	req *serialhistorypb.GetInventorySerialHistoryListPageDataRequest,
) (*serialhistorypb.GetInventorySerialHistoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory serial history list page data request is required")
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
	sortField := "ish.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with inventory_serial join
	query := `
		WITH enriched AS (
			SELECT
				ish.id,
				ish.date_created,
				ish.inventory_serial_id,
				ish.inventory_item_id,
				ish.from_status,
				ish.to_status,
				ish.reference_type,
				ish.reference_id,
				ish.notes,
				ish.changed_by,
				ish.changed_by_role,
				COALESCE(is2.serial_number, '') as serial_number
			FROM inventory_serial_history ish
			LEFT JOIN inventory_serial is2 ON ish.inventory_serial_id = is2.id AND is2.active = true
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       ish.from_status ILIKE $1 OR
			       ish.to_status ILIKE $1 OR
			       ish.reference_type ILIKE $1 OR
			       ish.notes ILIKE $1 OR
			       is2.serial_number ILIKE $1)
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
		return nil, fmt.Errorf("failed to query inventory serial history list page data: %w", err)
	}
	defer rows.Close()

	var serialHistories []*serialhistorypb.InventorySerialHistory
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			inventorySerialID string
			inventoryItemID   string
			fromStatus        string
			toStatus          string
			referenceType     string
			referenceID       string
			notes             string
			changedBy         string
			changedByRole     string
			serialNumber      string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&inventorySerialID,
			&inventoryItemID,
			&fromStatus,
			&toStatus,
			&referenceType,
			&referenceID,
			&notes,
			&changedBy,
			&changedByRole,
			&serialNumber,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory serial history row: %w", err)
		}

		totalCount = total

		serialHistory := &serialhistorypb.InventorySerialHistory{
			Id:                id,
			InventorySerialId: inventorySerialID,
			InventoryItemId:   inventoryItemID,
			FromStatus:        fromStatus,
			ToStatus:          toStatus,
			ReferenceType:     referenceType,
			ReferenceId:       referenceID,
			Notes:             notes,
			ChangedBy:         changedBy,
			ChangedByRole:     changedByRole,
		}

		// Parse timestamp if provided (no date_modified for immutable records)
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			serialHistory.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			serialHistory.DateCreatedString = &dcStr
		}

		// Note: serialNumber is available from the join but not directly mapped
		// to the InventorySerialHistory protobuf. Could be populated via the
		// Serial field if needed for frontend display.

		serialHistories = append(serialHistories, serialHistory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory serial history rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &serialhistorypb.GetInventorySerialHistoryListPageDataResponse{
		InventorySerialHistoryList: serialHistories,
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

// GetInventorySerialHistoryItemPageData retrieves a single inventory serial history with enhanced item page data using CTE
// This method joins with the inventory_serial table for the serial reference
func (r *PostgresInventorySerialHistoryRepository) GetInventorySerialHistoryItemPageData(
	ctx context.Context,
	req *serialhistorypb.GetInventorySerialHistoryItemPageDataRequest,
) (*serialhistorypb.GetInventorySerialHistoryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory serial history item page data request is required")
	}
	if req.InventorySerialHistoryId == "" {
		return nil, fmt.Errorf("inventory serial history ID is required")
	}

	// CTE Query - Single round-trip with inventory_serial join
	query := `
		WITH enriched AS (
			SELECT
				ish.id,
				ish.date_created,
				ish.inventory_serial_id,
				ish.inventory_item_id,
				ish.from_status,
				ish.to_status,
				ish.reference_type,
				ish.reference_id,
				ish.notes,
				ish.changed_by,
				ish.changed_by_role,
				COALESCE(is2.serial_number, '') as serial_number,
				COALESCE(ii.name, '') as inventory_item_name
			FROM inventory_serial_history ish
			LEFT JOIN inventory_serial is2 ON ish.inventory_serial_id = is2.id AND is2.active = true
			LEFT JOIN inventory_item ii ON ish.inventory_item_id = ii.id AND ii.active = true
			WHERE ish.id = $1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventorySerialHistoryId)

	var (
		id                string
		dateCreated       time.Time
		inventorySerialID string
		inventoryItemID   string
		fromStatus        string
		toStatus          string
		referenceType     string
		referenceID       string
		notes             string
		changedBy         string
		changedByRole     string
		serialNumber      string
		inventoryItemName string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&inventorySerialID,
		&inventoryItemID,
		&fromStatus,
		&toStatus,
		&referenceType,
		&referenceID,
		&notes,
		&changedBy,
		&changedByRole,
		&serialNumber,
		&inventoryItemName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory serial history with ID '%s' not found", req.InventorySerialHistoryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory serial history item page data: %w", err)
	}

	serialHistory := &serialhistorypb.InventorySerialHistory{
		Id:                id,
		InventorySerialId: inventorySerialID,
		InventoryItemId:   inventoryItemID,
		FromStatus:        fromStatus,
		ToStatus:          toStatus,
		ReferenceType:     referenceType,
		ReferenceId:       referenceID,
		Notes:             notes,
		ChangedBy:         changedBy,
		ChangedByRole:     changedByRole,
	}

	// Parse timestamp if provided (no date_modified for immutable records)
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		serialHistory.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		serialHistory.DateCreatedString = &dcStr
	}

	// Note: serialNumber and inventoryItemName are available from the join
	// but not directly mapped to the InventorySerialHistory protobuf. These could be
	// returned via the Serial/InventoryItem fields or processed separately.

	return &serialhistorypb.GetInventorySerialHistoryItemPageDataResponse{
		InventorySerialHistory: serialHistory,
		Success:                true,
	}, nil
}

// NewInventorySerialHistoryRepository creates a new PostgreSQL inventory serial history repository (old-style constructor)
func NewInventorySerialHistoryRepository(db *sql.DB, tableName string) serialhistorypb.InventorySerialHistoryDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInventorySerialHistoryRepository(dbOps, tableName)
}
