//go:build postgresql

package inventory_depreciation

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
	inventorydepreciationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_depreciation"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "inventory_depreciation", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_depreciation repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventoryDepreciationRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryDepreciationRepository implements inventory_depreciation CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_depreciation_active ON inventory_depreciation(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_depreciation_inventory_item_id ON inventory_depreciation(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_depreciation_method ON inventory_depreciation(method) - Search on method
//   - CREATE INDEX idx_inventory_depreciation_start_date ON inventory_depreciation(start_date) - Sort/filter by start_date
//   - CREATE INDEX idx_inventory_depreciation_date_created ON inventory_depreciation(date_created DESC) - Default sorting
type PostgresInventoryDepreciationRepository struct {
	inventorydepreciationpb.UnimplementedInventoryDepreciationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresInventoryDepreciationRepository creates a new PostgreSQL inventory depreciation repository
func NewPostgresInventoryDepreciationRepository(dbOps interfaces.DatabaseOperation, tableName string) inventorydepreciationpb.InventoryDepreciationDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_depreciation" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventoryDepreciationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryDepreciation creates a new inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) CreateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.CreateInventoryDepreciationRequest) (*inventorydepreciationpb.CreateInventoryDepreciationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory depreciation data is required")
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
		return nil, fmt.Errorf("failed to create inventory depreciation: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := protojson.Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.CreateInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// ReadInventoryDepreciation retrieves an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) ReadInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.ReadInventoryDepreciationRequest) (*inventorydepreciationpb.ReadInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory depreciation: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := protojson.Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.ReadInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// UpdateInventoryDepreciation updates an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) UpdateInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.UpdateInventoryDepreciationRequest) (*inventorydepreciationpb.UpdateInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
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
		return nil, fmt.Errorf("failed to update inventory depreciation: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
	if err := protojson.Unmarshal(resultJSON, inventoryDepreciation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventorydepreciationpb.UpdateInventoryDepreciationResponse{
		Data: []*inventorydepreciationpb.InventoryDepreciation{inventoryDepreciation},
	}, nil
}

// DeleteInventoryDepreciation deletes an inventory depreciation using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) DeleteInventoryDepreciation(ctx context.Context, req *inventorydepreciationpb.DeleteInventoryDepreciationRequest) (*inventorydepreciationpb.DeleteInventoryDepreciationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory depreciation: %w", err)
	}

	return &inventorydepreciationpb.DeleteInventoryDepreciationResponse{
		Success: true,
	}, nil
}

// ListInventoryDepreciations lists inventory depreciations using common PostgreSQL operations
func (r *PostgresInventoryDepreciationRepository) ListInventoryDepreciations(ctx context.Context, req *inventorydepreciationpb.ListInventoryDepreciationsRequest) (*inventorydepreciationpb.ListInventoryDepreciationsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory depreciations: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var inventoryDepreciations []*inventorydepreciationpb.InventoryDepreciation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{}
		if err := protojson.Unmarshal(resultJSON, inventoryDepreciation); err != nil {
			// Log error and continue with next item
			continue
		}
		inventoryDepreciations = append(inventoryDepreciations, inventoryDepreciation)
	}

	return &inventorydepreciationpb.ListInventoryDepreciationsResponse{
		Data: inventoryDepreciations,
	}, nil
}

