//go:build sqlserver

package audit

import (
	"context"
	"fmt"
	"time"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
)

// DiffAndLogRequest carries all metadata needed to compute a field-level
// diff and write one audit log entry.
// This file is identical in logic to the postgres gold standard — the SQL
// dialect differences live entirely in audit_adapter.go.
type DiffAndLogRequest struct {
	WorkspaceID    string // tenant scope — required
	EntityType     string
	EntityID       string
	Domain         string         // "centymo", "entydad", "fycha", "fayna"
	Action         int32          // AuditAction enum value: 1=INSERT, 2=UPDATE, 3=DELETE
	PermissionCode string         // "client:update"
	UseCase        string         // "SuspendClient"
	Reason         string         // optional
	MethodName     string         // "UpdateClientStatus"
	OldData        map[string]any // nil for INSERT
	NewData        map[string]any // nil for DELETE
}

// DiffAndLog computes field-level changes between OldData and NewData,
// filters excluded fields, and calls auditService.LogEntry.
// Returns nil immediately if svc is nil (audit disabled).
func DiffAndLog(ctx context.Context, svc infraports.AuditService, req DiffAndLogRequest) error {
	if svc == nil {
		return nil
	}

	var changes []infraports.AuditFieldChange

	switch {
	case req.Action == 1: // INSERT — every new field is recorded as a change
		for k, v := range req.NewData {
			if IsExcluded(k) {
				continue
			}
			changes = append(changes, infraports.AuditFieldChange{
				FieldName: k,
				FieldType: 1, // STRING — type registry deferred to V2
				OldValue:  "",
				NewValue:  serializeValue(v),
			})
		}

	case req.Action == 2: // UPDATE — diff old vs new; only record changed fields
		for k, newVal := range req.NewData {
			if IsExcluded(k) {
				continue
			}
			oldVal := req.OldData[k]
			oldStr := serializeValue(oldVal)
			newStr := serializeValue(newVal)
			if oldStr == newStr {
				continue
			}
			changes = append(changes, infraports.AuditFieldChange{
				FieldName: k,
				FieldType: 1,
				OldValue:  oldStr,
				NewValue:  newStr,
			})
		}

	case req.Action == 3: // DELETE — every old field is recorded as a change
		for k, v := range req.OldData {
			if IsExcluded(k) {
				continue
			}
			changes = append(changes, infraports.AuditFieldChange{
				FieldName: k,
				FieldType: 1,
				OldValue:  serializeValue(v),
				NewValue:  "",
			})
		}
	}

	// Also capture fields removed on UPDATE (present in old, absent in new)
	if req.Action == 2 && req.OldData != nil {
		for k, v := range req.OldData {
			if IsExcluded(k) {
				continue
			}
			if _, exists := req.NewData[k]; !exists {
				changes = append(changes, infraports.AuditFieldChange{
					FieldName: k,
					FieldType: 1,
					OldValue:  serializeValue(v),
					NewValue:  "",
				})
			}
		}
	}

	return svc.LogEntry(ctx, &infraports.AuditLogRequest{
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

// serializeValue converts any value to its canonical string representation
// for storage in the audit trail.
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
	case time.Time:
		return val.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}
