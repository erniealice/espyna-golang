//go:build sqlserver

package expenditure

import (
	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
)

// Compile-time assertions: every SQL Server expenditure-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// **Why these are MANDATORY** (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20):
//
// The expenditure dashboard composition root performs runtime type assertions
// to thread this adapter into the service-layer use case. A type assertion
// silently fails — returning `(nil, false)` — when the adapter's method
// signatures drift from the interface (e.g. a return type named
// `expenditure.TopSupplierRow` instead of `expendituredash.TopSupplierRow`).
//
// **Required adapter-side aliases (declared in expenditure_dashboard.go):**
//
//   - `type TimeBucket = expendituredash.TimeBucket`
//   - `type TopSupplierRow = expendituredash.TopSupplierRow`
//
// Without these aliases, SumByMonth and TopBySupplier return the wrong named
// type and the interface satisfaction silently fails. The assertion below
// catches the drift at build time.
var (
	_ expendituredash.ExpenditureDashboardRepository = (*SQLServerExpenditureRepository)(nil)
)
