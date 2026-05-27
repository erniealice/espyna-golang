//go:build mysql

package expenditure

import (
	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
)

// Compile-time assertions: the MySQL expenditure-dashboard repo MUST satisfy
// the service-layer dashboard repository interface (mirrors the postgres
// assertion — Q-SDM-DASHBOARD-COMPILE-ASSERTIONS).
//
// Named-type aliases (TimeBucket, TopSupplierRow) in expenditure_dashboard.go
// ensure method signatures match exactly; this var line enforces that at build
// time.
var (
	_ expendituredash.ExpenditureDashboardRepository = (*MySQLExpenditureRepository)(nil)
)
