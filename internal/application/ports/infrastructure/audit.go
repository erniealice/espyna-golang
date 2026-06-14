package infrastructure

import (
	"context"
	"fmt"
)

// AuditService writes and queries audit log entries.
// Implementations must write using direct SQL only — never via PostgresOperations.Create()
// (that would trigger another LogEntry call → infinite recursion).
type AuditService interface {
	// LogEntry writes one audit entry and its associated field changes.
	// Must be called within the same transaction as the business operation.
	LogEntry(ctx context.Context, entry *AuditLogRequest) error

	// ListByEntity returns audit entries for a specific entity, newest first.
	// Uses cursor pagination on (occurred_at, id).
	ListByEntity(ctx context.Context, req *ListAuditRequest) (*ListAuditResponse, error)

	// ListByActor returns audit entries for a specific actor, newest first,
	// optionally filtered by a use_case prefix (e.g. "switch_").
	// Used by the /me/recent-activity view.
	ListByActor(ctx context.Context, req *ListByActorRequest) (*ListAuditResponse, error)
}

// AuditLogRequest contains all data for one audit event.
type AuditLogRequest struct {
	WorkspaceID    string // tenant scope — required for multi-tenant queries
	EntityType     string
	EntityID       string
	Domain         string // "centymo", "entydad", "fycha", "fayna"
	Action         int32  // AuditAction enum value
	PermissionCode string // matches permission.permission_code
	UseCase        string // "SuspendClient", "AdjustInventory"
	Reason         string // optional user-supplied
	MethodName     string // Go method name (debug)
	FieldChanges   []AuditFieldChange
}

// AuditFieldChange represents one field-level change.
type AuditFieldChange struct {
	FieldName string
	FieldType int32  // FieldType enum value
	OldValue  string // canonical string serialization
	NewValue  string // canonical string serialization
}

// ListAuditRequest is the query parameters for listing audit entries.
type ListAuditRequest struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	Limit       int
	CursorToken string // opaque cursor for keyset pagination
}

// ListByActorRequest is the query parameters for listing audit entries by actor.
// UseCasePrefix filters to rows whose use_case starts with the given string
// (e.g. "switch_"). Empty prefix means no filter.
type ListByActorRequest struct {
	ActorID        string
	UseCasePrefix  string
	Limit          int
}

// ListAuditResponse is the paginated result.
type ListAuditResponse struct {
	Entries    []AuditEntryResult
	HasNext    bool
	NextCursor string
}

// AuditEntryResult is a single audit entry with its field changes.
type AuditEntryResult struct {
	ID             string
	WorkspaceID    string
	ActorID        string
	ActorType      int32
	EntityType     string
	EntityID       string
	Domain         string
	Action         int32
	PermissionCode string
	UseCase        string
	Reason         string
	MethodName     string
	RequestID      string
	RequestURL     string // HTTP request URL (populated by P3+P7 forensic logging)
	Referer        string // HTTP Referer header (populated by P3+P7 forensic logging)
	FieldCount     int32
	OccurredAt     string // RFC3339 UTC
	FieldChanges   []AuditFieldChange
}

// DiffAndLogRequest carries all metadata needed to compute a field-level
// diff and write one audit log entry.
type DiffAndLogRequest struct {
	WorkspaceID    string
	EntityType     string
	EntityID       string
	Domain         string          // "centymo", "entydad", "fycha", "fayna"
	Action         int32           // AuditAction enum value: 1=INSERT, 2=UPDATE, 3=DELETE
	PermissionCode string          // "client:update"
	UseCase        string          // "SuspendClient"
	Reason         string          // optional
	MethodName     string          // "UpdateClientStatus"
	OldData        map[string]any  // nil for INSERT
	NewData        map[string]any  // nil for DELETE
	ExcludedFields map[string]bool // fields to skip (e.g. password_hash)
}

// DiffAndLog computes field-level changes between OldData and NewData,
// filters excluded fields, and calls svc.LogEntry.
// Returns nil immediately if svc is nil (audit disabled).
func DiffAndLog(ctx context.Context, svc AuditService, req DiffAndLogRequest) error {
	if svc == nil {
		return nil
	}

	var changes []AuditFieldChange

	switch {
	case req.Action == 1: // INSERT
		for k, v := range req.NewData {
			if req.ExcludedFields[k] {
				continue
			}
			changes = append(changes, AuditFieldChange{
				FieldName: k,
				FieldType: 1,
				OldValue:  "",
				NewValue:  serializeValue(v),
			})
		}
	case req.Action == 2: // UPDATE
		for k, newVal := range req.NewData {
			if req.ExcludedFields[k] {
				continue
			}
			oldVal := req.OldData[k]
			oldStr := serializeValue(oldVal)
			newStr := serializeValue(newVal)
			if oldStr == newStr {
				continue
			}
			changes = append(changes, AuditFieldChange{
				FieldName: k,
				FieldType: 1,
				OldValue:  oldStr,
				NewValue:  newStr,
			})
		}
		// Capture fields removed on UPDATE (present in old, absent in new)
		if req.OldData != nil {
			for k, v := range req.OldData {
				if req.ExcludedFields[k] {
					continue
				}
				if _, exists := req.NewData[k]; !exists {
					changes = append(changes, AuditFieldChange{
						FieldName: k,
						FieldType: 1,
						OldValue:  serializeValue(v),
						NewValue:  "",
					})
				}
			}
		}
	case req.Action == 3: // DELETE
		for k, v := range req.OldData {
			if req.ExcludedFields[k] {
				continue
			}
			changes = append(changes, AuditFieldChange{
				FieldName: k,
				FieldType: 1,
				OldValue:  serializeValue(v),
				NewValue:  "",
			})
		}
	}

	return svc.LogEntry(ctx, &AuditLogRequest{
		WorkspaceID:    req.WorkspaceID,
		EntityType:     req.EntityType,
		EntityID:       req.EntityID,
		Domain:         req.Domain,
		Action:         req.Action,
		PermissionCode: req.PermissionCode,
		UseCase:        req.UseCase,
		Reason:         req.Reason,
		MethodName:     req.MethodName,
		FieldChanges:   changes,
	})
}

func serializeValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// NoOpAuditService is used when audit logging is disabled.
type NoOpAuditService struct{}

func NewNoOpAuditService() AuditService { return &NoOpAuditService{} }

func (s *NoOpAuditService) LogEntry(_ context.Context, _ *AuditLogRequest) error {
	return nil
}
func (s *NoOpAuditService) ListByEntity(_ context.Context, _ *ListAuditRequest) (*ListAuditResponse, error) {
	return &ListAuditResponse{}, nil
}
func (s *NoOpAuditService) ListByActor(_ context.Context, _ *ListByActorRequest) (*ListAuditResponse, error) {
	return &ListAuditResponse{}, nil
}
