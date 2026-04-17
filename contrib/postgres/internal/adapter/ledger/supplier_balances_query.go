//go:build postgresql

package ledger

import "fmt"

// buildSupplierBalancesQuery returns a SQL query that computes the outstanding
// balance (total billed minus total paid) for every supplier that has at least
// one non-cancelled, non-draft expenditure. Suppliers with a net-zero balance
// are excluded via the HAVING clause.
//
// Parameters:
//
//	$1 = workspace_id (text or NULL)
//
// Returns two columns: supplier_id (text), outstanding (int8 centavos).
func buildSupplierBalancesQuery(tc TableConfig) (string, []any) {
	return fmt.Sprintf(`
SELECT
    e.supplier_id,
    COALESCE(SUM(e.total_amount), 0) - COALESCE(SUM(paid.total_paid), 0) AS outstanding
FROM %s e
LEFT JOIN (
    SELECT d.expenditure_id, SUM(d.amount) AS total_paid
    FROM %s d
    WHERE d.active = true AND d.status IN ('paid', 'completed')
    GROUP BY d.expenditure_id
) paid ON paid.expenditure_id = e.id
WHERE e.active = true
  AND e.status NOT IN ('cancelled', 'draft')
  AND e.supplier_id IS NOT NULL
  AND ($1::text IS NULL OR e.workspace_id = $1)
GROUP BY e.supplier_id
HAVING COALESCE(SUM(e.total_amount), 0) - COALESCE(SUM(paid.total_paid), 0) != 0`,
		tc.Expenditure, tc.TreasuryDisbursement), nil
}