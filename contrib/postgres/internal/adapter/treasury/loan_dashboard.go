//go:build postgresql

package treasury

import (
	"context"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// TimeBucket is aliased to the service-layer Go-only TimeBucket so the
// adapter's dashboard methods satisfy the service-layer
// `LoanDashboardRepository` / `CollectionDashboardRepository` interfaces
// EXACTLY (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20). Using a
// fresh local type would silently break the runtime type assertion in
// `internal/composition/core/initializers/service.go`. Wave B P1.C.5
// adopted the same aliasing pattern from the Location pilot
// (P1.C.2) — see `internal/application/usecases/service/dashboard/
// location/get_location_dashboard.go` for the canonical doc-comment.
type TimeBucket = treasurydash.TimeBucket

// LoanSlice is aliased to the service-layer Go-only LoanSlice for the same
// reason as TimeBucket above. The runtime assertion in
// `initializers/service.go` requires exact named return type match.
type LoanSlice = treasurydash.LoanSlice

// loanScalarAggregate is the ONE consolidated scalar-count CTE for the loan
// dashboard (Q-DASHBOARD-FAILOPEN, Option A). A single `base` CTE selects the
// active, workspace-scoped loan rows once; the outer SELECT derives every
// scalar metric — total outstanding + per-status counts — via conditional
// aggregation over that single pass.
//
// Both SumOutstanding and CountByStatus delegate here so the dashboard's
// scalar metrics travel through ONE multi-aggregate query with ONE honest
// error seam (via core.RunDashboardAggregate), instead of N separate
// QueryRowContext helpers that each swallowed errors as (0, nil). centavos
// stay centavos — remaining_balance is summed, never scaled.
type loanScalarAggregate struct {
	TotalOutstanding int64
	ActiveCount      int64
	CompletedCount   int64
	DefaultedCount   int64
}

// loanScalarAggregateQuery centralizes the consolidated CTE. workspace_id is
// $1 in every branch (multi-tenancy guardrail). The base CTE is the single
// scan surface; FILTER (WHERE ...) derives each per-status count without a
// second round trip.
const loanScalarAggregateQuery = `
	WITH base AS (
		SELECT l.status, l.remaining_balance
		FROM loan l
		WHERE l.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
	)
	SELECT
		COALESCE(SUM(remaining_balance), 0)::bigint                      AS total_outstanding,
		COUNT(*) FILTER (WHERE status = 'ACTIVE')::bigint                AS active_count,
		COUNT(*) FILTER (WHERE status = 'COMPLETED')::bigint             AS completed_count,
		COUNT(*) FILTER (WHERE status = 'DEFAULTED')::bigint             AS defaulted_count
	FROM base`

// runLoanScalarAggregate executes the consolidated CTE once and returns the
// honest error. Workspace-scoped, centavos.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_loan_workspace_active
//	  ON loan(workspace_id, active);
func (r *PostgresLoanRepository) runLoanScalarAggregate(
	ctx context.Context,
	workspaceID string,
) (loanScalarAggregate, error) {
	var agg loanScalarAggregate
	if err := postgresCore.RunDashboardAggregate(
		ctx,
		r.db,
		loanScalarAggregateQuery,
		[]any{workspaceID},
		&agg.TotalOutstanding,
		&agg.ActiveCount,
		&agg.CompletedCount,
		&agg.DefaultedCount,
	); err != nil {
		return loanScalarAggregate{}, err
	}
	return agg, nil
}

// SumOutstanding returns the sum of remaining_balance across all active loans
// (centavos). Workspace-scoped. Routes through the consolidated scalar CTE so
// the metric shares ONE honest error seam with the per-status counts.
func (r *PostgresLoanRepository) SumOutstanding(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	agg, err := r.runLoanScalarAggregate(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	return agg.TotalOutstanding, nil
}

// CountByStatus returns a map of loan status (ACTIVE/COMPLETED/DEFAULTED) to
// count. Workspace-scoped. Routes through the consolidated scalar CTE — no
// separate GROUP BY round trip, no fail-open. Errors propagate honestly so a
// DB fault fails the dashboard instead of painting a false all-zeros picture.
func (r *PostgresLoanRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	agg, err := r.runLoanScalarAggregate(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		"ACTIVE":    agg.ActiveCount,
		"COMPLETED": agg.CompletedCount,
		"DEFAULTED": agg.DefaultedCount,
	}, nil
}

// SumInterestAccruedYTD sums recorded loan_payment.interest_amount year-to-date
// for loans in this workspace (centavos). This reads a different table
// (loan_payment) so it stays a separate single-aggregate query, but it now
// propagates its error honestly via core.RunDashboardAggregate instead of
// swallowing it as (0, nil). Workspace-scoped.
func (r *PostgresLoanRepository) SumInterestAccruedYTD(
	ctx context.Context,
	workspaceID string,
	year int,
) (int64, error) {
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	yearEnd := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC).Format("2006-01-02")

	const query = `
		SELECT COALESCE(SUM(lp.interest_amount), 0)::bigint
		FROM loan_payment lp
		JOIN loan l ON l.id = lp.loan_id
		WHERE lp.payment_date >= $2
		  AND lp.payment_date <= $3
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)`

	var total int64
	if err := postgresCore.RunDashboardAggregate(
		ctx,
		r.db,
		query,
		[]any{workspaceID, yearStart, yearEnd},
		&total,
	); err != nil {
		return 0, err
	}
	return total, nil
}

// TopByOutstanding returns active loans ranked by remaining_balance DESC.
// Workspace-scoped. List-shaped (multi-row) so it stays a separate
// QueryContext helper — but it now returns DB errors honestly instead of
// `return nil, nil`.
func (r *PostgresLoanRepository) TopByOutstanding(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]LoanSlice, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	const query = `
		SELECT
			l.id,
			l.loan_number,
			l.lender_name,
			l.remaining_balance,
			l.principal_amount,
			l.status
		FROM loan l
		WHERE l.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
		ORDER BY l.remaining_balance DESC NULLS LAST
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top loans: %w", err)
	}
	defer rows.Close()

	out := make([]LoanSlice, 0, limit)
	for rows.Next() {
		var row LoanSlice
		if scanErr := rows.Scan(
			&row.ID,
			&row.LoanNumber,
			&row.LenderName,
			&row.RemainingBalance,
			&row.PrincipalAmount,
			&row.Status,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan top loan row: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top loan rows: %w", err)
	}
	return out, nil
}

// OutstandingPrincipalByMonth returns one TimeBucket per month in the range,
// snapshotting SUM(remaining_balance) at month-end. The implementation here
// uses a synthetic generate_series + a join to active loans (treating
// remaining_balance as the current snapshot on every bucket — a refinement
// would track historic balances via a payments-applied-by-month CTE).
//
// List-shaped (one row per month) so it stays a separate QueryContext helper,
// but it now returns DB errors honestly instead of `return nil, nil`.
// Workspace-scoped, centavos.
func (r *PostgresLoanRepository) OutstandingPrincipalByMonth(
	ctx context.Context,
	workspaceID string,
	from, to time.Time,
) ([]TimeBucket, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if from.After(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	const query = `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', $2::timestamp),
				date_trunc('month', $3::timestamp),
				interval '1 month'
			) AS bucket
		),
		paid_after AS (
			SELECT m.bucket,
			       COALESCE(SUM(lp.principal_amount), 0)::bigint AS paid_principal
			FROM months m
			LEFT JOIN loan_payment lp
			  ON lp.payment_date::timestamp >= m.bucket + interval '1 month'
			LEFT JOIN loan l ON l.id = lp.loan_id
			WHERE ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1 OR l.workspace_id IS NULL)
			GROUP BY m.bucket
		),
		active_principal AS (
			SELECT COALESCE(SUM(l.remaining_balance), 0)::bigint AS now_balance
			FROM loan l
			WHERE l.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
		)
		SELECT m.bucket,
		       (ap.now_balance + COALESCE(p.paid_principal, 0))::bigint
		FROM months m
		CROSS JOIN active_principal ap
		LEFT JOIN paid_after p ON p.bucket = m.bucket
		ORDER BY m.bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query outstanding-by-month: %w", err)
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var (
			bucket time.Time
			value  int64
		)
		if scanErr := rows.Scan(&bucket, &value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan outstanding-by-month row: %w", scanErr)
		}
		out = append(out, TimeBucket{Period: bucket, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outstanding-by-month rows: %w", err)
	}
	return out, nil
}
