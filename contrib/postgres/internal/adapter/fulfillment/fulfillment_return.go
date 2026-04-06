//go:build postgresql

package fulfillment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.FulfillmentReturn, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fulfillment_return repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFulfillmentReturnRepository(dbOps, tableName), nil
	})
}

// PostgresFulfillmentReturnRepository handles CRUD for fulfillment return records.
// Soft delete is via active=false.
type PostgresFulfillmentReturnRepository struct {
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFulfillmentReturnRepository creates a new PostgreSQL fulfillment_return repository
func NewPostgresFulfillmentReturnRepository(dbOps interfaces.DatabaseOperation, tableName string) *PostgresFulfillmentReturnRepository {
	if tableName == "" {
		tableName = "fulfillment_return"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresFulfillmentReturnRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillmentReturn creates a new fulfillment return record
func (r *PostgresFulfillmentReturnRepository) CreateFulfillmentReturn(ctx context.Context, ret *pb.FulfillmentReturn) (*pb.FulfillmentReturn, error) {
	if ret == nil {
		return nil, fmt.Errorf("fulfillment return data is required")
	}
	if ret.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment_id is required")
	}

	jsonData, err := protojson.Marshal(ret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment return: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	created := &pb.FulfillmentReturn{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, created); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return created, nil
}

// GetFulfillmentReturn retrieves a fulfillment return by ID
func (r *PostgresFulfillmentReturnRepository) GetFulfillmentReturn(ctx context.Context, id string) (*pb.FulfillmentReturn, error) {
	if id == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		SELECT id, fulfillment_id, reason, status, refund_amount, currency,
		       processed_by_id, notes, active, date_created, completed_at
		FROM fulfillment_return
		WHERE id = $1 AND active = true
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var (
		retID         string
		fulfillmentID string
		reason        string
		status        string
		refundAmount  sql.NullFloat64
		currency      string
		processedByID sql.NullString
		notes         string
		active        bool
		dateCreated   time.Time
		completedAt   sql.NullTime
	)

	err := row.Scan(
		&retID, &fulfillmentID, &reason, &status, &refundAmount, &currency,
		&processedByID, &notes, &active, &dateCreated, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment return with ID '%s' not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read fulfillment return: %w", err)
	}

	ret := &pb.FulfillmentReturn{
		Id:            retID,
		FulfillmentId: fulfillmentID,
		Reason:        reason,
		Status:        status,
		Currency:      currency,
		Notes:         notes,
		Active:        active,
	}
	if refundAmount.Valid {
		ret.RefundAmount = &refundAmount.Float64
	}
	if processedByID.Valid {
		ret.ProcessedById = &processedByID.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		ret.DateCreated = &ts
	}

	return ret, nil
}

// UpdateFulfillmentReturn updates mutable fields: status, refund_amount, processed_by_id, completed_at
func (r *PostgresFulfillmentReturnRepository) UpdateFulfillmentReturn(ctx context.Context, ret *pb.FulfillmentReturn) (*pb.FulfillmentReturn, error) {
	if ret == nil || ret.Id == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		UPDATE fulfillment_return
		SET status = $1,
		    refund_amount = $2,
		    processed_by_id = $3,
		    completed_at = $4
		WHERE id = $5 AND active = true
	`

	var refundAmount *int64
	if ret.RefundAmount != nil {
		refundAmount = ret.RefundAmount
	}

	var processedByID *string
	if ret.ProcessedById != nil {
		processedByID = ret.ProcessedById
	}

	var completedAt *time.Time
	if ret.CompletedAt != nil {
		t := ret.CompletedAt.AsTime()
		completedAt = &t
	}

	_, err := r.db.ExecContext(ctx, query,
		ret.Status,
		refundAmount,
		processedByID,
		completedAt,
		ret.Id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update fulfillment return: %w", err)
	}

	return r.GetFulfillmentReturn(ctx, ret.Id)
}

// DeleteFulfillmentReturn soft-deletes a fulfillment return (SET active=false)
func (r *PostgresFulfillmentReturnRepository) DeleteFulfillmentReturn(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("fulfillment return ID is required")
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE fulfillment_return SET active = false WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete fulfillment return: %w", err)
	}
	return nil
}

// ListFulfillmentReturns lists all active returns for a given fulfillment, newest first
func (r *PostgresFulfillmentReturnRepository) ListFulfillmentReturns(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentReturn, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, reason, status, refund_amount, currency,
		       processed_by_id, notes, active, date_created, completed_at
		FROM fulfillment_return
		WHERE fulfillment_id = $1 AND active = true
		ORDER BY date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, fulfillmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillment returns: %w", err)
	}
	defer rows.Close()

	var returns []*pb.FulfillmentReturn
	for rows.Next() {
		var (
			retID         string
			fID           string
			reason        string
			status        string
			refundAmount  sql.NullFloat64
			currency      string
			processedByID sql.NullString
			notes         string
			active        bool
			dateCreated   time.Time
			completedAt   sql.NullTime
		)
		if err := rows.Scan(
			&retID, &fID, &reason, &status, &refundAmount, &currency,
			&processedByID, &notes, &active, &dateCreated, &completedAt,
		); err != nil {
			log.Printf("WARN: scan fulfillment_return row: %v", err)
			continue
		}
		ret := &pb.FulfillmentReturn{
			Id:            retID,
			FulfillmentId: fID,
			Reason:        reason,
			Status:        status,
			Currency:      currency,
			Notes:         notes,
			Active:        active,
		}
		if refundAmount.Valid {
			ret.RefundAmount = &refundAmount.Float64
		}
		if processedByID.Valid {
			ret.ProcessedById = &processedByID.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			ret.DateCreated = &ts
		}
		returns = append(returns, ret)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_return rows: %w", err)
	}

	return returns, nil
}
