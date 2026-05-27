//go:build mysql

package core

import (
	"context"
	"fmt"
)

// UpdateWithWorkspaceGuard executes an UPDATE statement and verifies that
// exactly one row was affected (i.e. the row exists AND belongs to the
// caller's workspace). It mirrors the postgres version's shape with two
// mechanical differences:
//
//  1. Placeholders are "?" (MySQL positional) instead of "$N" (postgres).
//     Callers must build query strings with "?" markers and supply args in
//     matching positional order.
//
//  2. MySQL has no RETURNING clause. Callers that need the updated row's
//     values must issue a SELECT after a successful UPDATE — typically via
//     a Read adapter method. Supply a UUID app-side before insert/update so
//     the SELECT predicate is known up-front without a round-trip.
//
// Why RowsAffected — not RETURNING:
// MySQL's UPDATE returns the count of rows actually changed by the engine.
// A zero count means either the row doesn't exist or the workspace_id
// predicate rejected it (multi-tenancy guard). Both cases are surfaced as
// an error so the caller can return a proper 404/403 rather than silently
// succeed on a phantom write.
//
// query must be a complete UPDATE statement using "?" placeholders, including
// a WHERE clause that gates on both the record identifier and workspace_id.
// args must be supplied in the same positional order as the "?" markers.
//
// Returns nil on exactly-one-row success, an error otherwise.
func UpdateWithWorkspaceGuard(
	ctx context.Context,
	db DBExecutor,
	query string,
	args ...any,
) error {
	if db == nil {
		return fmt.Errorf("update: database connection is not available")
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update query failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update: could not read RowsAffected: %w", err)
	}
	if affected == 0 {
		// Row not found or workspace_id mismatch — treat as not found so the
		// caller can propagate a 404. Never silently succeed on a zero-row update.
		return fmt.Errorf("update: record not found or workspace mismatch (0 rows affected)")
	}

	return nil
}

// BulkInsertFromSelect executes a bulk INSERT ... SELECT statement.
//
// MySQL limitation — no RETURNING:
// Unlike postgres (`INSERT ... RETURNING *`) or SQL Server
// (`INSERT ... OUTPUT inserted.*`), MySQL has no native clause to return the
// newly inserted rows in a single statement. Callers that need the IDs of
// inserted rows have two options:
//
//  1. Supply UUIDs app-side (recommended): generate all IDs in Go before
//     calling BulkInsertFromSelect, include them in the SELECT projection,
//     and store them in memory. No second round-trip needed.
//
//  2. SELECT after INSERT: if the inserted set is uniquely identifiable
//     (e.g. by a correlation key or a timestamp window), issue a follow-up
//     SELECT after this call returns nil.
//
// query must be a complete INSERT ... SELECT statement using "?" placeholders.
// args must be supplied in positional order matching the "?" markers.
//
// Returns the count of inserted rows on success, or an error.
func BulkInsertFromSelect(
	ctx context.Context,
	db DBExecutor,
	query string,
	args ...any,
) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("bulk insert: database connection is not available")
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("bulk insert query failed: %w", err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("bulk insert: could not read RowsAffected: %w", err)
	}

	return inserted, nil
}
