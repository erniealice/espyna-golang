// Package service hosts the service-driven domain use case sub-aggregates.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// LOCKED), service-driven domains are first-class but distinct from the
// entity-driven 14-layer canon: they have proto contracts and use cases
// but no entityid/provider/route. This package is the Layer-7 anchor for
// that category. Phase 1 anchors it with one sub-aggregate (audit); the
// follow-up plan migrates reporting/auth/security here too.
package service

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
)

// ServiceUseCases aggregates every service-driven use case package.
type ServiceUseCases struct {
	Audit *audit.UseCases
}

// NewServiceUseCases wires every service-driven sub-aggregate from its
// constituent use cases. Sub-aggregates may be nil when the relevant
// infrastructure provider is unregistered.
func NewServiceUseCases(audit *audit.UseCases) *ServiceUseCases {
	return &ServiceUseCases{Audit: audit}
}
