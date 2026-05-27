//go:build sqlserver

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.FulfillmentItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerFulfillmentItemRepository(dbOps, tableName), nil
	})
}

// SQLServerFulfillmentItemRepository handles CRUD for fulfillment line items.
// Items are deleted via CASCADE when their parent fulfillment is deleted.
type SQLServerFulfillmentItemRepository struct {
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerFulfillmentItemRepository creates a new SQL Server fulfillment_item repository.
func NewSQLServerFulfillmentItemRepository(dbOps interfaces.DatabaseOperation, tableName string) *SQLServerFulfillmentItemRepository {
	if tableName == "" {
		tableName = "fulfillment_item"
	}
	return &SQLServerFulfillmentItemRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerFulfillmentItemRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateFulfillmentItem inserts a new fulfillment line item.
// SQL Server: OUTPUT inserted.* instead of RETURNING; @pN placeholders.
func (r *SQLServerFulfillmentItemRepository) CreateFulfillmentItem(ctx context.Context, item *pb.FulfillmentItem) (*pb.FulfillmentItem, error) {
	if item == nil {
		return nil, fmt.Errorf("fulfillment item data is required")
	}
	if item.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment_id is required")
	}

	// SQL Server: INSERT ... OUTPUT inserted.<cols> (no RETURNING).
	const query = `
		INSERT INTO fulfillment_item
			(id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
			 source_type, source_id, quantity_ordered, quantity_delivered, status, notes)
		OUTPUT
			inserted.id, inserted.fulfillment_id, inserted.revenue_line_item_id,
			inserted.product_id, inserted.delivery_mode,
			inserted.source_type, inserted.source_id,
			inserted.quantity_ordered, inserted.quantity_delivered,
			inserted.status, inserted.notes
		VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11)
	`

	var sourceType, sourceID *string
	if item.SourceType != nil {
		sourceType = item.SourceType
	}
	if item.SourceId != nil {
		sourceID = item.SourceId
	}

	exec := r.getExec(ctx)
	row := exec.QueryRowContext(ctx, query,
		item.Id,
		item.FulfillmentId,
		item.RevenueLineItemId,
		item.ProductId,
		item.DeliveryMode,
		sourceType,
		sourceID,
		item.QuantityOrdered,
		item.QuantityDelivered,
		item.Status,
		item.Notes,
	)

	created := &pb.FulfillmentItem{}
	var srcType, srcID sql.NullString
	err := row.Scan(
		&created.Id,
		&created.FulfillmentId,
		&created.RevenueLineItemId,
		&created.ProductId,
		&created.DeliveryMode,
		&srcType,
		&srcID,
		&created.QuantityOrdered,
		&created.QuantityDelivered,
		&created.Status,
		&created.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment item: %w", err)
	}
	if srcType.Valid {
		created.SourceType = &srcType.String
	}
	if srcID.Valid {
		created.SourceId = &srcID.String
	}

	return created, nil
}

// ListFulfillmentItems returns all items for a given fulfillment, ordered by id ASC.
// SQL Server: @p1 placeholder.
func (r *SQLServerFulfillmentItemRepository) ListFulfillmentItems(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentItem, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	const query = `
		SELECT id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
		       source_type, source_id, quantity_ordered, quantity_delivered, status, notes
		FROM fulfillment_item
		WHERE fulfillment_id = @p1
		ORDER BY id ASC
	`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, fulfillmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillment items: %w", err)
	}
	defer rows.Close()

	var items []*pb.FulfillmentItem
	for rows.Next() {
		var (
			id                string
			fID               string
			revenueLineItemID string
			productID         string
			method            string
			sourceType        sql.NullString
			sourceID          sql.NullString
			quantityOrdered   float64
			quantityDelivered float64
			status            string
			notes             string
		)
		if err := rows.Scan(
			&id, &fID, &revenueLineItemID, &productID, &method,
			&sourceType, &sourceID, &quantityOrdered, &quantityDelivered, &status, &notes,
		); err != nil {
			log.Printf("WARN: scan fulfillment_item row: %v", err)
			continue
		}
		item := &pb.FulfillmentItem{
			Id:                id,
			FulfillmentId:     fID,
			RevenueLineItemId: revenueLineItemID,
			ProductId:         productID,
			DeliveryMode:      method,
			QuantityOrdered:   quantityOrdered,
			QuantityDelivered: quantityDelivered,
			Status:            status,
			Notes:             notes,
		}
		if sourceType.Valid {
			item.SourceType = &sourceType.String
		}
		if sourceID.Valid {
			item.SourceId = &sourceID.String
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_item rows: %w", err)
	}

	return items, nil
}

// UpdateFulfillmentItemDelivered updates quantity_delivered for a fulfillment item.
// SQL Server: @p1, @p2 placeholders; GETUTCDATE() for timestamp.
func (r *SQLServerFulfillmentItemRepository) UpdateFulfillmentItemDelivered(ctx context.Context, id string, quantityDelivered float64) error {
	if id == "" {
		return fmt.Errorf("fulfillment item ID is required")
	}

	exec := r.getExec(ctx)
	_, err := exec.ExecContext(ctx,
		`UPDATE fulfillment_item SET quantity_delivered = @p1 WHERE id = @p2`,
		quantityDelivered, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update fulfillment item delivered quantity: %w", err)
	}
	return nil
}
