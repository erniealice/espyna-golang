//go:build mysql

// Package audit provides a MySQL-backed AuditService implementation.
//
// Dialect differences from the postgres gold standard:
//   - Placeholders: $N → ? (positional, re-sequenced)
//   - No RETURNING: entryID is supplied app-side (UUID) before INSERT, then read back
//   - txid_current() is not available in MySQL; the column is omitted / defaults to NULL
//   - Keyset pagination uses a correlated WHERE on (occurred_at, id) — MySQL 8.0 supports
//     multi-column tuple comparison, so the predicate shape is identical to postgres
//   - ILIKE → LIKE (MySQL string collations are case-insensitive by default)
package audit

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/shared/database/operations"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/google/uuid"
)

// auditAdapter implements infraports.AuditService using direct SQL against
// the audit_trail schema on MySQL. It intentionally avoids MySQLOperations to
// prevent infinite recursion (LogEntry is called from within Create/Update).
type auditAdapter struct {
	db *sql.DB
}

// New returns an AuditService backed by MySQL.
func New(db *sql.DB) infraports.AuditService {
	return &auditAdapter{db: db}
}

// dbExecutor is the common interface for *sql.DB and *sql.Tx.
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// getExecutor returns the active *sql.Tx if one is present in ctx,
// otherwise falls back to the pool *sql.DB.
func (a *auditAdapter) getExecutor(ctx context.Context) dbExecutor {
	tx, ok := operations.GetTransactionFromContext(ctx)
	if ok {
		type txGetter interface {
			GetTx() *sql.Tx
		}
		if getter, ok := tx.(txGetter); ok {
			if sqlTx := getter.GetTx(); sqlTx != nil {
				return sqlTx
			}
		}
	}
	return a.db
}

// LogEntry writes one audit entry plus its associated field changes.
// It participates in the caller's transaction if one is present in ctx.
//
// MySQL differences:
//   - No RETURNING: entryID is generated app-side (UUID).
//   - txid_current() is postgres-only; the transaction_id column is omitted
//     (it must be nullable or removed from the MySQL schema).
//   - Placeholders are ? (positional).
func (a *auditAdapter) LogEntry(ctx context.Context, req *infraports.AuditLogRequest) error {
	ac, _ := infraports.GetAuditContext(ctx)

	actorType := int32(0)
	switch ac.ActorType {
	case "user":
		actorType = 1
	case "system":
		actorType = 2
	case "api_key":
		actorType = 3
	}

	exec := a.getExecutor(ctx)

	workspaceID := req.WorkspaceID
	entryID := uuid.NewString()
	occurredAt := time.Now().UTC()
	fieldCount := int32(len(req.FieldChanges))

	const entrySQL = `
		INSERT INTO audit_trail_audit_entry (
			id,
			workspace_id, actor_id, actor_type, actor_ip, actor_user_agent,
			entity_type, entity_id,
			domain, action, permission_code, use_case, reason, method_name,
			request_id, field_count, occurred_at
		) VALUES (
			?,
			?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?
		)`

	if _, err := exec.ExecContext(ctx, entrySQL,
		entryID,
		nullableString(workspaceID), ac.ActorID, actorType, nullableString(ac.IP), nullableString(ac.UserAgent),
		req.EntityType, req.EntityID,
		req.Domain, req.Action, req.PermissionCode, req.UseCase, req.Reason, req.MethodName,
		ac.RequestID, fieldCount, occurredAt,
	); err != nil {
		return fmt.Errorf("audit: insert audit_entry: %w", err)
	}

	const changeSQL = `
		INSERT INTO audit_trail_audit_field_change (
			audit_entry_id, field_name, field_type, old_value, new_value
		) VALUES (?, ?, ?, ?, ?)`

	for _, fc := range req.FieldChanges {
		if _, err := exec.ExecContext(ctx, changeSQL,
			entryID, fc.FieldName, fc.FieldType, fc.OldValue, fc.NewValue,
		); err != nil {
			return fmt.Errorf("audit: insert audit_field_change (field=%s): %w", fc.FieldName, err)
		}
	}

	return nil
}

// auditCursor is the JSON payload encoded in the cursor token.
type auditCursor struct {
	T  string `json:"t"`  // occurred_at in RFC3339Nano
	ID string `json:"id"` // entry UUID
}

