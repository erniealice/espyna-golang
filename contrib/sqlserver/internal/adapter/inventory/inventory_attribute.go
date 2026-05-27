//go:build sqlserver

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventoryAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventoryAttributeRepository(dbOps, tableName), nil
	})
}

// SQLServerInventoryAttributeRepository implements inventory_attribute CRUD operations using SQL Server.
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - ILIKE → LIKE
//   - active = true → active = 1
//   - Pagination: OFFSET/FETCH with mandatory ORDER BY
//   - COUNT(*) OVER () retained (SQL Server 2017+)
type SQLServerInventoryAttributeRepository struct {
	inventoryattributepb.UnimplementedInventoryAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventoryAttributeRepository creates a new SQL Server inventory attribute repository.
func NewSQLServerInventoryAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryattributepb.InventoryAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_attribute"
	}
	return &SQLServerInventoryAttributeRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerInventoryAttributeRepository) CreateInventoryAttribute(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) (*inventoryattributepb.CreateInventoryAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory attribute data is required")
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
		return nil, fmt.Errorf("failed to create inventory attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	attr := &inventoryattributepb.InventoryAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryattributepb.CreateInventoryAttributeResponse{Data: []*inventoryattributepb.InventoryAttribute{attr}}, nil
}

func (r *SQLServerInventoryAttributeRepository) ReadInventoryAttribute(ctx context.Context, req *inventoryattributepb.ReadInventoryAttributeRequest) (*inventoryattributepb.ReadInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	attr := &inventoryattributepb.InventoryAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryattributepb.ReadInventoryAttributeResponse{Data: []*inventoryattributepb.InventoryAttribute{attr}}, nil
}

func (r *SQLServerInventoryAttributeRepository) UpdateInventoryAttribute(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) (*inventoryattributepb.UpdateInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
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
		return nil, fmt.Errorf("failed to update inventory attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	attr := &inventoryattributepb.InventoryAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &inventoryattributepb.UpdateInventoryAttributeResponse{Data: []*inventoryattributepb.InventoryAttribute{attr}}, nil
}

func (r *SQLServerInventoryAttributeRepository) DeleteInventoryAttribute(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) (*inventoryattributepb.DeleteInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory attribute: %w", err)
	}
	return &inventoryattributepb.DeleteInventoryAttributeResponse{Success: true}, nil
}

func (r *SQLServerInventoryAttributeRepository) ListInventoryAttributes(ctx context.Context, req *inventoryattributepb.ListInventoryAttributesRequest) (*inventoryattributepb.ListInventoryAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory attributes: %w", err)
	}
	var attrs []*inventoryattributepb.InventoryAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		attr := &inventoryattributepb.InventoryAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
			continue
		}
		attrs = append(attrs, attr)
	}
	return &inventoryattributepb.ListInventoryAttributesResponse{Data: attrs}, nil
}

// inventoryAttributeSortableSQLCols lists the SQL column names safe to sort by.
var inventoryAttributeSortableSQLCols = []string{
	"ia.date_created",
	"ia.date_modified",
	"ia.inventory_item_id",
}

// GetInventoryAttributeListPageData retrieves inventory attributes with pagination.
//
// SQL Server differences vs postgres:
//   - ILIKE → LIKE; $N → @pN; active = true → active = 1.
//   - Pagination: OFFSET/FETCH; ORDER BY required (BuildOrderBy guarantees).
//   - workspace_id filter for multi-tenancy.
func (r *SQLServerInventoryAttributeRepository) GetInventoryAttributeListPageData(
	ctx context.Context,
	req *inventoryattributepb.GetInventoryAttributeListPageDataRequest,
) (*inventoryattributepb.GetInventoryAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	orderByClause, err := sqlserverCore.BuildOrderBy(inventoryAttributeSortableSQLCols, req.GetSort(), "ia.date_created DESC")
	if err != nil {
		return nil, err
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if op := req.Pagination.GetOffset(); op != nil && op.Page > 0 {
			page = op.Page
			offset = (page - 1) * limit
		}
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ia.id,
				ia.date_created,
				ia.date_modified,
				ia.active,
				ia.inventory_item_id,
				ia.attribute_id,
				ia.value
			FROM inventory_attribute ia
			WHERE ia.active = 1
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		%s OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY;
	`, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory attribute list: %w", err)
	}
	defer rows.Close()

	var attrs []*inventoryattributepb.InventoryAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			inventoryItemID string
			attributeID     string
			value           sql.NullString
			total           int64
		)
		if err := rows.Scan(&id, &dateCreated, &dateModified, &active, &inventoryItemID, &attributeID, &value, &total); err != nil {
			return nil, fmt.Errorf("failed to scan inventory attribute row: %w", err)
		}
		totalCount = total
		attr := &inventoryattributepb.InventoryAttribute{
			Id:              id,
			Active:          active,
			InventoryItemId: inventoryItemID,
			AttributeId:     attributeID,
		}
		if value.Valid {
			attr.Value = value.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			attr.DateCreated = &ts
		}
		attrs = append(attrs, attr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory attribute rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventoryattributepb.GetInventoryAttributeListPageDataResponse{
		InventoryAttributeList: attrs,
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

func (r *SQLServerInventoryAttributeRepository) GetInventoryAttributeItemPageData(ctx context.Context, req *inventoryattributepb.GetInventoryAttributeItemPageDataRequest) (*inventoryattributepb.GetInventoryAttributeItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryAttributeItemPageData not yet implemented")
}
