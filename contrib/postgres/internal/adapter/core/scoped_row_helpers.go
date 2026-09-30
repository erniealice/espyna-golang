//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Shared helpers for direct-workspace-column adapters that need typed proto <-> column-map
// conversion plus author-owned raw SQL row locks (20260927-usage-and-pass-through-charges,
// Slice B). Workspace scoping of ordinary CRUD is enforced by the workspace-aware operations
// (entities registered workspaceScopeDirectRequired); the helpers below add only the lock paths,
// each with a strict, non-empty workspace predicate. The `table`/`col` arguments are package
// constants (entityid / column literals) supplied by adapter code, never caller input.

// ScopedProtoToMap marshals a proto message to a snake_case column map and strips any
// client-supplied workspace key: the workspace-aware decorator injects the trusted value on
// create and never lets update reassign it.
func ScopedProtoToMap(msg proto.Message) (map[string]any, error) {
	raw, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var in map[string]any
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	out := make(map[string]any, len(in))
	claimed := make(map[string]string, len(in))
	for k, v := range in {
		col := camelToSnake(k)
		if prev, dup := claimed[col]; dup {
			return nil, fmt.Errorf("proto->column key collision: %q and %q both canonicalize to %q", prev, k, col)
		}
		claimed[col] = k
		if col == "workspace_id" {
			continue
		}
		out[col] = v
	}
	return out, nil
}

// ScopedResultToProto converts a database row map into a proto message.
func ScopedResultToProto(result any, msg proto.Message) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return nil
}

// ScopedListParams forwards search/filters/sort/pagination and refuses OR filter logic: the
// workspace decorator APPENDs its predicate, which under OR grouping would be OR-ed away.
func ScopedListParams(search *commonpb.SearchRequest, filters *commonpb.FilterRequest, sort *commonpb.SortRequest, pagination *commonpb.PaginationRequest) (*interfaces.ListParams, error) {
	if filters != nil && filters.GetLogic() == commonpb.FilterLogic_OR && len(filters.GetFilters()) > 0 {
		return nil, fmt.Errorf("INVALID_LIST_REQUEST: OR filter logic is not supported for workspace-scoped lists (tenant scope must be AND-ed)")
	}
	if search == nil && filters == nil && sort == nil && pagination == nil {
		return nil, nil
	}
	return &interfaces.ListParams{Search: search, Filters: filters, Sort: sort, Pagination: pagination}, nil
}

// RequireTrustedWorkspace returns the trusted workspace of the request or fails closed.
func RequireTrustedWorkspace(ctx context.Context) (string, error) {
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return "", fmt.Errorf("no trusted workspace in context (fail closed)")
	}
	return idn.WorkspaceID, nil
}

// TxExecutor returns the ambient-transaction-aware executor of the operations, or nil.
func TxExecutor(ctx context.Context, dbOps interfaces.DatabaseOperation) sqlexec.DBExecutor {
	if ep, ok := dbOps.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}); ok {
		return ep.GetExecutor(ctx)
	}
	return nil
}

// LockScopedRowForUpdate takes SELECT ... FOR UPDATE on one row of a direct-workspace table inside
// the ambient transaction. Requires a trusted workspace and a transaction (never r.db); a row of
// another workspace is reported as not found (wrapping domainports.ErrLockedRowNotFound;
// indistinguishable from a missing row).
func LockScopedRowForUpdate(ctx context.Context, dbOps interfaces.DatabaseOperation, table, id string) error {
	if id == "" {
		return fmt.Errorf("%s lock: ID is required", table)
	}
	wsID, err := RequireTrustedWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("%s lock: %w", table, err)
	}
	exec := TxExecutor(ctx, dbOps)
	if exec == nil {
		return fmt.Errorf("%s lock: no SQL executor available", table)
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return fmt.Errorf("%s lock: requires an ambient transaction (fail closed)", table)
	}
	// The workspace predicate is part of the lock statement (C5): a foreign-workspace row is never
	// locked, and is indistinguishable from a missing one.
	var lockedID string
	if err := exec.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE id = $1 AND workspace_id = $2 FOR UPDATE`, id, wsID).Scan(&lockedID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%s lock: %w", table, domainports.ErrLockedRowNotFound)
		}
		return fmt.Errorf("%s lock: %w", table, err)
	}
	return nil
}

// LockScopedRowIDsForUpdate locks every row of `table` whose `whereCol` equals `whereVal` in the
// trusted workspace, ordered by orderBy (deterministic lock order avoids deadlocks), inside the
// ambient transaction, and returns the ids in that order. Zero rows is not an error.
func LockScopedRowIDsForUpdate(ctx context.Context, dbOps interfaces.DatabaseOperation, table, whereCol, whereVal, orderBy string) ([]string, error) {
	if whereVal == "" {
		return nil, fmt.Errorf("%s lock by %s: value is required", table, whereCol)
	}
	wsID, err := RequireTrustedWorkspace(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s lock by %s: %w", table, whereCol, err)
	}
	exec := TxExecutor(ctx, dbOps)
	if exec == nil {
		return nil, fmt.Errorf("%s lock by %s: no SQL executor available", table, whereCol)
	}
	if _, isTx := exec.(*sql.Tx); !isTx {
		return nil, fmt.Errorf("%s lock by %s: requires an ambient transaction (fail closed)", table, whereCol)
	}
	rows, err := exec.QueryContext(ctx, `SELECT id FROM `+table+` WHERE `+whereCol+` = $1 AND workspace_id = $2 ORDER BY `+orderBy+` FOR UPDATE`, whereVal, wsID)
	if err != nil {
		return nil, fmt.Errorf("%s lock by %s: %w", table, whereCol, err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%s lock by %s: %w", table, whereCol, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