// ListByEntity returns audit entries for one entity, newest first,
// using keyset (cursor) pagination on (occurred_at DESC, id DESC).
//
// MySQL 8.0+ supports multi-column tuple comparisons:
//
//	WHERE (occurred_at, id) < (?, ?)
//
// which is identical to the postgres predicate.
func (a *auditAdapter) ListByEntity(ctx context.Context, req *infraports.ListAuditRequest) (*infraports.ListAuditResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	// Decode optional cursor.
	var cursorTime time.Time
	var cursorID string
	if req.CursorToken != "" {
		raw, err := base64.StdEncoding.DecodeString(req.CursorToken)
		if err != nil {
			return nil, fmt.Errorf("audit: invalid cursor token: %w", err)
		}
		var c auditCursor
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("audit: invalid cursor payload: %w", err)
		}
		cursorTime, err = time.Parse(time.RFC3339Nano, c.T)
		if err != nil {
			return nil, fmt.Errorf("audit: invalid cursor time: %w", err)
		}
		cursorID = c.ID
	}

	exec := a.getExecutor(ctx)

	// LIMIT+1 pattern to detect whether a next page exists.
	fetchLimit := limit + 1

	var (
		rows *sql.Rows
		err  error
	)

	if cursorID == "" {
		const q = `
			SELECT id, actor_id, actor_type, entity_type, entity_id,
			       domain, action, permission_code, use_case, reason, method_name,
			       request_id, field_count, occurred_at
			FROM audit_trail_audit_entry
			WHERE entity_type = ?
			  AND entity_id   = ?
			  AND workspace_id = ?
			ORDER BY occurred_at DESC, id DESC
			LIMIT ?`
		rows, err = exec.QueryContext(ctx, q,
			req.EntityType, req.EntityID, req.WorkspaceID, fetchLimit)
	} else {
		// MySQL 8.0+ supports tuple comparison for keyset pagination.
		const q = `
			SELECT id, actor_id, actor_type, entity_type, entity_id,
			       domain, action, permission_code, use_case, reason, method_name,
			       request_id, field_count, occurred_at
			FROM audit_trail_audit_entry
			WHERE entity_type = ?
			  AND entity_id   = ?
			  AND workspace_id = ?
			  AND (occurred_at, id) < (?, ?)
			ORDER BY occurred_at DESC, id DESC
			LIMIT ?`
		rows, err = exec.QueryContext(ctx, q,
			req.EntityType, req.EntityID, req.WorkspaceID,
			cursorTime, cursorID, fetchLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("audit: query audit_entry: %w", err)
	}
	defer rows.Close()

	var entries []infraports.AuditEntryResult
	for rows.Next() {
		var e infraports.AuditEntryResult
		var occurredAt time.Time
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorType, &e.EntityType, &e.EntityID,
			&e.Domain, &e.Action, &e.PermissionCode, &e.UseCase, &e.Reason, &e.MethodName,
			&e.RequestID, &e.FieldCount, &occurredAt,
		); err != nil {
			return nil, fmt.Errorf("audit: scan audit_entry: %w", err)
		}
		e.WorkspaceID = req.WorkspaceID
		e.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: iterate audit_entry rows: %w", err)
	}

	hasNext := len(entries) > limit
	if hasNext {
		entries = entries[:limit]
	}

	// Load field changes for each returned entry.
	const changesSQL = `
		SELECT field_name, field_type, old_value, new_value
		FROM audit_trail_audit_field_change
		WHERE audit_entry_id = ?
		ORDER BY id`

	for i := range entries {
		crows, err := exec.QueryContext(ctx, changesSQL, entries[i].ID)
		if err != nil {
			return nil, fmt.Errorf("audit: query field_changes for entry %s: %w", entries[i].ID, err)
		}
		for crows.Next() {
			var fc infraports.AuditFieldChange
			if err := crows.Scan(&fc.FieldName, &fc.FieldType, &fc.OldValue, &fc.NewValue); err != nil {
				crows.Close()
				return nil, fmt.Errorf("audit: scan field_change: %w", err)
			}
			entries[i].FieldChanges = append(entries[i].FieldChanges, fc)
		}
		crows.Close()
		if err := crows.Err(); err != nil {
			return nil, fmt.Errorf("audit: iterate field_change rows: %w", err)
		}
	}

	var nextCursor string
	if hasNext && len(entries) > 0 {
		last := entries[len(entries)-1]
		t, _ := time.Parse(time.RFC3339Nano, last.OccurredAt)
		payload, _ := json.Marshal(auditCursor{T: t.UTC().Format(time.RFC3339Nano), ID: last.ID})
		nextCursor = base64.StdEncoding.EncodeToString(payload)
	}

	return &infraports.ListAuditResponse{
		Entries:    entries,
		HasNext:    hasNext,
		NextCursor: nextCursor,
	}, nil
}

// ListByActor returns audit entries for a specific actor, newest first.
// MySQL stub — returns empty; the feature is postgres-first.
func (a *auditAdapter) ListByActor(_ context.Context, _ *infraports.ListByActorRequest) (*infraports.ListAuditResponse, error) {
	return &infraports.ListAuditResponse{}, nil
}

// nullableString returns nil for empty strings, otherwise the string value.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
