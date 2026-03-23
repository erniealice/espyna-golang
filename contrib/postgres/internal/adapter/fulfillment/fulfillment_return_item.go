//go:build postgresql

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.FulfillmentReturnItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fulfillment_return_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresFulfillmentReturnItemRepository(dbOps, tableName), nil
	})
}

// PostgresFulfillmentReturnItemRepository handles immutable return line items.
// Return items are created once and never updated or deleted.
type PostgresFulfillmentReturnItemRepository struct {
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFulfillmentReturnItemRepository creates a new PostgreSQL fulfillment_return_item repository
func NewPostgresFulfillmentReturnItemRepository(dbOps interfaces.DatabaseOperation, tableName string) *PostgresFulfillmentReturnItemRepository {
	if tableName == "" {
		tableName = "fulfillment_return_item"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresFulfillmentReturnItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillmentReturnItem inserts a new return line item.
// Return items are immutable once created — no Update, no Delete.
func (r *PostgresFulfillmentReturnItemRepository) CreateFulfillmentReturnItem(ctx context.Context, item *pb.FulfillmentReturnItem) (*pb.FulfillmentReturnItem, error) {
	if item == nil {
		return nil, fmt.Errorf("fulfillment return item data is required")
	}
	if item.FulfillmentReturnId == "" {
		return nil, fmt.Errorf("fulfillment_return_id is required")
	}
	if item.FulfillmentItemId == "" {
		return nil, fmt.Errorf("fulfillment_item_id is required")
	}

	query := `
		INSERT INTO fulfillment_return_item
			(id, fulfillment_return_id, fulfillment_item_id, quantity_returned, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, fulfillment_return_id, fulfillment_item_id, quantity_returned, reason, date_created
	`

	row := r.db.QueryRowContext(ctx, query,
		item.Id,
		item.FulfillmentReturnId,
		item.FulfillmentItemId,
		item.QuantityReturned,
		item.Reason,
	)

	created := &pb.FulfillmentReturnItem{}
	var dateCreated time.Time
	err := row.Scan(
		&created.Id,
		&created.FulfillmentReturnId,
		&created.FulfillmentItemId,
		&created.QuantityReturned,
		&created.Reason,
		&dateCreated,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment return item: %w", err)
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		created.DateCreated = &ts
	}

	return created, nil
}

// ListFulfillmentReturnItems returns all items for a given fulfillment_return_id, ordered by id ASC.
func (r *PostgresFulfillmentReturnItemRepository) ListFulfillmentReturnItems(ctx context.Context, fulfillmentReturnID string) ([]*pb.FulfillmentReturnItem, error) {
	if fulfillmentReturnID == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		SELECT id, fulfillment_return_id, fulfillment_item_id, quantity_returned, reason, date_created
		FROM fulfillment_return_item
		WHERE fulfillment_return_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, fulfillmentReturnID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillment return items: %w", err)
	}
	defer rows.Close()

	var items []*pb.FulfillmentReturnItem
	for rows.Next() {
		var (
			id                  string
			fulfillmentReturnID string
			fulfillmentItemID   string
			quantityReturned    float64
			reason              string
			dateCreated         time.Time
		)
		if err := rows.Scan(
			&id, &fulfillmentReturnID, &fulfillmentItemID, &quantityReturned, &reason, &dateCreated,
		); err != nil {
			log.Printf("WARN: scan fulfillment_return_item row: %v", err)
			continue
		}
		item := &pb.FulfillmentReturnItem{
			Id:                  id,
			FulfillmentReturnId: fulfillmentReturnID,
			FulfillmentItemId:   fulfillmentItemID,
			QuantityReturned:    quantityReturned,
			Reason:              reason,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_return_item rows: %w", err)
	}

	return items, nil
}
