package service

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/amortization"
)

// initServiceAmortization wires the service-driven Amortization sub-aggregate.
// Pure computation — no database dependencies. Always returns a non-nil
// *amortization.UseCases that enumerates tranches and computes next-due values.
func initServiceAmortization() *amortization.UseCases {
	return amortization.NewUseCases()
}
