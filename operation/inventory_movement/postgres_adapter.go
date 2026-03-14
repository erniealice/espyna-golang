//go:build postgresql

package inventory_movement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/inventory_movement"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.InventoryMovement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_movement repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventoryMovementRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryMovementRepository implements inventory_movement CRUD + custom operations using PostgreSQL
type PostgresInventoryMovementRepository struct {
	pb.UnimplementedInventoryMovementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresInventoryMovementRepository creates a new PostgreSQL inventory_movement repository
func NewPostgresInventoryMovementRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.InventoryMovementDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_movement"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresInventoryMovementRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryMovement creates a new inventory movement
func (r *PostgresInventoryMovementRepository) CreateInventoryMovement(ctx context.Context, req *pb.CreateInventoryMovementRequest) (*pb.CreateInventoryMovementResponse, error) {
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

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	movement := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, movement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateInventoryMovementResponse{
		Data:    []*pb.InventoryMovement{movement},
		Success: true,
	}, nil
}

// ReadInventoryMovement retrieves an inventory movement by ID
func (r *PostgresInventoryMovementRepository) ReadInventoryMovement(ctx context.Context, req *pb.ReadInventoryMovementRequest) (*pb.ReadInventoryMovementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory movement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	movement := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, movement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadInventoryMovementResponse{
		Data:    []*pb.InventoryMovement{movement},
		Success: true,
	}, nil
}

// UpdateInventoryMovement updates an inventory movement
func (r *PostgresInventoryMovementRepository) UpdateInventoryMovement(ctx context.Context, req *pb.UpdateInventoryMovementRequest) (*pb.UpdateInventoryMovementResponse, error) {
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

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	movement := &pb.InventoryMovement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, movement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateInventoryMovementResponse{
		Data:    []*pb.InventoryMovement{movement},
		Success: true,
	}, nil
}

// DeleteInventoryMovement soft-deletes an inventory movement
func (r *PostgresInventoryMovementRepository) DeleteInventoryMovement(ctx context.Context, req *pb.DeleteInventoryMovementRequest) (*pb.DeleteInventoryMovementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory movement: %w", err)
	}

	return &pb.DeleteInventoryMovementResponse{
		Success: true,
	}, nil
}

// ListInventoryMovements lists all active inventory movements
func (r *PostgresInventoryMovementRepository) ListInventoryMovements(ctx context.Context, req *pb.ListInventoryMovementsRequest) (*pb.ListInventoryMovementsResponse, error) {
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
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		movement := &pb.InventoryMovement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, movement); err != nil {
			continue
		}
		movements = append(movements, movement)
	}

	return &pb.ListInventoryMovementsResponse{
		Data:    movements,
		Success: true,
	}, nil
}

