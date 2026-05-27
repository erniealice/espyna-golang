//go:build sqlserver

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.FulfillmentReturnItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_return_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		ep, ok := dbOps.(executorProvider)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_return_item: dbOps does not implement executorProvider")
		}
		return NewSQLServerFulfillmentReturnItemRepository(ep, tableName), nil
	})
}

// SQLServerFulfillmentReturnItemRepository handles immutable return line items.
// Return items are created once and never updated or deleted.
//
// SQL Server differences vs postgres gold standard:
//   - $N → @pN
//   - INSERT ... RETURNING → INSERT ... OUTPUT inserted.*
//   - ORDER BY id ASC unchanged (column, not placeholder)
type SQLServerFulfillmentReturnItemRepository struct {
	dbOps     executorProvider
	tableName string
}

// NewSQLServerFulfillmentReturnItemRepository creates a new SQL Server fulfillment_return_item repository.
func NewSQLServerFulfillmentReturnItemRepository(dbOps executorProvider, tableName string) *SQLServerFulfillmentReturnItemRepository {
	if tableName == "" {
		tableName = "fulfillment_return_item"
	}
	return &SQLServerFulfillmentReturnItemRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFulfillmentReturnItem inserts a new return line item.
// Return items are immutable once created — no Update, no Delete.
//
// SQL Server: INSERT ... OUTPUT inserted.* instead of RETURNING.
// Placeholders: @p1..@p5 instead of $1..$5.
func (r *SQLServerFulfillmentReturnItemRepository) CreateFulfillmentReturnItem(ctx context.Context, item *pb.FulfillmentReturnItem) (*pb.FulfillmentReturnItem, error) {
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
		OUTPUT inserted.id, inserted.fulfillment_return_id, inserted.fulfillment_item_id,
		       inserted.quantity_returned, inserted.reason, inserted.date_created
		VALUES (@p1, @p2, @p3, @p4, @p5)
	`

	exec := r.dbOps.GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query,
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
//
// SQL Server differences vs postgres: @p1 placeholder.
func (r *SQLServerFulfillmentReturnItemRepository) ListFulfillmentReturnItems(ctx context.Context, fulfillmentReturnID string) ([]*pb.FulfillmentReturnItem, error) {
	if fulfillmentReturnID == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		SELECT id, fulfillment_return_id, fulfillment_item_id, quantity_returned, reason, date_created
		FROM fulfillment_return_item
		WHERE fulfillment_return_id = @p1
		ORDER BY id ASC
	`

	exec := r.dbOps.GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, fulfillmentReturnID)
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
