//go:build postgresql

package core

import (
	"context"
	"fmt"

	sqlexec "github.com/erniealice/espyna-golang/database/sqlexec"
)

// write_helpers.go hoists the two direct-SQL write shapes that domain adapters
// previously open-coded (Q-WRITE-PREPARE, Option A + A6 + A7):
//
//   - UpdateWithWorkspaceGuard — a single-row UPDATE scoped by BOTH id and
//     workspace_id, returning RowsAffected so callers can treat 0 as not-found
//     (closes the A6 "ignored RowsAffected + missing tenant predicate" class).
//   - BulkInsertFromSelect — a set-based INSERT ... SELECT helper that replaces
//     per-row INSERT loops (closes the A7 N+1-INSERT class).
//
// Both take an sqlexec.DBExecutor so they work uniformly over *sql.DB and
// *sql.Tx (the shared executor interface; see
// internal/infrastructure/adapters/secondary/database/common/interface/operations.go).

// UpdateWithWorkspaceGuard runs an UPDATE scoped by BOTH id AND workspace_id and
// returns the number of rows affected. Callers MUST treat a 0 return as
// not-found (or cross-tenant access) and surface the appropriate error — this is
// the A6 fix that turns previously-silent tenant leaks / no-op updates into
// explicit failures.
//
//	table     — the (already-trusted, not user-supplied) table name.
//	setClause — the column assignments WITHOUT the leading "SET", using
//	            positional placeholders starting at $1 (e.g. "status = $1, date_modified = NOW()").
//	setArgs   — the args bound to the placeholders in setClause, in order.
//	id         — the row primary key.
//	workspaceID— the tenant guard; pass the value from
//	             consumer.GetWorkspaceIDFromContext(ctx).
//
// The id and workspaceID placeholders are appended AFTER setArgs, so the final
// query is: UPDATE <table> SET <setClause> WHERE id = $N AND workspace_id = $N+1.
func UpdateWithWorkspaceGuard(
	ctx context.Context,
	db sqlexec.DBExecutor,
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

	idPos := len(setArgs) + 1
	wsPos := len(setArgs) + 2
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d AND workspace_id = $%d",
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

// HardDeleteByColumn runs a physical DELETE scoped by a single equality
// predicate (e.g. "DELETE FROM <table> WHERE <column> = $1") and returns the
// number of rows deleted. It is the sanctioned funnel for the small set of
// adapters that must HARD-delete a set of rows keyed by a foreign column
// rather than soft-delete a single row by primary key (the generic
// PostgresOperations.Delete path).
//
//	table  — the (already-trusted, not user-supplied) table name.
//	column — the (already-trusted, not user-supplied) predicate column name.
//	value  — the value bound to the single $1 placeholder.
//
// The emitted SQL is exactly:
//
//	DELETE FROM <table> WHERE <column> = $1
//
// which is byte-equivalent to the open-coded statements this helper replaces.
// Callers that previously discarded RowsAffected may ignore the returned count.
func HardDeleteByColumn(
	ctx context.Context,
	db sqlexec.DBExecutor,
	table string,
	column string,
	value any,
) (int64, error) {
	if table == "" {
		return 0, fmt.Errorf("HardDeleteByColumn: table name is required")
	}
	if column == "" {
		return 0, fmt.Errorf("HardDeleteByColumn: column name is required")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, column)
	res, err := db.ExecContext(ctx, query, value)
	if err != nil {
		return 0, fmt.Errorf("HardDeleteByColumn: delete %s: %w", table, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("HardDeleteByColumn: rows affected for %s: %w", table, err)
	}
	return affected, nil
}

// UpdateColumnByID runs a single-row UPDATE scoped by primary key ONLY (no
// workspace predicate) and returns the number of rows affected. It is the
// sanctioned funnel for the small set of adapters that legitimately update a
// row keyed by `id` alone — typically a soft-delete (SET active = false) or a
// single-column write on a child row whose tenant is already established by its
// parent. For UPDATEs that MUST be tenant-scoped, use UpdateWithWorkspaceGuard
// instead; do NOT reach for this helper to drop a workspace predicate.
//
//	table     — the (already-trusted, not user-supplied) table name.
//	setClause — the column assignments WITHOUT the leading "SET", using
//	            positional placeholders starting at $1 (e.g. "active = false" or
//	            "quantity_delivered = $1").
//	setArgs   — the args bound to the placeholders in setClause, in order. May be
//	            empty when the setClause is fully literal (e.g. "active = false").
//	id        — the row primary key, bound to the trailing $N placeholder.
//
// The id placeholder is appended AFTER setArgs, so the emitted SQL is exactly:
//
//	UPDATE <table> SET <setClause> WHERE id = $N
//
// which is byte-equivalent to the open-coded statements this helper replaces.
// Callers that previously discarded RowsAffected may ignore the returned count.
func UpdateColumnByID(
	ctx context.Context,
	db sqlexec.DBExecutor,
	table string,
	setClause string,
	setArgs []any,
	id string,
) (int64, error) {
	if table == "" {
		return 0, fmt.Errorf("UpdateColumnByID: table name is required")
	}
	if setClause == "" {
		return 0, fmt.Errorf("UpdateColumnByID: set clause is required")
	}

	idPos := len(setArgs) + 1
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", table, setClause, idPos)

	args := make([]any, 0, len(setArgs)+1)
	args = append(args, setArgs...)
	args = append(args, id)

	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("UpdateColumnByID: update %s: %w", table, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("UpdateColumnByID: rows affected for %s: %w", table, err)
	}
	return affected, nil
}

// BulkInsertFromSelect runs a single set-based INSERT (typically an
// "INSERT INTO <table> (...) SELECT ... FROM <source> WHERE ..." statement) and
// returns the number of rows inserted. It is the A7 fix for converting per-row
// INSERT loops into one round-trip.
//
// The caller supplies the full insertSQL (including the INSERT INTO ... SELECT
// body and any positional placeholders) plus the bound args. Existing WHERE
// predicates and ORDER BY clauses MUST be preserved verbatim by the caller.
//
// For postgres the caller may use gen_random_uuid()/uuid_generate_v4() inside
// the SELECT for per-row IDs, or pass app-side IDs; append "RETURNING id" and
// switch to QueryContext at the call site when the inserted IDs are needed.
func BulkInsertFromSelect(
	ctx context.Context,
	db sqlexec.DBExecutor,
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
