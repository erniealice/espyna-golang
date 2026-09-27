//go:build mysql

package core

import (
	"context"
	"database/sql"
	"sync"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/model"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// WorkspaceAwareOperations decorates a DatabaseOperation with automatic
// workspace_id isolation derived from the request context.
//
// It mirrors contrib/postgres/internal/adapter/core.WorkspaceAwareOperations
// with MySQL-specific schema introspection (information_schema scoped to
// DATABASE() rather than the current search_path).
type WorkspaceAwareOperations struct {
	inner         interfaces.DatabaseOperation
	db            *sql.DB
	columnCache   map[string]map[string]bool // table → column → exists
	columnCacheMu sync.RWMutex
}

// Compile-time assertion.
var _ interfaces.DatabaseOperation = (*WorkspaceAwareOperations)(nil)

// NewWorkspaceAwareOperations returns a workspace-scoped DatabaseOperation
// backed by a fresh MySQLOperations instance.
func NewWorkspaceAwareOperations(db *sql.DB) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       NewMySQLOperations(db),
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// NewWorkspaceAwareOperationsFromInner wraps an existing DatabaseOperation
// with workspace-aware filtering.
func NewWorkspaceAwareOperationsFromInner(db *sql.DB, inner interfaces.DatabaseOperation) interfaces.DatabaseOperation {
	return &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
}

// ── DatabaseOperation methods ────────────────────────────────────────────────

func (w *WorkspaceAwareOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if wsID != "" && hasWorkspaceColumn {
		if scoped, ok := w.inner.(interface {
			ListWithScope(context.Context, string, []*commonpb.TypedFilter, *interfaces.ListParams) (*interfaces.ListResult, error)
		}); ok {
			return scoped.ListWithScope(ctx, tableName, []*commonpb.TypedFilter{workspaceIDFilter(wsID)}, params)
		}
		return nil, model.NewDatabaseError("scoped list operation unavailable", "TENANT_SCOPE_UNAVAILABLE", 500)
	}
	return w.inner.List(ctx, tableName, params)
}

func (w *WorkspaceAwareOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if wsID != "" && hasWorkspaceColumn {
		cloned := make(map[string]any, len(data)+1)
		for k, v := range data {
			if k == "workspace_id" || k == "workspaceId" {
				continue
			}
			cloned[k] = v
		}
		cloned["workspace_id"] = wsID
		data = cloned
	}
	return w.inner.Create(ctx, tableName, data)
}

func (w *WorkspaceAwareOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return nil, err
	}
	result, err := w.inner.Read(ctx, tableName, id)
	if err != nil {
		return nil, err
	}

	if wsID == "" || !hasWorkspaceColumn {
		return result, nil
	}

	recordWsID, hasCol := result["workspace_id"]
	if s, ok := recordWsID.(string); !hasCol || !ok || s != wsID {
		return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
	}
	return result, nil
}

func (w *WorkspaceAwareOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if wsID != "" && hasWorkspaceColumn {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return nil, err
		}
		cloned := make(map[string]any, len(data))
		for k, v := range data {
			if k != "workspace_id" && k != "workspaceId" {
				cloned[k] = v
			}
		}
		data = cloned
	}
	return w.inner.Update(ctx, tableName, id, data)
}

func (w *WorkspaceAwareOperations) Delete(ctx context.Context, tableName string, id string) error {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return err
	}
	if wsID != "" && hasWorkspaceColumn {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.Delete(ctx, tableName, id)
}

func (w *WorkspaceAwareOperations) HardDelete(ctx context.Context, tableName string, id string) error {
	wsID, hasWorkspaceColumn, err := w.resolveWorkspaceScope(ctx, tableName)
	if err != nil {
		return err
	}
	if wsID != "" && hasWorkspaceColumn {
		if _, err := w.Read(ctx, tableName, id); err != nil {
			return err
		}
	}
	return w.inner.HardDelete(ctx, tableName, id)
}

func (w *WorkspaceAwareOperations) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	return w.inner.Query(ctx, tableName, query)
}

func (w *WorkspaceAwareOperations) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	return w.inner.QueryOne(ctx, tableName, query)
}

// ── Optional interface methods ───────────────────────────────────────────────

// GetDB returns the underlying *sql.DB.
func (w *WorkspaceAwareOperations) GetDB() *sql.DB { return w.db }

// GetExecutor returns the transaction-aware executor from the inner operation.
// Entity adapters that type-assert to executorProvider use this to participate
// in active transactions.
func (w *WorkspaceAwareOperations) GetExecutor(ctx context.Context) sqlexec.DBExecutor {
	type executorProvider interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}
	if ep, ok := w.inner.(executorProvider); ok {
		return ep.GetExecutor(ctx)
	}
	return w.db
}

// ── Helper methods ───────────────────────────────────────────────────────────

func (w *WorkspaceAwareOperations) getWorkspaceID(ctx context.Context) string {
	if requestIdentity, ok := identity.FromContext(ctx); ok {
		return requestIdentity.WorkspaceID
	}
	return ""
}

func (w *WorkspaceAwareOperations) resolveWorkspaceScope(ctx context.Context, tableName string) (string, bool, error) {
	wsID := w.getWorkspaceID(ctx)
	if wsID == "" {
		return "", false, nil
	}
	hasWorkspaceColumn, err := w.tableHasWorkspaceColumn(ctx, tableName)
	if err != nil {
		return "", false, model.NewDatabaseError("tenant scope is temporarily unavailable", "TENANT_SCOPE_UNAVAILABLE", 503)
	}
	return wsID, hasWorkspaceColumn, nil
}

// tableHasWorkspaceColumn reports whether tableName has a workspace_id column.
// Results are cached; the first miss queries information_schema scoped to
// DATABASE() (MySQL's current schema).
func (w *WorkspaceAwareOperations) tableHasWorkspaceColumn(ctx context.Context, tableName string) (bool, error) {
	w.columnCacheMu.RLock()
	cols, cached := w.columnCache[tableName]
	w.columnCacheMu.RUnlock()
	if cached {
		return cols["workspace_id"], nil
	}

	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position
	`
	rows, err := w.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	colMap := make(map[string]bool)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return false, err
		}
		colMap[col] = true
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	w.columnCacheMu.Lock()
	w.columnCache[tableName] = colMap
	w.columnCacheMu.Unlock()

	return colMap["workspace_id"], nil
}

// injectWorkspaceFilter returns a copy of params with a workspace_id
// StringFilter prepended. The original params value is never mutated.
func (w *WorkspaceAwareOperations) injectWorkspaceFilter(params *interfaces.ListParams, wsID string) *interfaces.ListParams {
	wsFilter := workspaceIDFilter(wsID)

	if params == nil {
		return &interfaces.ListParams{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{wsFilter}}}
	}
	cloned := *params
	newFilters := make([]*commonpb.TypedFilter, 0, len(cloned.Filters.GetFilters())+1)
	newFilters = append(newFilters, wsFilter)
	newFilters = append(newFilters, cloned.Filters.GetFilters()...)
	cloned.Filters = &commonpb.FilterRequest{Filters: newFilters, Logic: cloned.Filters.GetLogic()}
	return &cloned
}

func workspaceIDFilter(wsID string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: "workspace_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:         wsID,
				Operator:      commonpb.StringOperator_STRING_EQUALS,
				CaseSensitive: true,
			},
		},
	}

}
