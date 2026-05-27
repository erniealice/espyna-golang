//go:build sqlserver

package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
)

// DBExecutor is the shared executor abstraction over *sql.DB and *sql.Tx.
// Re-exported here so dashboard aggregate callers in this package and in the
// domain adapters can name a single type without importing the internal
// interface package directly.
type DBExecutor = interfaces.DBExecutor

// RunDashboardAggregate runs ONE row-returning aggregate query and scans the
// single result row into dest in column order.
//
// SQL Server note: the SQL passed to this helper must use CASE aggregation
// (SUM(CASE WHEN cond THEN expr END)) rather than the postgres-only
// FILTER (WHERE ...) clause. The SQL is caller-supplied; this function is
// dialect-agnostic Go — the dialect difference lives in the SQL text, not here.
//
// Why this exists — the fail-open-per-metric anti-pattern:
//
// Dashboard adapters historically issued N separate QueryRowContext(...).Scan
// calls (one per scalar metric), swallowing each error as `return 0, nil`. A
// transient DB fault then silently rendered a dashboard full of zeros. This
// helper makes that pattern structurally impossible:
//
//   - sql.ErrNoRows → treated as a zeroed result (dest left untouched, nil err).
//     An aggregate over zero rows still returns one row, so ErrNoRows means the
//     aggregate produced no row; callers want zero-valued dest, not an error.
//   - any other error → returned verbatim (wrapped) so the caller can fail the
//     whole dashboard rather than paint a false all-zeros picture.
//
// Consolidating N metrics into one multi-aggregate CTE and calling this once
// removes both the N round-trips and the N independent fail-open seams.
//
// dest must be the scan targets in the same column order the query SELECTs.
func RunDashboardAggregate(
	ctx context.Context,
	db DBExecutor,
	query string,
	args []any,
	dest ...any,
) error {
	if db == nil {
		return fmt.Errorf("dashboard aggregate: database connection is not available")
	}

	err := db.QueryRowContext(ctx, query, args...).Scan(dest...)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sql.ErrNoRows):
		// No row from the aggregate — leave dest at its zero values and report
		// success. This is the ONLY error class treated as a zeroed result.
		return nil
	default:
		return fmt.Errorf("dashboard aggregate query failed: %w", err)
	}
}
