//go:build !postgresql

package consumer

import "database/sql"

// NewLedgerReportingService returns nil when postgres is not available.
// The consumer app should handle nil gracefully (reports will be unavailable).
func NewLedgerReportingService(_ *sql.DB, _ LedgerReportingTableConfig) LedgerReportingService {
	return nil
}
