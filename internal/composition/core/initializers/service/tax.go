package service

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"
	servicetax "github.com/erniealice/espyna-golang/internal/application/usecases/service/tax"
)

// initServiceTax wires the service-driven Tax sub-aggregate.
// Per Q-SDM-TAX (20260520), this wraps the entity-layer
// ComputeTaxesForRevenue use case with a proto contract.
//
// Returns nil if entityCompute is nil (e.g., no SQL provider or
// tax aggregate initialization failed). The recognize-revenue hook
// degrades gracefully when nil.
func initServiceTax(entityCompute *compute_taxes_for_revenue.ComputeTaxesForRevenueUseCase) *servicetax.UseCases {
	if entityCompute == nil {
		return nil
	}
	return servicetax.NewUseCases(servicetax.Repositories{EntityCompute: entityCompute})
}
