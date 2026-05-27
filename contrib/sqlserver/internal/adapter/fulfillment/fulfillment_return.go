//go:build sqlserver

package fulfillment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.FulfillmentReturn, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_return repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerFulfillmentReturnRepository(dbOps, tableName), nil
	})
}

// SQLServerFulfillmentReturnRepository handles CRUD for fulfillment return records.
// Soft delete is via active=false.
//
// SQL Server differences vs postgres gold standard:
//   - $1/$2 → @p1/@p2
//   - active = true → active = 1
//   - RETURNING → OUTPUT inserted.*
//   - LIMIT n → TOP 1 / OFFSET/FETCH
type SQLServerFulfillmentReturnRepository struct {
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerFulfillmentReturnRepository creates a new SQL Server fulfillment_return repository.
func NewSQLServerFulfillmentReturnRepository(dbOps interfaces.DatabaseOperation, tableName string) *SQLServerFulfillmentReturnRepository {
	if tableName == "" {
		tableName = "fulfillment_return"
	}
	return &SQLServerFulfillmentReturnRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFulfillmentReturn creates a new fulfillment return record.
func (r *SQLServerFulfillmentReturnRepository) CreateFulfillmentReturn(ctx context.Context, ret *pb.FulfillmentReturn) (*pb.FulfillmentReturn, error) {
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment return: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	created := &pb.FulfillmentReturn{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, created); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return created, nil
}

// GetFulfillmentReturn retrieves a fulfillment return by ID.
//
// SQL Server differences vs postgres: @p1 placeholder; active = 1.
func (r *SQLServerFulfillmentReturnRepository) GetFulfillmentReturn(ctx context.Context, id string) (*pb.FulfillmentReturn, error) {
	if id == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		SELECT id, fulfillment_id, reason, status, refund_amount, currency,
		       processed_by_id, notes, active, date_created, completed_at
		FROM fulfillment_return
		WHERE id = @p1 AND active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, id)

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
		refundAmtInt := int64(refundAmount.Float64)
		ret.RefundAmount = &refundAmtInt
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

// UpdateFulfillmentReturn updates mutable fields: status, refund_amount, processed_by_id, completed_at.
//
// SQL Server differences vs postgres: @pN placeholders; active = 1.
func (r *SQLServerFulfillmentReturnRepository) UpdateFulfillmentReturn(ctx context.Context, ret *pb.FulfillmentReturn) (*pb.FulfillmentReturn, error) {
	if ret == nil || ret.Id == "" {
		return nil, fmt.Errorf("fulfillment return ID is required")
	}

	query := `
		UPDATE fulfillment_return
		SET status = @p1,
		    refund_amount = @p2,
		    processed_by_id = @p3,
		    completed_at = @p4
		WHERE id = @p5 AND active = 1
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

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	_, err := exec.ExecContext(ctx, query,
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

// DeleteFulfillmentReturn soft-deletes a fulfillment return (SET active=0).
//
// SQL Server differences vs postgres: @p1 placeholder; active = false → active = 0.
func (r *SQLServerFulfillmentReturnRepository) DeleteFulfillmentReturn(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("fulfillment return ID is required")
	}

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	_, err := exec.ExecContext(ctx,
		`UPDATE fulfillment_return SET active = 0 WHERE id = @p1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete fulfillment return: %w", err)
	}
	return nil
}

// ListFulfillmentReturns lists all active returns for a given fulfillment, newest first.
//
// SQL Server differences vs postgres: @p1 placeholder; active = 1.
func (r *SQLServerFulfillmentReturnRepository) ListFulfillmentReturns(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentReturn, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, reason, status, refund_amount, currency,
		       processed_by_id, notes, active, date_created, completed_at
		FROM fulfillment_return
		WHERE fulfillment_id = @p1 AND active = 1
		ORDER BY date_created DESC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, fulfillmentID)
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
			refundAmtInt := int64(refundAmount.Float64)
			ret.RefundAmount = &refundAmtInt
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

// listFulfillmentReturnsSS is unexported in fulfillment package to avoid conflict
// with fulfillment_item's List method.
var _ interfaces.DatabaseOperation = nil // compile-time interface assertion suppressed
