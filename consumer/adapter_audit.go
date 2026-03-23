package consumer

import (
	"context"
	"database/sql"

	dbinterfaces "github.com/erniealice/espyna-golang/database/interfaces"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// AuditListRequest is the query parameters for listing audit entries.
// Mirrors infraports.ListAuditRequest — kept in sync manually.
type AuditListRequest struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	Limit       int
	CursorToken string
}

// AuditFieldChangeView represents one field-level change.
// Mirrors infraports.AuditFieldChange — kept in sync manually.
type AuditFieldChangeView struct {
	FieldName string
	FieldType int32
	OldValue  string
	NewValue  string
}

// AuditEntryView is a single audit entry with its field changes.
// Mirrors infraports.AuditEntryResult — kept in sync manually.
type AuditEntryView struct {
	ID             string
	ActorID        string
	ActorType      int32
	Action         int32
	PermissionCode string
	UseCase        string
	FieldCount     int32
	OccurredAt     string
	FieldChanges   []AuditFieldChangeView
}

// AuditListResponse is the paginated result.
// Mirrors infraports.ListAuditResponse — kept in sync manually.
type AuditListResponse struct {
	Entries    []AuditEntryView
	HasNext    bool
	NextCursor string
}

// AuditService provides the ListByEntity operation for consumer apps.
// Consumer apps use this to populate auditlog.AuditOps.ListAuditHistory
// via a bridge closure at the composition root.
//
// The inner() method is intentionally unexported — it returns the underlying
// infraports.AuditService for use only within the consumer package
// (e.g. creating audit-enabled database operations).
type AuditService interface {
	ListByEntity(ctx context.Context, req *AuditListRequest) (*AuditListResponse, error)

	// inner returns the underlying infraports.AuditService.
	// Unexported: only usable within the consumer package.
	inner() infraports.AuditService
}

// auditServiceWrapper bridges infraports.AuditService → consumer.AuditService.
type auditServiceWrapper struct {
	svc infraports.AuditService
}

func (w *auditServiceWrapper) inner() infraports.AuditService {
	return w.svc
}

func (w *auditServiceWrapper) ListByEntity(ctx context.Context, req *AuditListRequest) (*AuditListResponse, error) {
	innerReq := &infraports.ListAuditRequest{
		WorkspaceID: req.WorkspaceID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		Limit:       req.Limit,
		CursorToken: req.CursorToken,
	}
	resp, err := w.svc.ListByEntity(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	out := &AuditListResponse{
		HasNext:    resp.HasNext,
		NextCursor: resp.NextCursor,
		Entries:    make([]AuditEntryView, len(resp.Entries)),
	}
	for i, e := range resp.Entries {
		changes := make([]AuditFieldChangeView, len(e.FieldChanges))
		for j, fc := range e.FieldChanges {
			changes[j] = AuditFieldChangeView{
				FieldName: fc.FieldName,
				FieldType: fc.FieldType,
				OldValue:  fc.OldValue,
				NewValue:  fc.NewValue,
			}
		}
		out.Entries[i] = AuditEntryView{
			ID:             e.ID,
			ActorID:        e.ActorID,
			ActorType:      e.ActorType,
			Action:         e.Action,
			PermissionCode: e.PermissionCode,
			UseCase:        e.UseCase,
			FieldCount:     e.FieldCount,
			OccurredAt:     e.OccurredAt,
			FieldChanges:   changes,
		}
	}
	return out, nil
}

// NewAuditService creates an AuditService backed by the registered provider.
// If no audit provider has been registered (e.g. via contrib/postgres init()),
// this returns nil — the consumer app should handle nil gracefully.
// When the postgresql build tag is active, contrib/postgres registers the
// PostgreSQL-backed audit adapter automatically via its init() function.
func NewAuditService(db *sql.DB) AuditService {
	factory, ok := internalregistry.GetAuditServiceFactory()
	if !ok {
		return nil
	}
	result := factory(db)
	if svc, ok := result.(infraports.AuditService); ok {
		return &auditServiceWrapper{svc: svc}
	}
	return nil
}

// NewDatabaseAdapterWithAudit creates a DatabaseAdapter backed by
// audit-enabled database operations. Mutations (Create/Update/Delete)
// automatically write audit trail entries using the provided auditSvc.
//
// If no audit-enabled operations provider is registered, or auditSvc is nil,
// returns nil. Consumer apps should fall back to NewDatabaseAdapterFromContainer.
func NewDatabaseAdapterWithAudit(db *sql.DB, auditSvc AuditService) *DatabaseAdapter {
	if auditSvc == nil {
		return nil
	}

	factory, ok := internalregistry.GetAuditEnabledOperationsFactory()
	if !ok {
		return nil
	}

	result := factory(db, auditSvc.inner())
	ops, ok := result.(dbinterfaces.DatabaseOperation)
	if !ok {
		return nil
	}

	return &DatabaseAdapter{ops: ops}
}
