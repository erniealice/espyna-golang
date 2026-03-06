// Package customization re-exports internal route customization types for use by contrib sub-modules.
// Contrib packages (which are separate Go modules) cannot import internal/ directly,
// so this package provides stable public aliases.
package customization

import (
	internal "github.com/erniealice/espyna-golang/internal/composition/routing/customization"
)

// RouteCustomizer manages route path customizations.
type RouteCustomizer = internal.RouteCustomizer

// CustomizationConfig holds all path overrides (for YAML/JSON loading).
type CustomizationConfig = internal.CustomizationConfig

// NewRouteCustomizer creates a new route customizer.
var NewRouteCustomizer = internal.NewRouteCustomizer
