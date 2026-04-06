//go:build postgresql

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.FulfillmentItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fulfillment_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFulfillmentItemRepository(dbOps, tableName), nil
	})
}

// PostgresFulfillmentItemRepository handles CRUD for fulfillment line items.
// Items are deleted via CASCADE when their parent fulfillment is deleted — no Delete method here.
type PostgresFulfillmentItemRepository struct {
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFulfillmentItemRepository creates a new PostgreSQL fulfillment_item repository
func NewPostgresFulfillmentItemRepository(dbOps interfaces.DatabaseOperation, tableName string) *PostgresFulfillmentItemRepository {
	if tableName == "" {
		tableName = "fulfillment_item"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresFulfillmentItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillmentItem inserts a new fulfillment line item.
// quantity_remaining is a DB GENERATED ALWAYS column — it must not be written.
func (r *PostgresFulfillmentItemRepository) CreateFulfillmentItem(ctx context.Context, item *pb.FulfillmentItem) (*pb.FulfillmentItem, error) {
	if item == nil {
		return nil, fmt.Errorf("fulfillment item data is required")
	}
	if item.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment_id is required")
	}

	query := `
		INSERT INTO fulfillment_item
			(id, fulfillment_id, revenue_line_item_id, product_id, fulfillment_method,
			 source_type, source_id, quantity_ordered, quantity_delivered, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, fulfillment_id, revenue_line_item_id, product_id, fulfillment_method,
		          source_type, source_id, quantity_ordered, quantity_delivered, status, notes
	`

	var sourceType, sourceID *string
	if item.SourceType != nil {
		sourceType = item.SourceType
	}
	if item.SourceId != nil {
		sourceID = item.SourceId
	}

	row := r.db.QueryRowContext(ctx, query,
		item.Id,
		item.FulfillmentId,
		item.RevenueLineItemId,
		item.ProductId,
		item.FulfillmentMethod,
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
		&created.FulfillmentMethod,
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
// quantity_remaining is read from the DB generated column.
func (r *PostgresFulfillmentItemRepository) ListFulfillmentItems(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentItem, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, revenue_line_item_id, product_id, fulfillment_method,
		       source_type, source_id, quantity_ordered, quantity_delivered, status, notes
		FROM fulfillment_item
		WHERE fulfillment_id = $1
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
			FulfillmentMethod: method,
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
// quantity_remaining is a generated column and must not be written directly.
func (r *PostgresFulfillmentItemRepository) UpdateFulfillmentItemDelivered(ctx context.Context, id string, quantityDelivered float64) error {
	if id == "" {
		return fmt.Errorf("fulfillment item ID is required")
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE fulfillment_item SET quantity_delivered = $1 WHERE id = $2`,
		quantityDelivered, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update fulfillment item delivered quantity: %w", err)
	}
	return nil
}

