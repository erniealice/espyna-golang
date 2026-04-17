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
	registry.RegisterRepositoryFactory("postgresql", entityid.FulfillmentStatusEvent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fulfillment_status_event repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFulfillmentStatusEventRepository(dbOps, tableName), nil
	})
}

// PostgresFulfillmentStatusEventRepository handles the append-only fulfillment status event log.
// id is BIGSERIAL — never generated here, always assigned by the DB.
// There is no Update and no Delete — this table is append-only.
type PostgresFulfillmentStatusEventRepository struct {
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFulfillmentStatusEventRepository creates a new PostgreSQL fulfillment_status_event repository
func NewPostgresFulfillmentStatusEventRepository(dbOps interfaces.DatabaseOperation, tableName string) *PostgresFulfillmentStatusEventRepository {
	if tableName == "" {
		tableName = "fulfillment_status_event"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresFulfillmentStatusEventRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// InsertStatusEvent appends a new status event to the log.
// The id is BIGSERIAL — the DB auto-assigns it; do not pass an id in the request.
func (r *PostgresFulfillmentStatusEventRepository) InsertStatusEvent(ctx context.Context, evt *pb.FulfillmentStatusEvent) (*pb.FulfillmentStatusEvent, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, fulfillment_id, from_status, to_status, provider_status, provider_reference,
		          triggered_by_id, reason, occurred_at
	`

	var fromStatus, triggeredByID *string
	if evt.FromStatus != nil {
		fromStatus = evt.FromStatus
	}
	if evt.TriggeredById != nil {
		triggeredByID = evt.TriggeredById
	}

	row := r.db.QueryRowContext(ctx, query,
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
func (r *PostgresFulfillmentStatusEventRepository) ListStatusEvents(ctx context.Context, fulfillmentID string) ([]*pb.FulfillmentStatusEvent, error) {
	if fulfillmentID == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, from_status, to_status, provider_status, provider_reference,
		       triggered_by_id, reason, occurred_at
		FROM fulfillment_status_event
		WHERE fulfillment_id = $1
		ORDER BY occurred_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, fulfillmentID)
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