//go:build postgresql

package ledger

import "fmt"

// buildClientBalancesQuery returns a SQL query that computes the outstanding
// balance (total billed minus total received) for every client that has at least
// one non-cancelled, non-draft revenue. Clients with a net-zero balance
// are excluded via the HAVING clause.
//
// Parameters:
//
//	$1 = workspace_id (text or NULL)
//
// Returns two columns: client_id (text), outstanding (int8 centavos).
func buildClientBalancesQuery(tc TableConfig) (string, []any) {
	return fmt.Sprintf(`
SELECT
    r.client_id,
    COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) AS outstanding
FROM %s r
LEFT JOIN (
    SELECT tc.revenue_id, SUM(tc.amount) AS total_received
    FROM %s tc
    WHERE tc.active = true AND tc.status IN ('paid', 'completed')
    GROUP BY tc.revenue_id
) received ON received.revenue_id = r.id
WHERE r.active = true
  AND r.status NOT IN ('cancelled', 'draft')
  AND r.client_id IS NOT NULL
  AND ($1::text IS NULL OR r.workspace_id = $1)
GROUP BY r.client_id
HAVING COALESCE(SUM(r.total_amount), 0) - COALESCE(SUM(received.total_received), 0) != 0`,
		tc.Revenue, tc.TreasuryCollection), nil
}