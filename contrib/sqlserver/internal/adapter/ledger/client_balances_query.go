//go:build sqlserver

package ledger

import "fmt"

// buildClientBalancesQuery returns a SQL query that computes the outstanding
// balance (total billed minus total received) for every client that has at least
// one non-cancelled, non-draft revenue. Clients with a net-zero balance
// are excluded via the HAVING clause.
//
// Parameters:
//
//	@p1 = workspace_id (text or NULL)
//
// SQL Server differences from the postgres gold standard:
//   - $1 → @p1 placeholder.
//   - active = true → active = 1.
//   - ($1::text IS NULL OR …) → (@p1 IS NULL OR …).
//
// Returns two columns: client_id (text), outstanding (bigint centavos).
func buildClientBalancesQuery(tc TableConfig) (string, []any) {
	return fmt.Sprintf(`
SELECT
    r.client_id,
    COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) AS outstanding
FROM %s r
LEFT JOIN (
    SELECT tc.revenue_id, SUM(tc.amount) AS total_received
    FROM %s tc
    WHERE tc.active = 1 AND tc.status IN ('paid', 'completed')
    GROUP BY tc.revenue_id
) received ON received.revenue_id = r.id
WHERE r.active = 1
  AND r.status NOT IN ('cancelled', 'draft')
  AND r.client_id IS NOT NULL
  AND (@p1 IS NULL OR r.workspace_id = @p1)
GROUP BY r.client_id
HAVING COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) != 0`,
		tc.Revenue, tc.TreasuryCollection), nil
}