// GetInventoryDepreciationListPageData retrieves inventory depreciations with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the inventory_item table to include the parent item name
// Supports search on depreciation method
func (r *PostgresInventoryDepreciationRepository) GetInventoryDepreciationListPageData(
	ctx context.Context,
	req *inventorydepreciationpb.GetInventoryDepreciationListPageDataRequest,
) (*inventorydepreciationpb.GetInventoryDepreciationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory depreciation list page data request is required")
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
	sortField := "id2.date_created"
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
				id2.id,
				id2.date_created,
				id2.date_modified,
				id2.active,
				id2.inventory_item_id,
				id2.method,
				id2.cost_basis,
				id2.salvage_value,
				id2.useful_life_months,
				id2.start_date,
				id2.accumulated_depreciation,
				id2.book_value,
				COALESCE(ii.name, '') as inventory_item_name
			FROM inventory_depreciation id2
			LEFT JOIN inventory_item ii ON id2.inventory_item_id = ii.id AND ii.active = true
			WHERE id2.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       id2.method ILIKE $1 OR
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
		return nil, fmt.Errorf("failed to query inventory depreciation list page data: %w", err)
	}
	defer rows.Close()

	var inventoryDepreciations []*inventorydepreciationpb.InventoryDepreciation
	var totalCount int64

	for rows.Next() {
		var (
			id                      string
			dateCreated             time.Time
			dateModified            time.Time
			active                  bool
			inventoryItemID         string
			method                  string
			costBasis               float64
			salvageValue            float64
			usefulLifeMonths        int32
			startDate               string
			accumulatedDepreciation float64
			bookValue               float64
			inventoryItemName       string
			total                   int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&inventoryItemID,
			&method,
			&costBasis,
			&salvageValue,
			&usefulLifeMonths,
			&startDate,
			&accumulatedDepreciation,
			&bookValue,
			&inventoryItemName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory depreciation row: %w", err)
		}

		totalCount = total

		inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{
			Id:                      id,
			Active:                  active,
			InventoryItemId:         inventoryItemID,
			Method:                  method,
			CostBasis:               costBasis,
			SalvageValue:            salvageValue,
			UsefulLifeMonths:        usefulLifeMonths,
			StartDate:               startDate,
			AccumulatedDepreciation: accumulatedDepreciation,
			BookValue:               bookValue,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			inventoryDepreciation.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			inventoryDepreciation.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			inventoryDepreciation.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			inventoryDepreciation.DateModifiedString = &dmStr
		}

		// Note: inventoryItemName is available from the join but not directly mapped
		// to the InventoryDepreciation protobuf. Could be populated via the
		// InventoryItem field if needed for frontend display.

		inventoryDepreciations = append(inventoryDepreciations, inventoryDepreciation)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory depreciation rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventorydepreciationpb.GetInventoryDepreciationListPageDataResponse{
		InventoryDepreciationList: inventoryDepreciations,
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

// GetInventoryDepreciationItemPageData retrieves a single inventory depreciation with enhanced item page data using CTE
// This method joins with the inventory_item table for the parent item reference
func (r *PostgresInventoryDepreciationRepository) GetInventoryDepreciationItemPageData(
	ctx context.Context,
	req *inventorydepreciationpb.GetInventoryDepreciationItemPageDataRequest,
) (*inventorydepreciationpb.GetInventoryDepreciationItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory depreciation item page data request is required")
	}
	if req.InventoryDepreciationId == "" {
		return nil, fmt.Errorf("inventory depreciation ID is required")
	}

	// CTE Query - Single round-trip with inventory_item join
	query := `
		WITH enriched AS (
			SELECT
				id2.id,
				id2.date_created,
				id2.date_modified,
				id2.active,
				id2.inventory_item_id,
				id2.method,
				id2.cost_basis,
				id2.salvage_value,
				id2.useful_life_months,
				id2.start_date,
				id2.accumulated_depreciation,
				id2.book_value,
				COALESCE(ii.name, '') as inventory_item_name,
				COALESCE(ii.sku, '') as inventory_item_sku
			FROM inventory_depreciation id2
			LEFT JOIN inventory_item ii ON id2.inventory_item_id = ii.id AND ii.active = true
			WHERE id2.id = $1 AND id2.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventoryDepreciationId)

	var (
		id                      string
		dateCreated             time.Time
		dateModified            time.Time
		active                  bool
		inventoryItemID         string
		method                  string
		costBasis               float64
		salvageValue            float64
		usefulLifeMonths        int32
		startDate               string
		accumulatedDepreciation float64
		bookValue               float64
		inventoryItemName       string
		inventoryItemSku        string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&inventoryItemID,
		&method,
		&costBasis,
		&salvageValue,
		&usefulLifeMonths,
		&startDate,
		&accumulatedDepreciation,
		&bookValue,
		&inventoryItemName,
		&inventoryItemSku,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory depreciation with ID '%s' not found", req.InventoryDepreciationId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory depreciation item page data: %w", err)
	}

	inventoryDepreciation := &inventorydepreciationpb.InventoryDepreciation{
		Id:                      id,
		Active:                  active,
		InventoryItemId:         inventoryItemID,
		Method:                  method,
		CostBasis:               costBasis,
		SalvageValue:            salvageValue,
		UsefulLifeMonths:        usefulLifeMonths,
		StartDate:               startDate,
		AccumulatedDepreciation: accumulatedDepreciation,
		BookValue:               bookValue,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		inventoryDepreciation.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		inventoryDepreciation.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		inventoryDepreciation.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		inventoryDepreciation.DateModifiedString = &dmStr
	}

	// Note: inventoryItemName and inventoryItemSku are available from the join
	// but not directly mapped to the InventoryDepreciation protobuf. These could be
	// returned via the InventoryItem field or processed separately.

	return &inventorydepreciationpb.GetInventoryDepreciationItemPageDataResponse{
		InventoryDepreciation: inventoryDepreciation,
		Success:               true,
	}, nil
}

// NewInventoryDepreciationRepository creates a new PostgreSQL inventory depreciation repository (old-style constructor)
func NewInventoryDepreciationRepository(db *sql.DB, tableName string) inventorydepreciationpb.InventoryDepreciationDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInventoryDepreciationRepository(dbOps, tableName)
}
