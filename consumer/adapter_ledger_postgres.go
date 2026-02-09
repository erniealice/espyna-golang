//go:build postgres

package consumer

import (
	"database/sql"

	ledgeradapter "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/ledger"
)

// NewLedgerReportingService creates a new ledger reporting service backed by PostgreSQL.
// It spans multiple tables (revenue, inventory, product, location) to compute
// gross profit reports with configurable group-by dimensions.
//
// The returned service satisfies any interface with the GetGrossProfitReport method,
// including fycha.DataSource for direct use in report views.
func NewLedgerReportingService(db *sql.DB, config LedgerReportingTableConfig) LedgerReportingService {
	return ledgeradapter.NewLedgerReportingAdapter(db, ledgeradapter.TableConfig{
		Revenue:              config.Revenue,
		RevenueLineItem:      config.RevenueLineItem,
		InventoryTransaction: config.InventoryTransaction,
		InventoryItem:        config.InventoryItem,
		Product:              config.Product,
		Location:             config.Location,
		RevenueCategory:      config.RevenueCategory,
	})
}
