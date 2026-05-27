//go:build sqlserver

package fulfillment

import (
	fulfillmentdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
)

// Compile-time assertions: every SQL Server fulfillment-dashboard repo MUST
// satisfy the corresponding service-layer dashboard repository interface.
//
// See contrib/postgres/internal/adapter/fulfillment/fulfillment_dashboard_assertions.go
// for the full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20).
var (
	_ fulfillmentdash.FulfillmentDashboardRepository = (*SQLServerFulfillmentRepository)(nil)
)
