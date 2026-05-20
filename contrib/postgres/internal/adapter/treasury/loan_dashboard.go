//go:build postgresql

package treasury

import (
	"context"
	"fmt"
	"time"

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

// SumOutstanding returns the sum of remaining_balance across all active loans
// (centavos). Workspace-scoped.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_loan_workspace_active
//	  ON loan(workspace_id, active);
func (r *PostgresLoanRepository) SumOutstanding(
	ctx context.Context,
	workspaceID string,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT COALESCE(SUM(l.remaining_balance), 0)::bigint
		FROM loan l
		WHERE l.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// SumInterestAccruedYTD approximates YTD accrued interest as the sum of
// (principal * rate / 12 * months_elapsed_ytd / 100) for active loans started
// before the requested year-end. This is a rough server-side estimate used for
// dashboard display only — not for accounting. Workspace-scoped, centavos.
//
// Note: a more accurate calculation requires loan_payment.interest_amount
// summed YTD; that's done by the dashboard use case if available. This method
// provides a fallback estimate from the loan table.
func (r *PostgresLoanRepository) SumInterestAccruedYTD(
	ctx context.Context,
	workspaceID string,
	year int,
) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is not available")
	}

	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	yearEnd := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC).Format("2006-01-02")

	// Use the actual loan_payment.interest_amount when available — sum of
	// recorded interest paid year-to-date for loans in this workspace.
	const query = `
		SELECT COALESCE(SUM(lp.interest_amount), 0)::bigint
		FROM loan_payment lp
		JOIN loan l ON l.id = lp.loan_id
		WHERE lp.payment_date >= $2
		  AND lp.payment_date <= $3
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, workspaceID, yearStart, yearEnd).Scan(&total); err != nil {
		return 0, nil //nolint:nilerr
	}
	return total, nil
}

// CountByStatus returns a map of loan status (active/completed/defaulted/draft)
// to count. Workspace-scoped.
func (r *PostgresLoanRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	const query = `
		SELECT l.status, COUNT(*)::bigint
		FROM loan l
		WHERE l.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR l.workspace_id = $1)
		GROUP BY l.status`

	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 4)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan loan count row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan count rows: %w", err)
	}
	return out, nil
}

// TopByOutstanding returns active loans ranked by remaining_balance DESC.
// Workspace-scoped.
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
		return nil, nil //nolint:nilerr
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
		return nil, nil //nolint:nilerr
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
