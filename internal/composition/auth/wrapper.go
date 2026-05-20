// Package auth hosts composition-level helpers that adapt the entity-layer
// usecases/auth.UseCases to the service-driven proto contract at
// proto/v1/service/auth/.
//
// Per docs/plan/20260520-service-domain-migration/ (Wave 3 / Plan 2
// auth-session candidate, 2026-05-20), the previous
// `consumer/auth_aliases.go` Go type-alias visibility bridge has been
// retired. Its replacement is the service-driven sub-aggregate
// (usecases/service/auth/UseCases), exposed as a typed field on
// `*service.ServiceUseCases.Auth` (matching the existing Audit and
// Security sub-aggregates) so apps can reach it via chained field
// access without importing `internal/`.
//
// The typed-field path is used (rather than the dynamic registry) for
// app-visible candidates because Go's `internal/` rule prevents apps
// from calling `service.Get[*auth.UseCases]("auth")` directly — the
// generic instantiation requires importing the (internal) `service/auth`
// package to name the type parameter. A typed field on
// `*service.ServiceUseCases` is reachable via field chain
// (`useCases.Service.Auth`) without naming the internal type.
//
// This package is the sibling to `composition/security/` from the
// permission_query candidate (2026-05-20).
package auth

import (
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
)

// BuildServiceAuthUseCases assembles the service-driven Auth/Session
// sub-aggregate by wrapping the entity-layer Auth use cases (built by
// InitializeAuth) into the proto-shaped Request/Response surface
// defined at proto/v1/service/auth/session.proto.
//
// Returns a non-nil *serviceauth.UseCases even when entityAuth is nil —
// each wrapped use case carries a per-call nil-inner guard that returns
// a translated "service unavailable" error so callers can degrade
// gracefully rather than crash on nil-pointer deref.
//
// Called from initializers/service.go.
func BuildServiceAuthUseCases(entityAuth *entityauth.UseCases, deps *service.Deps) *serviceauth.UseCases {
	svc := serviceauth.Services{}
	if deps != nil {
		svc.TranslationService = deps.TranslationService
	}
	if entityAuth == nil {
		return &serviceauth.UseCases{
			AuthenticateSession: serviceauth.NewAuthenticateSessionUseCase(nil, svc),
			IssueSession:        serviceauth.NewIssueSessionUseCase(nil, svc),
			InvalidateSession:   serviceauth.NewInvalidateSessionUseCase(nil, svc),
		}
	}
	return &serviceauth.UseCases{
		AuthenticateSession: serviceauth.NewAuthenticateSessionUseCase(entityAuth.AuthenticateSession, svc),
		IssueSession:        serviceauth.NewIssueSessionUseCase(entityAuth.IssueSession, svc),
		InvalidateSession:   serviceauth.NewInvalidateSessionUseCase(entityAuth.InvalidateSession, svc),
	}
}
