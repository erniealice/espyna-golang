package ports

import "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"

// Checker re-exports the public FK reference-checking contract so separate Go
// modules (contrib, blocks) can reach it without importing internal/.
type Checker = infrastructure.Checker

// NewNoOp returns a Checker that reports nothing in use and never errors.
// Useful as a sane default in non-postgres providers and tests that don't
// care about reference checks.
func NewNoOp() infrastructure.Checker { return infrastructure.NewNoOp() }
