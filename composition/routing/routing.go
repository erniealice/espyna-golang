// Package routing re-exports internal composition routing types for use by contrib sub-modules.
// Contrib packages (which are separate Go modules) cannot import internal/ directly,
// so this package provides stable public aliases.
package routing

import (
	internal "github.com/erniealice/espyna-golang/internal/composition/routing"
)

// Route is a framework-agnostic route definition (aliased from contracts).
type Route = internal.Route

// RouteMetadata contains additional information about a route.
type RouteMetadata = internal.RouteMetadata

// RouteGroup represents a collection of related routes.
type RouteGroup = internal.RouteGroup

// Config holds routing configuration.
type Config = internal.Config
