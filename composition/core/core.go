// Package core re-exports internal composition core types for use by contrib sub-modules.
// Contrib packages (which are separate Go modules) cannot import internal/ directly,
// so this package provides stable public aliases.
package core

import (
	internal "github.com/erniealice/espyna-golang/internal/composition/core"
)

// Container is the main dependency injection container.
type Container = internal.Container
