// Package tax hosts the service-driven tax use case sub-aggregate.
//
// Per Q-SDM-TAX (LOCKED 2026-05-20), tax compute satisfies Q7 §8 signal 1
// (cross-domain read from revenue + revenue_line_item + workspace + product
// + tax_* + withholding_certificate domains) plus signal 2 (append-only
// revenue_tax_line writes as side-effect). It belongs in the service-driven
// layer, not the entity-driven layer.
//
// This sub-aggregate formalizes the previously Go-only Request/Response
// types at packages/espyna-golang/internal/application/usecases/tax/
// compute_taxes_for_revenue/compute.go:92, :108 as proto messages
// (proto/v1/service/tax/compute.proto). The entity-layer use case is
// retained as the algorithmic implementation; the service-layer wrapper
// here translates proto messages to/from that use case.
package tax

import (
	taxcompute "github.com/erniealice/espyna-golang/internal/application/usecases/tax/compute_taxes_for_revenue"
)

// UseCases aggregates every service-driven tax use case.
type UseCases struct {
	ComputeTaxesForRevenue *ComputeTaxesForRevenueUseCase
}

// Repositories groups the entity-layer dependencies the wrapper needs.
type Repositories struct {
	// EntityCompute is the entity-layer ComputeTaxesForRevenueUseCase
	// that does the actual algorithmic work. The wrapper translates
	// proto messages to/from this use case's Go-shaped Request/Response.
	// May be nil when no SQL provider is registered; the wrapper returns
	// an error in that case.
	EntityCompute *taxcompute.ComputeTaxesForRevenueUseCase
}

// NewUseCases wires every tax service use case from shared dependencies.
func NewUseCases(repositories Repositories) *UseCases {
	return &UseCases{
		ComputeTaxesForRevenue: NewComputeTaxesForRevenueUseCase(repositories.EntityCompute),
	}
}
