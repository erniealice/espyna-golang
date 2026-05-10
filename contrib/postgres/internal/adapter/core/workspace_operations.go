//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"sync"

	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/model"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// WorkspaceAwareOperations is a decorator over DatabaseOperation that
// transparently injects workspace_id filtering derived from the request context.
//
// Every operation checks two preconditions before applying workspace isolation:
//  1. The request context carries a non-empty workspace_id.
//  2. The target table has a workspace_id column (checked via information_schema,
//     result cached per table name).
//
// When either precondition is false the call is forwarded to the inner
// DatabaseOperation unchanged — this makes the decorator safe for
// service-to-service calls and global (non-tenanted) tables.
//
// Usage:
//
//	dbOps := core.NewWorkspaceAwareOperations(db)
type WorkspaceAwareOperations struct {
	inner         interfaces.DatabaseOperation
	db            *sql.DB
	columnCache   map[string]map[string]bool // table name → column name → exists
	columnCacheMu sync.RWMutex
}

// Ensure WorkspaceAwareOperations satisfies the full DatabaseOperation interface
// at compile time.
var _ interfaces.DatabaseOperation = (*WorkspaceAwareOperations)(nil)

// NewWorkspaceAwareOperations returns a DatabaseOperation that wraps a new
// PostgresOperations instance with automatic workspace_id isolation.
func NewWorkspaceAwareOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       NewPostgresOperations(db),
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// NewWorkspaceAwareOperationsFromInner wraps an existing DatabaseOperation
// with workspace-aware filtering. Use this when you already have an instance
// (e.g. one created with NewPostgresOperationsWithAudit).
func NewWorkspaceAwareOperationsFromInner(db *sql.DB, inner interfaces.DatabaseOperation) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// ── DatabaseOperation methods ────────────────────────────────────────────────

// List delegates to the inner List, optionally prepending a workspace_id
// StringFilter when the context carries a workspace and the table has the
// column.
func (w *WorkspaceAwareOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		params = w.injectWorkspaceFilter(params, wsID)
	}
	return w.inner.List(ctx, tableName, params)
}

// Create injects workspace_id into the data map before inserting, when the
// context carries a workspace and the table has the column.
func (w *WorkspaceAwareOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		// Clone the map to avoid mutating the caller's data.
		cloned := make(map[string]any, len(data)+1)
		for k, v := range data {
			cloned[k] = v
		}
		cloned["workspace_id"] = wsID
		data = cloned
	}
	return w.inner.Create(ctx, tableName, data)
}

// Read delegates to the inner Read, then verifies that the returned record
// belongs to the context workspace. Returns a 404 if the workspace_id does
// not match, preventing cross-workspace data leakage via direct-ID lookup.
//
// NULL-row rejection (Phase 1.5 — 2026-05-10 — codex C1):
// When the context carries a workspace and the table has a workspace_id column,
// a row whose workspace_id is NULL or empty is also rejected as 404. Without
// this check, legacy NULL rows left by the Phase 1 migration are visible across
// all workspaces via direct-ID reads, since any non-NULL != wsID comparison
// returns false (the previous condition only fired on non-nil mismatches).
// Update and Delete already call Read for ownership verification, so they
// inherit this fix automatically.
func (w *WorkspaceAwareOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	result, err := w.inner.Read(ctx, tableName, id)
	if err != nil {
		return nil, err
	}

	wsID := w.getWorkspaceID(ctx)
	if wsID == "" {
		return result, nil
	}

	// Only apply workspace enforcement when the table has the column.
	// Tables without workspace_id are not tenant-scoped and pass through.
	if !w.tableHasWorkspaceColumn(ctx, tableName) {
		return result, nil
	}

	recordWsID, hasCol := result["workspace_id"]
	if !hasCol {
		// Column not present in result set — defensive pass-through; should not
		// happen for tables that tableHasWorkspaceColumn confirmed.
		return result, nil
	}

	// Reject both NULL workspace_id rows AND rows that belong to a different workspace.
	// A NULL workspace_id means the row pre-dates the tenancy migration and has not
	// been backfilled; treating it as "accessible by anyone" would be a tenancy leak.
	if recordWsID == nil {
		return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}
	if recordWsIDStr, ok := recordWsID.(string); ok {
		if recordWsIDStr == "" || recordWsIDStr != wsID {
			return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
		}
	}

	return result, nil
}

// Update verifies workspace ownership via a Read before delegating to the
// inner Update. It also strips any workspace_id key from the data payload to
// prevent cross-workspace reassignment.
func (w *WorkspaceAwareOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		// Verify ownership — reuse Read which already enforces workspace check.
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return nil, err
		}
		// Strip workspace_id from the update payload; it must never change.
		cloned := make(map[string]any, len(data))
		for k, v := range data {
			if k != "workspace_id" {
				cloned[k] = v
			}
		}
		data = cloned
	}
	return w.inner.Update(ctx, tableName, id, data)
}

