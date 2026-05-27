//go:build sqlserver

package product

import (
	productdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/product"
)

// Compile-time assertions: every sqlserver product-dashboard repo MUST satisfy
// the corresponding service-layer dashboard repository interface.
// Mirrors the postgres gold standard — see product_dashboard_assertions.go for
// the full rationale (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS — LOCKED 2026-05-20).
var (
	_ productdash.ProductDashboardRepository = (*SQLServerProductRepository)(nil)
)
