//go:build mysql

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.FulfillmentItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql fulfillment_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLFulfillmentItemRepository(dbOps, tableName), nil
	})
}

// MySQLFulfillmentItemRepository handles CRUD for fulfillment line items.
// Items are deleted via CASCADE when their parent fulfillment is deleted — no Delete method here.
type MySQLFulfillmentItemRepository struct {
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLFulfillmentItemRepository creates a new MySQL fulfillment_item repository.
func NewMySQLFulfillmentItemRepository(dbOps interfaces.DatabaseOperation, tableName string) *MySQLFulfillmentItemRepository {
	if tableName == "" {
		tableName = "fulfillment_item"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLFulfillmentItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillmentItem inserts a new fulfillment line item.
//
// Dialect translation from postgres gold standard:
//   - INSERT ... RETURNING → two-step: INSERT then SELECT (MySQL has no RETURNING)
//   - $1..$11 → ? (positional)
//   - quantity_remaining is a DB GENERATED ALWAYS column — must not be written
func (r *MySQLFulfillmentItemRepository) CreateFulfillmentItem(ctx context.Context, item *pb.FulfillmentItem) (*pb.FulfillmentItem, error) {
	if item == nil {
		return nil, fmt.Errorf("fulfillment item data is required")
	}
	if item.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment_id is required")
	}

	var sourceType, sourceID *string
	if item.SourceType != nil {
		sourceType = item.SourceType
	}
	if item.SourceId != nil {
		sourceID = item.SourceId
	}

	// Dialect: no RETURNING — INSERT then SELECT back by id.
	insertQuery := `
		INSERT INTO fulfillment_item
			(id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
			 source_type, source_id, quantity_ordered, quantity_delivered, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, insertQuery,
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
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment item: %w", err)
	}

	// Re-read the inserted row (MySQL two-step for INSERT ... RETURNING equivalent).
	selectQuery := `
		SELECT id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
		       source_type, source_id, quantity_ordered, quantity_delivered, status, notes
		FROM fulfillment_item
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, selectQuery, item.Id)

	created := &pb.FulfillmentItem{}
	var srcType, srcID sql.NullString
	err = row.Scan(
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
		return nil, fmt.Errorf("failed to read back created fulfillment item: %w", err)
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
//
// Dialect: $1 → ?.
func (r *MySQLFulfillmentItemRepository) ListFulfillmentItems(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentItem, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	// Dialect: $1 → ?
	query := `
		SELECT id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
		       source_type, source_id, quantity_ordered, quantity_delivered, status, notes
		FROM fulfillment_item
		WHERE fulfillment_id = ?
		ORDER BY id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, fulfillmentID)
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
//
// Dialect: $1, $2 → ?.
func (r *MySQLFulfillmentItemRepository) UpdateFulfillmentItemDelivered(ctx context.Context, id string, quantityDelivered float64) error {
	if id == "" {
		return fmt.Errorf("fulfillment item ID is required")
	}

	// Dialect: $1, $2 → ?
	_, err := r.db.ExecContext(ctx,
		`UPDATE fulfillment_item SET quantity_delivered = ? WHERE id = ?`,
		quantityDelivered, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update fulfillment item delivered quantity: %w", err)
	}
	return nil
}
