package infrastructure

import "github.com/erniealice/espyna-golang/reference"

// ReferenceChecker re-exports the public reference.Checker contract so
// existing internal imports keep working without modification.
// New code should import github.com/erniealice/espyna-golang/reference directly.
type ReferenceChecker = reference.Checker

// NewNoOpReferenceChecker returns a checker that reports nothing in use.
// Useful as a sane default in non-postgres providers and tests that don't
// care about reference checks.
func NewNoOpReferenceChecker() ReferenceChecker { return reference.NewNoOp() }