// Delete verifies workspace ownership via a Read, then delegates the soft
// delete to the inner operation.
func (w *WorkspaceAwareOperations) Delete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.Delete(ctx, tableName, id)
}

// HardDelete verifies workspace ownership via a Read, then delegates the
// permanent delete to the inner operation.
func (w *WorkspaceAwareOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	wsID := w.getWorkspaceID(ctx)
	if wsID != "" && w.tableHasWorkspaceColumn(ctx, tableName) {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.HardDelete(ctx, tableName, id)
}

// Query passes through to the inner operation. Injecting workspace filters
// into QueryBuilder is non-trivial; callers that use Query are expected to
// include workspace filtering themselves.
func (w *WorkspaceAwareOperations) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	return w.inner.Query(ctx, tableName, query)
}

// QueryOne passes through to the inner operation (see Query).
func (w *WorkspaceAwareOperations) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	return w.inner.QueryOne(ctx, tableName, query)
}

// ── Optional interface methods (type-asserted by adapters) ───────────────────

// GetDB returns the underlying *sql.DB so that adapters performing raw SQL
// (CTEs, JOINs) can obtain a connection via the standard type assertion:
//
//	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok { ... }
func (w *WorkspaceAwareOperations) GetDB() *sql.DB {
	return w.db
}

// GetExecutor returns the transaction-aware executor from the inner operation.
// Adapters that type-assert to an executorProvider interface use this to
// participate in active transactions.
func (w *WorkspaceAwareOperations) GetExecutor(ctx context.Context) interfaces.DBExecutor {
	type executorProvider interface {
		GetExecutor(ctx context.Context) interfaces.DBExecutor
	}
	if ep, ok := w.inner.(executorProvider); ok {
		return ep.GetExecutor(ctx)
	}
	return w.db
}

// ── Helper methods ───────────────────────────────────────────────────────────

// getWorkspaceID extracts the workspace_id from the request context.
// Returns an empty string if no workspace is present (e.g. service-to-service
// calls or unauthenticated contexts), which disables all workspace injection.
func (w *WorkspaceAwareOperations) getWorkspaceID(ctx context.Context) string {
	return consumer.GetWorkspaceIDFromContext(ctx)
}

// tableHasWorkspaceColumn reports whether tableName has a workspace_id column.
// Results are cached with a read-preferred RWMutex; the first miss for a table
// performs a live query against information_schema.columns.
func (w *WorkspaceAwareOperations) tableHasWorkspaceColumn(ctx context.Context, tableName string) bool {
	w.columnCacheMu.RLock()
	cols, cached := w.columnCache[tableName]
	w.columnCacheMu.RUnlock()

	if cached {
		return cols["workspace_id"]
	}

	// Cache miss — query the schema. Use the underlying db directly to avoid
	// a potential recursive call through the decorated operation.
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`
	rows, err := w.db.QueryContext(ctx, query, tableName)
	if err != nil {
		// On error, conservatively skip injection rather than blocking the call.
		return false
	}
	defer rows.Close()

	colMap := make(map[string]bool)
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			continue
		}
		colMap[colName] = true
	}
	if rows.Err() != nil {
		return false
	}

	w.columnCacheMu.Lock()
	w.columnCache[tableName] = colMap
	w.columnCacheMu.Unlock()

	return colMap["workspace_id"]
}

// injectWorkspaceFilter returns a copy of params with a workspace_id
// StringFilter prepended. The original params value is never mutated.
// If params is nil a new ListParams is allocated.
func (w *WorkspaceAwareOperations) injectWorkspaceFilter(params *interfaces.ListParams, wsID string) *interfaces.ListParams {
	wsFilter := &commonpb.TypedFilter{
		Field: "workspace_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         wsID,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}

	if params == nil {
		return &interfaces.ListParams{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{wsFilter},
			},
		}
	}

	// Clone ListParams shallowly, deep-copy only the Filters slice.
	cloned := *params
	if cloned.Filters == nil {
		cloned.Filters = &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{wsFilter},
		}
	} else {
		newFilters := make([]*commonpb.TypedFilter, 0, len(cloned.Filters.Filters)+1)
		newFilters = append(newFilters, wsFilter)
		newFilters = append(newFilters, cloned.Filters.Filters...)
		cloned.Filters = &commonpb.FilterRequest{
			Filters: newFilters,
			Logic:   cloned.Filters.Logic,
		}
	}

	return &cloned
}