// Package auth hosts the service-driven Auth/Session use cases.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// service-driven domain category) and the follow-up migration plan
// docs/plan/20260520-service-domain-migration/ (Wave 3 / Plan 2 candidate,
// 2026-05-20), session validation/issuance/invalidation are service-shape
// operations: they operate over the existing `domain.entity.v1.Session`
// aggregate but own no aggregate of their own. That makes them a
// service-driven domain, not entity-driven.
//
// Their proto contract lives at `proto/v1/service/auth/session.proto`.
// These use cases WRAP the existing entity-layer
// `internal/application/usecases/auth.UseCases` (which take Go struct
// requests built around `sessionpb` types) and adapt the proto-shaped
// Request/Response surface required by service-driven consumers — the
// previous visibility bridge was `consumer/auth_aliases.go`, which this
// migration retires.
//
// Wiring: this package defines ONLY the type surface so that the parent
// `service` package can import it for the typed `*ServiceUseCases.Auth`
// field (matching the existing Audit/Security pattern). Construction
// lives in `internal/composition/core/initializers/service/auth.go`
// (`initServiceAuth`) — a fused initializer (Option B, 20260521-composition-
// reshape Q-CR8) that builds the entity-layer `*auth.UseCases` AND wires
// the proto-shaped wrappers in a single function. The former separate
// `internal/composition/auth/wrapper.go` and `composition/auth/` directory
// are DELETED.
//
// The typed-field path is used (rather than the Q-ORCH-2 dynamic
// registry at `service.Register`) because Go's `internal/` rule
// prevents apps from naming `*service/auth.UseCases` as a generic type
// parameter to call `service.Get[*UseCases]`. The typed field is
// reachable from apps via chained field access through
// `consumer.UseCases.Service.Auth` without naming the internal type.
package auth

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// UseCases aggregates every service-driven Auth/Session use case.
type UseCases struct {
	AuthenticateSession *AuthenticateSessionUseCase
	IssueSession        *IssueSessionUseCase
	InvalidateSession   *InvalidateSessionUseCase
}

// Services groups application services consumed by every wrapper.
// Translator backs error-message localization; no
// Authorizer — these use cases establish/terminate identity, so
// per the invariant on usecases/auth/usecases.go authcheck cannot apply.
type Services struct {
	Translator ports.Translator
}
