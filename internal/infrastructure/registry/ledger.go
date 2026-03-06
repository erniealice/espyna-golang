package registry

import "sync"

// =============================================================================
// Ledger Reporting Factory Registry
// =============================================================================
//
// LedgerReportingFactory provides self-registration for ledger reporting service
// implementations. This allows contrib sub-modules (e.g. contrib/postgres) to
// register their concrete LedgerReportingService implementation at init() time,
// and the consumer package can discover it at runtime without build tags.
//
// The factory uses `any` parameters because the concrete type
// (consumer.LedgerReportingService) is defined in the consumer package and
// cannot be imported here without creating a circular dependency.
//
// =============================================================================

// ledgerReportingRegistry holds the registered ledger reporting factory
var ledgerReportingRegistry = struct {
	factory func(db any, config any) any
	mutex   sync.RWMutex
}{}

// RegisterLedgerReportingFactory registers a factory for creating LedgerReportingService.
// This is called from init() in provider-specific packages (e.g. contrib/postgres).
//
// Example:
//
//	func init() {
//	    registry.RegisterLedgerReportingFactory(func(db any, config any) any {
//	        sqlDB := db.(*sql.DB)
//	        cfg := config.(consumer.LedgerReportingTableConfig)
//	        return ledgeradapter.NewLedgerReportingAdapter(sqlDB, ...)
//	    })
//	}
func RegisterLedgerReportingFactory(factory func(db any, config any) any) {
	ledgerReportingRegistry.mutex.Lock()
	defer ledgerReportingRegistry.mutex.Unlock()

	if factory == nil {
		panic("RegisterLedgerReportingFactory: factory is nil")
	}
	ledgerReportingRegistry.factory = factory
}

// GetLedgerReportingFactory retrieves the registered ledger reporting factory.
// Returns (factory, true) if registered, (nil, false) otherwise.
func GetLedgerReportingFactory() (func(db any, config any) any, bool) {
	ledgerReportingRegistry.mutex.RLock()
	defer ledgerReportingRegistry.mutex.RUnlock()

	return ledgerReportingRegistry.factory, ledgerReportingRegistry.factory != nil
}