// GetInventoryMovementListPageData retrieves paginated, filtered, sorted movements with product + location JOINs
func (r *PostgresInventoryMovementRepository) GetInventoryMovementListPageData(ctx context.Context, req *pb.GetInventoryMovementListPageDataRequest) (*pb.GetInventoryMovementListPageDataResponse, error) {
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// JOIN with product, from_location, and to_location
	query := `
		WITH
		search_filtered AS (
			SELECT im.*
			FROM inventory_movement im
			WHERE ($1::text = '' OR im.product_id ILIKE $1 OR im.id ILIKE $1)
		),
		enriched AS (
			SELECT
				sf.id,
				sf.workspace_id,
				sf.movement_type,
				sf.product_id,
				sf.quantity,
				sf.unit_cost,
				sf.from_location_id,
				sf.to_location_id,
				sf.movement_date,
				sf.created_by,
				sf.date_created,
				sf.job_id,
				sf.job_activity_id,
				sf.inventory_item_id,
				sf.inventory_serial_id,
				sf.reference_type,
				sf.reference_id,
				sf.status,
				sf.notes,
				sf.performed_by,
				sf.active,
				jsonb_build_object(
					'id', p.id,
					'name', p.name,
					'active', p.active
				) as product,
				jsonb_build_object(
					'id', fl.id,
					'name', fl.name,
					'active', fl.active
				) as from_location,
				jsonb_build_object(
					'id', tl.id,
					'name', tl.name,
					'active', tl.active
				) as to_location
			FROM search_filtered sf
			LEFT JOIN product p ON sf.product_id = p.id AND p.active = true
			LEFT JOIN location fl ON sf.from_location_id = fl.id AND fl.active = true
			LEFT JOIN location tl ON sf.to_location_id = tl.id AND tl.active = true
		),
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'quantity' AND $5 = 'DESC' THEN quantity END DESC,
				CASE WHEN $4 = 'quantity' AND $5 = 'ASC' THEN quantity END ASC,
				CASE WHEN $4 = 'unit_cost' AND $5 = 'DESC' THEN unit_cost END DESC,
				CASE WHEN $4 = 'unit_cost' AND $5 = 'ASC' THEN unit_cost END ASC,
				CASE WHEN $4 = 'movement_date' AND $5 = 'DESC' THEN movement_date END DESC,
				CASE WHEN $4 = 'movement_date' AND $5 = 'ASC' THEN movement_date END ASC
		),
		total_count AS (
			SELECT count(*) as total FROM sorted
		)
		SELECT
			s.id,
			s.workspace_id,
			s.movement_type,
			s.product_id,
			s.quantity,
			s.unit_cost,
			s.from_location_id,
			s.to_location_id,
			s.movement_date,
			s.created_by,
			s.date_created,
			s.job_id,
			s.job_activity_id,
			s.inventory_item_id,
			s.inventory_serial_id,
			s.reference_type,
			s.reference_id,
			s.status,
			s.notes,
			s.performed_by,
			s.active,
			s.product,
			s.from_location,
			s.to_location,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available for raw SQL queries")
	}

	rows, err := r.db.QueryContext(ctx, query, searchQuery, limit, offset, sortField, sortDirection)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetInventoryMovementListPageData query: %w", err)
	}
	defer rows.Close()

	var movements []*pb.InventoryMovement
	var totalCount int32

	for rows.Next() {
		var (
			id               string
			workspaceId      sql.NullString
			movementType     string
			productId        string
			quantity         float64
			unitCost         float64
			fromLocationId   sql.NullString
			toLocationId     sql.NullString
			movementDate     sql.NullTime
			createdBy        sql.NullString
			dateCreated      sql.NullTime
			jobId            sql.NullString
			jobActivityId    sql.NullString
			inventoryItemId  sql.NullString
			inventorySerialId sql.NullString
			referenceType    sql.NullString
			referenceId      sql.NullString
			status           sql.NullString
			notes            sql.NullString
			performedBy      sql.NullString
			active           bool
			productJSON      []byte
			fromLocationJSON []byte
			toLocationJSON   []byte
			rowTotalCount    int32
		)

		err := rows.Scan(
			&id, &workspaceId, &movementType, &productId,
			&quantity, &unitCost, &fromLocationId, &toLocationId,
			&movementDate, &createdBy, &dateCreated,
			&jobId, &jobActivityId, &inventoryItemId, &inventorySerialId,
			&referenceType, &referenceId, &status, &notes, &performedBy, &active,
			&productJSON, &fromLocationJSON, &toLocationJSON, &rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory movement row: %w", err)
		}

		totalCount = rowTotalCount

		movement := &pb.InventoryMovement{
			Id:           id,
			MovementType: enumspb.MovementType(enumspb.MovementType_value[movementType]),
			ProductId:    productId,
			Quantity:     quantity,
			UnitCost:     unitCost,
			Active:       active,
		}

		if workspaceId.Valid {
			movement.WorkspaceId = &workspaceId.String
		}
		if fromLocationId.Valid {
			movement.FromLocationId = &fromLocationId.String
		}
		if toLocationId.Valid {
			movement.ToLocationId = &toLocationId.String
		}
		if movementDate.Valid {
			ts := movementDate.Time.UnixMilli()
			movement.MovementDate = &ts
		}
		if createdBy.Valid {
			movement.CreatedBy = &createdBy.String
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.UnixMilli()
			movement.DateCreated = &ts
		}
		if jobId.Valid {
			movement.JobId = &jobId.String
		}
		if jobActivityId.Valid {
			movement.JobActivityId = &jobActivityId.String
		}
		if inventoryItemId.Valid {
			movement.InventoryItemId = &inventoryItemId.String
		}
		if inventorySerialId.Valid {
			movement.InventorySerialId = &inventorySerialId.String
		}
		if referenceType.Valid {
			movement.ReferenceType = &referenceType.String
		}
		if referenceId.Valid {
			movement.ReferenceId = &referenceId.String
		}
		if status.Valid {
			movement.Status = &status.String
		}
		if notes.Valid {
			movement.Notes = &notes.String
		}
		if performedBy.Valid {
			movement.PerformedBy = &performedBy.String
		}

		movements = append(movements, movement)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory movement rows: %w", err)
	}

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetInventoryMovementListPageDataResponse{
		Success:               true,
		InventoryMovementList: movements,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetInventoryMovementItemPageData retrieves a single inventory movement with related data
func (r *PostgresInventoryMovementRepository) GetInventoryMovementItemPageData(ctx context.Context, req *pb.GetInventoryMovementItemPageDataRequest) (*pb.GetInventoryMovementItemPageDataResponse, error) {
	if req.InventoryMovementId == "" {
		return nil, fmt.Errorf("inventory movement ID is required")
	}

	readResp, err := r.ReadInventoryMovement(ctx, &pb.ReadInventoryMovementRequest{
		Data: &pb.InventoryMovement{Id: req.InventoryMovementId},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, fmt.Errorf("inventory movement not found with ID: %s", req.InventoryMovementId)
	}

	return &pb.GetInventoryMovementItemPageDataResponse{
		Success:           true,
		InventoryMovement: readResp.Data[0],
	}, nil
}

// ListByJob returns all inventory movements for a specific job_id
func (r *PostgresInventoryMovementRepository) ListByJob(ctx context.Context, req *pb.ListInventoryMovementsByJobRequest) (*pb.ListInventoryMovementsByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT id, workspace_id, movement_type, product_id,
			quantity, unit_cost, from_location_id, to_location_id,
			movement_date, created_by, date_created,
			job_id, job_activity_id, inventory_item_id, inventory_serial_id,
			reference_type, reference_id, status, notes, performed_by, active
		FROM inventory_movement
		WHERE job_id = $1
		ORDER BY date_created DESC
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := r.db.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list movements by job: %w", err)
	}
	defer rows.Close()

	var movements []*pb.InventoryMovement
	for rows.Next() {
		m, err := scanMovementRow(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movement rows: %w", err)
	}

	return &pb.ListInventoryMovementsByJobResponse{
		InventoryMovements: movements,
		Success:            true,
	}, nil
}

// ListByProduct returns all inventory movements for a specific product_id
func (r *PostgresInventoryMovementRepository) ListByProduct(ctx context.Context, req *pb.ListInventoryMovementsByProductRequest) (*pb.ListInventoryMovementsByProductResponse, error) {
	if req.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	query := `
		SELECT id, workspace_id, movement_type, product_id,
			quantity, unit_cost, from_location_id, to_location_id,
			movement_date, created_by, date_created,
			job_id, job_activity_id, inventory_item_id, inventory_serial_id,
			reference_type, reference_id, status, notes, performed_by, active
		FROM inventory_movement
		WHERE product_id = $1
		ORDER BY date_created DESC
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	rows, err := r.db.QueryContext(ctx, query, req.ProductId)
	if err != nil {
		return nil, fmt.Errorf("failed to list movements by product: %w", err)
	}
	defer rows.Close()

	var movements []*pb.InventoryMovement
	for rows.Next() {
		m, err := scanMovementRow(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movement rows: %w", err)
	}

	return &pb.ListInventoryMovementsByProductResponse{
		InventoryMovements: movements,
		Success:            true,
	}, nil
}

// scanMovementRow is a helper to scan a single inventory_movement row
func scanMovementRow(rows *sql.Rows) (*pb.InventoryMovement, error) {
	var (
		id                string
		workspaceId       sql.NullString
		movementType      string
		productId         string
		quantity          float64
		unitCost          float64
		fromLocationId    sql.NullString
		toLocationId      sql.NullString
		movementDate      sql.NullTime
		createdBy         sql.NullString
		dateCreated       sql.NullTime
		jobId             sql.NullString
		jobActivityId     sql.NullString
		inventoryItemId   sql.NullString
		inventorySerialId sql.NullString
		referenceType     sql.NullString
		referenceId       sql.NullString
		status            sql.NullString
		notes             sql.NullString
		performedBy       sql.NullString
		active            bool
	)

	err := rows.Scan(
		&id, &workspaceId, &movementType, &productId,
		&quantity, &unitCost, &fromLocationId, &toLocationId,
		&movementDate, &createdBy, &dateCreated,
		&jobId, &jobActivityId, &inventoryItemId, &inventorySerialId,
		&referenceType, &referenceId, &status, &notes, &performedBy, &active,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan movement row: %w", err)
	}

	movement := &pb.InventoryMovement{
		Id:           id,
		MovementType: enumspb.MovementType(enumspb.MovementType_value[movementType]),
		ProductId:    productId,
		Quantity:     quantity,
		UnitCost:     unitCost,
		Active:       active,
	}

	if workspaceId.Valid {
		movement.WorkspaceId = &workspaceId.String
	}
	if fromLocationId.Valid {
		movement.FromLocationId = &fromLocationId.String
	}
	if toLocationId.Valid {
		movement.ToLocationId = &toLocationId.String
	}
	if movementDate.Valid {
		ts := movementDate.Time.UnixMilli()
		movement.MovementDate = &ts
	}
	if createdBy.Valid {
		movement.CreatedBy = &createdBy.String
	}
	if dateCreated.Valid {
		ts := dateCreated.Time.UnixMilli()
		movement.DateCreated = &ts
	}
	if jobId.Valid {
		movement.JobId = &jobId.String
	}
	if jobActivityId.Valid {
		movement.JobActivityId = &jobActivityId.String
	}
	if inventoryItemId.Valid {
		movement.InventoryItemId = &inventoryItemId.String
	}
	if inventorySerialId.Valid {
		movement.InventorySerialId = &inventorySerialId.String
	}
	if referenceType.Valid {
		movement.ReferenceType = &referenceType.String
	}
	if referenceId.Valid {
		movement.ReferenceId = &referenceId.String
	}
	if status.Valid {
		movement.Status = &status.String
	}
	if notes.Valid {
		movement.Notes = &notes.String
	}
	if performedBy.Valid {
		movement.PerformedBy = &performedBy.String
	}

	return movement, nil
}
