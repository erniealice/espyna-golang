//go:build postgresql

package revenue

import (
	"context"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// LockRevenueForUpdate takes one revenue row FOR UPDATE inside the ambient transaction
// (domainports.RevenueLocker). The workspace predicate is part of the lock statement; a foreign or
// missing row is "not found", and without an ambient transaction it fails closed. Receive & apply
// locks the client's open invoices with it (20260927-usage-and-pass-through-charges, build-spec §7 C4/C5).
func (r *PostgresRevenueRepository) LockRevenueForUpdate(ctx context.Context, id string) (*revenuepb.Revenue, error) {
	if err := postgresCore.LockScopedRowForUpdate(ctx, r.dbOps, r.tableName, id); err != nil {
		return nil, err
	}
	resp, err := r.ReadRevenue(ctx, &revenuepb.ReadRevenueRequest{Data: &revenuepb.Revenue{Id: id}})
	if err != nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, fmt.Errorf("revenue lock: not found")
	}
	return resp.Data[0], nil
}
