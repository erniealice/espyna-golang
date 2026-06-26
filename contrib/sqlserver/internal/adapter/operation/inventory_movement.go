//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/inventory_movement"
)

// inventoryMovementSortableSQLColsSS mirrors the postgres whitelist for SQL Server.
var inventoryMovementSortableSQLColsSS = []string{
	"date_created",
	"quantity",
	"unit_cost",
	"movement_date",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventoryMovement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_movement repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventoryMovementRepository(dbOps, tableName), nil
	})
}

// SQLServerInventoryMovementRepository implements inventory_movement CRUD + custom operations using SQL Server.
type SQLServerInventoryMovementRepository struct {
	pb.UnimplementedInventoryMovementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventoryMovementRepository creates a new SQL Server inventory_movement repository.
func NewSQLServerInventoryMovementRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.InventoryMovementDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_movement"
	}
	return &SQLServerInventoryMovementRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerInventoryMovementRepository) CreateInventoryMovement(ctx context.Context, req *pb.CreateInventoryMovementRequest) (*pb.CreateInventoryMovementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory movement data is required")
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
		return nil, fmt.Errorf("failed to create inventory movement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	mov := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, mov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.CreateInventoryMovementResponse{Data: []*pb.InventoryMovement{mov}}, nil
}

func (r *SQLServerInventoryMovementRepository) ReadInventoryMovement(ctx context.Context, req *pb.ReadInventoryMovementRequest) (*pb.ReadInventoryMovementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory movement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	mov := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, mov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.ReadInventoryMovementResponse{Data: []*pb.InventoryMovement{mov}}, nil
}

func (r *SQLServerInventoryMovementRepository) UpdateInventoryMovement(ctx context.Context, req *pb.UpdateInventoryMovementRequest) (*pb.UpdateInventoryMovementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
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
		return nil, fmt.Errorf("failed to update inventory movement: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	mov := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, mov); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.UpdateInventoryMovementResponse{Data: []*pb.InventoryMovement{mov}}, nil
}

func (r *SQLServerInventoryMovementRepository) DeleteInventoryMovement(ctx context.Context, req *pb.DeleteInventoryMovementRequest) (*pb.DeleteInventoryMovementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory movement: %w", err)
	}
	return &pb.DeleteInventoryMovementResponse{Success: true}, nil
}

func (r *SQLServerInventoryMovementRepository) ListInventoryMovements(ctx context.Context, req *pb.ListInventoryMovementsRequest) (*pb.ListInventoryMovementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory movements: %w", err)
	}
	var movements []*pb.InventoryMovement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		mov := &pb.InventoryMovement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, mov); err != nil {
			continue
		}
		movements = append(movements, mov)
	}
	return &pb.ListInventoryMovementsResponse{Data: movements}, nil
}

// GetInventoryMovementListPageData retrieves inventory movements with pagination.
//
// SQL Server differences vs postgres:
//   - $N → @pN; active = true → active = 1; ILIKE → LIKE.
//   - Pagination: OFFSET/FETCH; ORDER BY required.
//   - workspace_id filter (multi-tenancy guardrail).
//   - Sort whitelist validated by BuildOrderBy (A2 guard).
func (r *SQLServerInventoryMovementRepository) GetInventoryMovementListPageData(
	ctx context.Context,
	req *pb.GetInventoryMovementListPageDataRequest,
) (*pb.GetInventoryMovementListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	// Validate sort column against whitelist (A2 guard).
	orderByClause, err := sqlserverCore.BuildOrderBy(inventoryMovementSortableSQLColsSS, req.GetSort(), "date_created DESC")
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

	offsetIdx := 2
	limitIdx := 3
	queryArgs := []any{workspaceID, offset, limit}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				im.id,
				im.date_created,
				im.date_modified,
				im.active,
				im.inventory_item_id,
				im.quantity,
				im.unit_cost,
				im.movement_date,
				im.movement_type,
				im.reference_id,
				im.reference_type,
				im.notes
			FROM inventory_movement im
			WHERE im.workspace_id = @p1 AND im.active = 1
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, orderByClause, offsetIdx, limitIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory movement list: %w", err)
	}
	defer rows.Close()

	var movements []*pb.InventoryMovement
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     sql.NullTime
			dateModified    sql.NullTime
			active          bool
			inventoryItemID sql.NullString
			quantity        float64
			unitCost        int64
			movementDate    sql.NullTime
			movementType    string
			referenceID     sql.NullString
			referenceType   sql.NullString
			notes           sql.NullString
			total           int64
		)
		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active,
			&inventoryItemID, &quantity, &unitCost, &movementDate,
			&movementType, &referenceID, &referenceType, &notes, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan inventory movement row: %w", err)
		}
		totalCount = total

		mov := &pb.InventoryMovement{
			Id:           id,
			Active:       active,
			Quantity:     quantity,
			UnitCost:     unitCost,
			MovementType: enumspb.MovementType(enumspb.MovementType_value[movementType]),
		}
		if inventoryItemID.Valid {
			mov.InventoryItemId = &inventoryItemID.String
		}
		if referenceID.Valid {
			mov.ReferenceId = &referenceID.String
		}
		if referenceType.Valid {
			mov.ReferenceType = &referenceType.String
		}
		if notes.Valid {
			mov.Notes = &notes.String
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.UnixMilli()
			mov.DateCreated = &ts
		}
		movements = append(movements, mov)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory movement rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetInventoryMovementListPageDataResponse{
		InventoryMovementList: movements,
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

func (r *SQLServerInventoryMovementRepository) GetInventoryMovementItemPageData(ctx context.Context, req *pb.GetInventoryMovementItemPageDataRequest) (*pb.GetInventoryMovementItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetInventoryMovementItemPageData not yet implemented")
}
