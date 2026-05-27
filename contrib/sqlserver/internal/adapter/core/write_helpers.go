//go:build sqlserver

package core

import (
	"context"
	"fmt"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// write_helpers.go hoists the two direct-SQL write shapes that domain adapters
// previously open-coded (Q-WRITE-PREPARE, Option A + A6 + A7).
//
// SQL Server differences from the postgres gold standard:
//
//   - Placeholders: @p1, @p2, … (not $1, $2). The idPos and wsPos args are
//     formatted as @pN accordingly.
//   - Identifier quoting: square brackets ([table], [col]) rather than double
//     quotes. Table/column names passed as trusted constants must use this form.
//   - RETURNING: SQL Server uses OUTPUT inserted.* instead of RETURNING. For
//     UpdateWithWorkspaceGuard the brief says to keep it simple with
//     RowsAffected — no OUTPUT clause here. For BulkInsertFromSelect the caller
//     may optionally embed "OUTPUT inserted.id" in their INSERT SQL directly.
//   - No RETURNING on UPDATE: RowsAffected is the tenant-guard signal.

// UpdateWithWorkspaceGuard runs an UPDATE scoped by BOTH id AND workspace_id and
// returns the number of rows affected. Callers MUST treat a 0 return as
// not-found (or cross-tenant access) and surface the appropriate error — this is
// the A6 fix that turns previously-silent tenant leaks / no-op updates into
// explicit failures.
//
//	table      — the (already-trusted, not user-supplied) table name.
//	setClause  — the column assignments WITHOUT the leading "SET", using SQL Server
//	             positional placeholders starting at @p1 (e.g.
//	             "status = @p1, date_modified = GETUTCDATE()").
//	setArgs    — the args bound to the placeholders in setClause, in order.
//	id         — the row primary key.
//	workspaceID— the tenant guard; pass the value from
//	             consumer.GetWorkspaceIDFromContext(ctx).
//
// The id and workspaceID placeholders are appended AFTER setArgs, so the final
// query is: UPDATE <table> SET <setClause> WHERE id = @pN AND workspace_id = @pN+1.
//
// Note: SQL Server also supports OUTPUT inserted.* to return affected rows, but
// this helper keeps it simple with RowsAffected per the brief (A6 fix only).
func UpdateWithWorkspaceGuard(
	ctx context.Context,
	db interfaces.DBExecutor,
	table string,
	setClause string,
	setArgs []any,
	id string,
	workspaceID string,
) (int64, error) {
	if table == "" {
		return 0, fmt.Errorf("UpdateWithWorkspaceGuard: table name is required")
	}
	if setClause == "" {
		return 0, fmt.Errorf("UpdateWithWorkspaceGuard: set clause is required")
	}

	// SQL Server uses @pN placeholders. The caller's setClause already uses @p1…
	// @pLen(setArgs). The id and workspace_id predicates consume the next two slots.
	idPos := len(setArgs) + 1
	wsPos := len(setArgs) + 2
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = @p%d AND workspace_id = @p%d",
		table, setClause, idPos, wsPos,
	)

	args := make([]any, 0, len(setArgs)+2)
	args = append(args, setArgs...)
	args = append(args, id, workspaceID)

	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("UpdateWithWorkspaceGuard: update %s: %w", table, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("UpdateWithWorkspaceGuard: rows affected for %s: %w", table, err)
	}
	return affected, nil
}

// BulkInsertFromSelect runs a single set-based INSERT (typically an
// "INSERT INTO <table> (...) SELECT ... FROM <source> WHERE ..." statement) and
// returns the number of rows inserted. It is the A7 fix for converting per-row
// INSERT loops into one round-trip.
//
// The caller supplies the full insertSQL (including INSERT INTO ... SELECT body
// and any @pN placeholders) plus the bound args. Existing WHERE predicates MUST
// be preserved verbatim by the caller.
//
// SQL Server specifics for callers:
//
//   - Use @p1, @p2, … placeholders (not $N).
//   - Use square-bracket quoting for identifiers: [table], [col].
//   - To capture inserted IDs, embed "OUTPUT inserted.id" directly in the
//     INSERT SQL (between INSERT INTO ... and SELECT), then use QueryContext at
//     the call site. This function uses ExecContext (no returned rows). For the
//     rows-returned variant, call db.QueryContext directly.
//   - UUIDs: use NEWID() inside the SELECT or supply app-side IDs; there is no
//     gen_random_uuid() in SQL Server.
func BulkInsertFromSelect(
	ctx context.Context,
	db interfaces.DBExecutor,
	insertSQL string,
	args []any,
) (int64, error) {
	if insertSQL == "" {
		return 0, fmt.Errorf("BulkInsertFromSelect: insert SQL is required")
	}
	res, err := db.ExecContext(ctx, insertSQL, args...)
	if err != nil {
		return 0, fmt.Errorf("BulkInsertFromSelect: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("BulkInsertFromSelect: rows affected: %w", err)
	}
	return affected, nil
}
