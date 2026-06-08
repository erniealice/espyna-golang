// Package amortization hosts the service-driven amortization schedule use
// case sub-aggregate.
//
// Promoted from internal/application/shared/amortize_schedule/ — the pure-math
// leaf scored 3/3 on the Mantra guide (4 consumers across 3 domains, cross-
// domain read/compute). The Go implementation remains in the shared package
// as the computation engine; the service-layer wrapper here translates proto
// messages to/from that engine.
//
// NOTE: pnpm build must be run in packages/esqyma/ to generate the proto
// types before this package compiles. The import path
// github.com/erniealice/esqyma/pkg/schema/v1/service/amortization will not
// resolve until generation completes.
package amortization

// UseCases aggregates every service-driven amortization use case.
type UseCases struct {
	EnumerateTranches      *EnumerateTranchesUseCase
	ComputeNextDueTranche  *ComputeNextDueTrancheUseCase
}

// NewUseCases wires both amortization service use cases. No DB or repo deps —
// this is pure computation.
func NewUseCases() *UseCases {
	return &UseCases{
		EnumerateTranches:     NewEnumerateTranchesUseCase(),
		ComputeNextDueTranche: NewComputeNextDueTrancheUseCase(),
	}
}
