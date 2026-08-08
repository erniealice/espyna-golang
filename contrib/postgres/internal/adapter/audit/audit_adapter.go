//go:build postgresql

package audit

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/shared/identity"
	"github.com/lib/pq"
)

// auditAdapter implements infraports.AuditService using direct SQL against
// the audit_trail schema. It intentionally avoids PostgresOperations to
// prevent infinite recursion (LogEntry is called from within Create/Update).
type auditAdapter struct {
	db *sql.DB
}

// New returns an AuditService backed by PostgreSQL.
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
	// PostgreSQLTransaction is defined in the core package; we access it via
	// the exported GetTx() method on the interfaces.Transaction value stored
	// in context.
	tx, ok := operations.GetTransactionFromContext(ctx)
	if ok {
		// Use a local interface assertion — only PostgreSQLTransaction has GetTx.
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
func (a *auditAdapter) LogEntry(ctx context.Context, req *infraports.AuditLogRequest) error {
	if req == nil {
		return fmt.Errorf("audit: log entry request is required")
	}
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

	// workspace_id comes from the request (set by DiffAndLog or caller).
	workspaceID := req.WorkspaceID

	const entrySQL = `
		INSERT INTO ` + entityid.AuditEntry + ` (
			workspace_id, actor_id, actor_type, actor_ip, actor_user_agent,
			entity_type, entity_id,
			domain, action, permission_code, use_case, reason, method_name,
			request_id, field_count, transaction_id
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, txid_current()
		) RETURNING id, occurred_at`

	fieldCount := int32(len(req.FieldChanges))

	var entryID string
	var occurredAt time.Time
	err := exec.QueryRowContext(ctx, entrySQL,
		nullableString(workspaceID), ac.ActorID, actorType, nullableString(ac.IP), nullableString(ac.UserAgent),
		req.EntityType, req.EntityID,
		req.Domain, req.Action, req.PermissionCode, req.UseCase, req.Reason, req.MethodName,
		ac.RequestID, fieldCount,
	).Scan(&entryID, &occurredAt)
	if err != nil {
		return fmt.Errorf("audit: insert audit_entry: %w", err)
	}

	// Batched insert (A7 N+1 fix): LogEntry runs on every audited write, and the
	// prior loop issued one INSERT round-trip per changed field. unnest() expands
	// the parallel arrays row-wise so all field changes land in a single statement.
	if len(req.FieldChanges) > 0 {
		names := make([]string, len(req.FieldChanges))
		types := make([]int32, len(req.FieldChanges))
		olds := make([]string, len(req.FieldChanges))
		news := make([]string, len(req.FieldChanges))
		for i, fc := range req.FieldChanges {
			names[i] = fc.FieldName
			types[i] = fc.FieldType
			olds[i] = fc.OldValue
			news[i] = fc.NewValue
		}

		const changeSQL = `
			INSERT INTO ` + entityid.AuditFieldChange + ` (
				audit_entry_id, field_name, field_type, old_value, new_value
			)
			SELECT $1::uuid, u.field_name, u.field_type, u.old_value, u.new_value
			FROM unnest($2::text[], $3::smallint[], $4::text[], $5::text[])
			     AS u(field_name, field_type, old_value, new_value)`

		if _, err := exec.ExecContext(ctx, changeSQL,
			entryID, pq.Array(names), pq.Array(types), pq.Array(olds), pq.Array(news),
		); err != nil {
			return fmt.Errorf("audit: insert audit_field_change batch (%d fields): %w", len(req.FieldChanges), err)
		}
	}

	return nil
}

// auditCursor is the JSON payload encoded in the cursor token.
type auditCursor struct {
	T  string `json:"t"`  // occurred_at in RFC3339
	ID string `json:"id"` // entry UUID
}

const maxAuditCursorTokenBytes = 1024

// ListByEntity returns audit entries for one entity, newest first,
// using keyset (cursor) pagination on (occurred_at DESC, id DESC).
func (a *auditAdapter) ListByEntity(ctx context.Context, req *infraports.ListAuditRequest) (*infraports.ListAuditResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("audit: list by entity request is required")
	}
	requestIdentity, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return nil, fmt.Errorf("audit: list by entity: %w", err)
	}
	if req.WorkspaceID != "" && req.WorkspaceID != requestIdentity.WorkspaceID {
		return nil, fmt.Errorf("audit: requested workspace does not match selected workspace")
	}
	workspaceID := requestIdentity.WorkspaceID
	limit := boundedEntityListLimit(req.Limit)

	// Decode optional cursor.
	var cursorTime time.Time
	var cursorID string
	if req.CursorToken != "" {
		if err := validateAuditCursorTokenLength(req.CursorToken); err != nil {
			return nil, err
		}
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

	var rows *sql.Rows

	// LIMIT+1 pattern to detect whether a next page exists.
	fetchLimit := limit + 1

	if cursorID == "" {
		const q = `
			SELECT id, actor_id, actor_type, entity_type, entity_id,
			       domain, action, permission_code, use_case, reason, method_name,
			       request_id, field_count, occurred_at
			FROM ` + entityid.AuditEntry + `
			WHERE entity_type = $1
			  AND entity_id   = $2
			  AND workspace_id = $3
			ORDER BY occurred_at DESC, id DESC
			LIMIT $4`
		rows, err = exec.QueryContext(ctx, q,
			req.EntityType, req.EntityID, workspaceID, fetchLimit)
	} else {
		const q = `
			SELECT id, actor_id, actor_type, entity_type, entity_id,
			       domain, action, permission_code, use_case, reason, method_name,
			       request_id, field_count, occurred_at
			FROM ` + entityid.AuditEntry + `
			WHERE entity_type = $1
			  AND entity_id   = $2
			  AND workspace_id = $3
			  AND (occurred_at, id) < ($4, $5)
			ORDER BY occurred_at DESC, id DESC
			LIMIT $6`
		rows, err = exec.QueryContext(ctx, q,
			req.EntityType, req.EntityID, workspaceID,
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
		e.WorkspaceID = workspaceID
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

	// Load field changes for ALL returned entries in a SINGLE batched query
	// (A7 N+1 fix). The previous implementation issued one
	// "WHERE audit_entry_id = $1" query PER entry — 21 round-trips for a 20-row
	// page. We now fetch every entry's field changes with one
	// "WHERE audit_entry_id = ANY($1)" round-trip and group the rows back onto
	// their parent entry in Go. ORDER BY (audit_entry_id, id) preserves the
	// original per-entry id ordering within each group.
	if len(entries) > 0 {
		entryIDs := make([]string, len(entries))
		entryByID := make(map[string]*infraports.AuditEntryResult, len(entries))
		for i := range entries {
			entryIDs[i] = entries[i].ID
			entryByID[entries[i].ID] = &entries[i]
		}

		const changesSQL = `
			SELECT audit_entry_id, field_name, field_type, old_value, new_value
			FROM ` + entityid.AuditFieldChange + `
			WHERE audit_entry_id = ANY($1)
			ORDER BY audit_entry_id, id`

		crows, err := exec.QueryContext(ctx, changesSQL, pq.Array(entryIDs))
		if err != nil {
			return nil, fmt.Errorf("audit: query field_changes: %w", err)
		}
		for crows.Next() {
			var entryID string
			var fc infraports.AuditFieldChange
			if err := crows.Scan(&entryID, &fc.FieldName, &fc.FieldType, &fc.OldValue, &fc.NewValue); err != nil {
				crows.Close()
				return nil, fmt.Errorf("audit: scan field_change: %w", err)
			}
			if e, ok := entryByID[entryID]; ok {
				e.FieldChanges = append(e.FieldChanges, fc)
			}
		}
		crows.Close()
		if err := crows.Err(); err != nil {
			return nil, fmt.Errorf("audit: iterate field_change rows: %w", err)
		}
	}

	var nextCursor string
	if hasNext && len(entries) > 0 {
		last := entries[len(entries)-1]
		t, _ := time.Parse(time.RFC3339, last.OccurredAt)
		payload, _ := json.Marshal(auditCursor{T: t.UTC().Format(time.RFC3339Nano), ID: last.ID})
		nextCursor = base64.StdEncoding.EncodeToString(payload)
	}

	return &infraports.ListAuditResponse{
		Entries:    entries,
		HasNext:    hasNext,
		NextCursor: nextCursor,
	}, nil
}

func validateAuditCursorTokenLength(token string) error {
	if len(token) > maxAuditCursorTokenBytes {
		return fmt.Errorf("audit: cursor token exceeds %d bytes", maxAuditCursorTokenBytes)
	}
	return nil
}

// ListByActor returns audit entries for a specific actor, newest first,
// optionally filtered by a use_case prefix (e.g. "switch_").
// COALESCE guards against missing request_url / referer columns in older schemas.
func (a *auditAdapter) ListByActor(ctx context.Context, req *infraports.ListByActorRequest) (*infraports.ListAuditResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("audit: list by actor request is required")
	}
	requestIdentity, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return nil, fmt.Errorf("audit: list by actor: %w", err)
	}
	if req.Limit > 200 {
		return nil, fmt.Errorf("audit: list by actor: query limit %d exceeds maximum 200", req.Limit)
	}
	limit, err := postgresCore.BoundedQueryLimit(int32(req.Limit), 50, 200)
	if err != nil {
		return nil, fmt.Errorf("audit: list by actor: %w", err)
	}
	prefix, err := postgresCore.BoundedPrefixPattern(req.UseCasePrefix)
	if err != nil {
		return nil, fmt.Errorf("audit: list by actor: %w", err)
	}

	exec := a.getExecutor(ctx)

	var rows *sql.Rows

	if prefix != "" {
		const q = `
			SELECT id, actor_id, actor_type, use_case,
			       COALESCE(request_url, ''), COALESCE(referer, ''),
			       occurred_at
			FROM ` + entityid.AuditEntry + `
			WHERE actor_id = $1
			  AND workspace_id = $2
			  AND use_case LIKE $3 ESCAPE '\'
			ORDER BY occurred_at DESC
			LIMIT $4`
		rows, err = exec.QueryContext(ctx, q,
			req.ActorID, requestIdentity.WorkspaceID, prefix, limit)
	} else {
		const q = `
			SELECT id, actor_id, actor_type, use_case,
			       COALESCE(request_url, ''), COALESCE(referer, ''),
			       occurred_at
			FROM ` + entityid.AuditEntry + `
			WHERE actor_id = $1
			  AND workspace_id = $2
			ORDER BY occurred_at DESC
			LIMIT $3`
		rows, err = exec.QueryContext(ctx, q,
			req.ActorID, requestIdentity.WorkspaceID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("audit: query audit_entry by actor: %w", err)
	}
	defer rows.Close()

	var entries []infraports.AuditEntryResult
	for rows.Next() {
		var e infraports.AuditEntryResult
		var occurredAt time.Time
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorType, &e.UseCase,
			&e.RequestURL, &e.Referer,
			&occurredAt,
		); err != nil {
			return nil, fmt.Errorf("audit: scan audit_entry by actor: %w", err)
		}
		e.WorkspaceID = requestIdentity.WorkspaceID
		e.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: iterate audit_entry by actor rows: %w", err)
	}

	return &infraports.ListAuditResponse{
		Entries: entries,
	}, nil
}

// boundedEntityListLimit owns the keyset page budget. Capping rather than
// rejecting preserves existing callers while ensuring LIMIT+1 cannot overflow.
func boundedEntityListLimit(requested int) int {
	const (
		defaultLimit = 20
		maximumLimit = 100
	)
	if requested <= 0 {
		return defaultLimit
	}
	if requested > maximumLimit {
		return maximumLimit
	}
	return requested
}

// nullableString returns nil for empty strings, otherwise the string value.
// Used for optional INET/TEXT columns that accept NULL.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
