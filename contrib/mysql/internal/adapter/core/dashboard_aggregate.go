//go:build mysql

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
// This helper is dialect-agnostic Go: the differences between postgres and MySQL
// live in the SQL string the caller passes in, not in this scan logic. In
// particular, MySQL callers supply CASE-based conditional aggregation:
//
//	SUM(CASE WHEN cond THEN expr END)
//
// instead of postgres's FILTER-based form:
//
//	SUM(expr) FILTER (WHERE cond)
//
// See MySQLDialect.ConditionalSum for the canonical rendering helper.
//
// Why this exists — the fail-open-per-metric anti-pattern:
//
// Dashboard adapters historically issued N separate QueryRowContext(...).Scan
// helpers (one per scalar metric), and each helper swallowed its error as
// `return 0, nil`. A transient DB fault then silently rendered a dashboard full
// of zeros that looked like real data. This helper makes that pattern structurally
// impossible by funnelling every scalar dashboard metric through a SINGLE
// round-trip whose error is propagated honestly:
//
//   - sql.ErrNoRows  → treated as a zeroed result (dest left untouched, nil err).
//     An aggregate like COUNT(*)/SUM(...) over zero rows still returns one row,
//     so ErrNoRows here means the aggregate genuinely produced no row; callers
//     want zero-valued dest in that case, not an error.
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
