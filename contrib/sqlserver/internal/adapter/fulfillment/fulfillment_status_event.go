//go:build sqlserver

package fulfillment

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.FulfillmentStatusEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_status_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		ep, ok := dbOps.(executorProvider)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment_status_event: dbOps does not implement executorProvider")
		}
		return NewSQLServerFulfillmentStatusEventRepository(ep, tableName), nil
	})
}

// SQLServerFulfillmentStatusEventRepository handles the append-only fulfillment status event log.
// id is BIGINT IDENTITY — never generated here, always assigned by the DB.
// There is no Update and no Delete — this table is append-only.
//
// SQL Server differences vs postgres gold standard:
//   - id is IDENTITY (BIGINT), not BIGSERIAL.
//   - NOW() → SYSUTCDATETIME().
//   - RETURNING → OUTPUT inserted.*.
//   - $N → @pN.
type SQLServerFulfillmentStatusEventRepository struct {
	dbOps     executorProvider
	tableName string
}

// NewSQLServerFulfillmentStatusEventRepository creates a new SQL Server fulfillment_status_event repository.
func NewSQLServerFulfillmentStatusEventRepository(dbOps executorProvider, tableName string) *SQLServerFulfillmentStatusEventRepository {
	if tableName == "" {
		tableName = "fulfillment_status_event"
	}
	return &SQLServerFulfillmentStatusEventRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// InsertStatusEvent appends a new status event to the log.
// The id is IDENTITY — the DB auto-assigns it; do not pass an id in the request.
//
// SQL Server differences vs postgres:
//   - NOW() → SYSUTCDATETIME()
//   - RETURNING ... → OUTPUT inserted.*
//   - $N → @pN
func (r *SQLServerFulfillmentStatusEventRepository) InsertStatusEvent(ctx context.Context, evt *pb.FulfillmentStatusEvent) (*pb.FulfillmentStatusEvent, error) {
	if evt == nil {
		return nil, fmt.Errorf("fulfillment status event data is required")
	}
	if evt.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment_id is required")
	}
	if evt.ToStatus == "" {
		return nil, fmt.Errorf("to_status is required")
	}

	query := `
		INSERT INTO fulfillment_status_event
			(fulfillment_id, from_status, to_status, provider_status, provider_reference,
			 triggered_by_id, reason, occurred_at)
		OUTPUT inserted.id, inserted.fulfillment_id, inserted.from_status, inserted.to_status,
		       inserted.provider_status, inserted.provider_reference,
		       inserted.triggered_by_id, inserted.reason, inserted.occurred_at
		VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, SYSUTCDATETIME())
	`

	var fromStatus, triggeredByID *string
	if evt.FromStatus != nil {
		fromStatus = evt.FromStatus
	}
	if evt.TriggeredById != nil {
		triggeredByID = evt.TriggeredById
	}

	exec := r.dbOps.GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query,
		evt.FulfillmentId,
		fromStatus,
		evt.ToStatus,
		evt.ProviderStatus,
		evt.ProviderReference,
		triggeredByID,
		evt.Reason,
	)

	created := &pb.FulfillmentStatusEvent{}
	var dbFromStatus, dbTriggeredByID sql.NullString
	var dbOccurredAt sql.NullTime
	err := row.Scan(
		&created.Id,
		&created.FulfillmentId,
		&dbFromStatus,
		&created.ToStatus,
		&created.ProviderStatus,
		&created.ProviderReference,
		&dbTriggeredByID,
		&created.Reason,
		&dbOccurredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert fulfillment status event: %w", err)
	}
	if dbFromStatus.Valid {
		created.FromStatus = &dbFromStatus.String
	}
	if dbTriggeredByID.Valid {
		created.TriggeredById = &dbTriggeredByID.String
	}

	return created, nil
}

// ListStatusEvents returns all status events for a fulfillment, ordered by occurred_at DESC.
//
// SQL Server differences vs postgres: @p1 placeholder.
func (r *SQLServerFulfillmentStatusEventRepository) ListStatusEvents(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentStatusEvent, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, from_status, to_status, provider_status, provider_reference,
		       triggered_by_id, reason, occurred_at
		FROM fulfillment_status_event
		WHERE fulfillment_id = @p1
		ORDER BY occurred_at DESC
	`

	exec := r.dbOps.GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, fulfillmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillment status events: %w", err)
	}
	defer rows.Close()

	var events []*pb.FulfillmentStatusEvent
	for rows.Next() {
		var (
			eventID           int64
			fID               string
			fromStatus        sql.NullString
			toStatus          string
			providerStatus    string
			providerReference string
			triggeredByID     sql.NullString
			reason            string
			occurredAt        sql.NullTime
		)
		if err := rows.Scan(
			&eventID, &fID, &fromStatus, &toStatus, &providerStatus, &providerReference,
			&triggeredByID, &reason, &occurredAt,
		); err != nil {
			log.Printf("WARN: scan fulfillment_status_event row: %v", err)
			continue
		}
		evt := &pb.FulfillmentStatusEvent{
			Id:                eventID,
			FulfillmentId:     fID,
			ToStatus:          toStatus,
			ProviderStatus:    providerStatus,
			ProviderReference: providerReference,
			Reason:            reason,
		}
		if fromStatus.Valid {
			evt.FromStatus = &fromStatus.String
		}
		if triggeredByID.Valid {
			evt.TriggeredById = &triggeredByID.String
		}
		events = append(events, evt)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_status_event rows: %w", err)
	}

	return events, nil
}
