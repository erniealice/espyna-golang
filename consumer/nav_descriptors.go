// Package consumer — nav_descriptors.go
//
// Framework-level nav descriptor utilities. The NoopRegistrar satisfies
// view.RouteRegistrar for compose.Engine.Assemble when units have no Mount
// functions and therefore never register routes.
//
// Moved from apps/service-admin/internal/composition/ — this is a framework
// concern with no app-internal dependencies.

package consumer

import (
	"github.com/erniealice/pyeza-golang/view"
)

// NoopRegistrar satisfies view.RouteRegistrar for compose.Engine.Assemble
// when the units being assembled have no Mount functions and therefore never
// register routes. Used by app-level Nav unit assembly.
type NoopRegistrar struct{}

func (NoopRegistrar) GET(_ string, _ view.View, _ ...string)  {}
func (NoopRegistrar) POST(_ string, _ view.View, _ ...string) {}
