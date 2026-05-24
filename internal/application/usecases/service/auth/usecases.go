// Package auth hosts the service-driven Auth/Session use cases.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// service-driven domain category) and docs/wiki/articles/proto-categories.md
// §2 ("Why auth lives in service/ and not in domain/"), session
// authentication / issuance / invalidation are application-service
// operations: they coordinate the existing `domain.entity.v1.Session`
// and `domain.entity.v1.User` aggregates but own no aggregate of their
// own. The proto contract lives at `proto/v1/service/auth/session.proto`.
//
// Wiring lives in `internal/composition/core/initializers/service/auth.go`
// (`initServiceAuth`). This package owns the use-case bodies directly —
// it consumes the proto repositories (`sessionpb.SessionDomainServiceServer`,
// `userpb.UserDomainServiceServer`) without a Go-struct intermediate layer.
//
// Invariant: every file in this package must either establish identity
// (authenticate_session, issue_session — future: login, register,
// request_password_reset, execute_password_reset) or terminate an
// established session (invalidate_session — future: rotate_session). This
// is why the authcheck coverage test skips this directory — these use
// cases run BEFORE authorization can be applied or AFTER it has been
// revoked. Authenticated business operations that are merely auth-adjacent
// (e.g. "admin revokes another user's sessions") belong in
// usecases/domain/entity/session/ with authcheck wired in.
package auth

import (
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// SessionExpiryConfig lets the caller override the default session TTL. A
// zero Duration means IssueSession falls back to defaultSessionExpiry.
type SessionExpiryConfig struct {
	Duration time.Duration
}

// UseCases aggregates every service-driven Auth/Session use case.
type UseCases struct {
	AuthenticateSession *AuthenticateSessionUseCase
	IssueSession        *IssueSessionUseCase
	InvalidateSession   *InvalidateSessionUseCase
	SwitchPrincipal     *SwitchPrincipalUseCase
}

// Repositories groups proto-level domain services needed by auth flows.
// Any field may be nil; each Execute fails closed with service_unavailable
// when the repository it requires is nil.
//
// SessionSwitch is the narrow extension interface SwitchPrincipal consumes
// (the concrete *PostgresSessionRepository at
// packages/espyna-golang/contrib/postgres/internal/adapter/entity/
// session_switch_principal.go satisfies it implicitly). Phase 4 (pending) of
// docs/plan/20260524-principal-switch-typed-stack/ wires this by
// type-asserting the session repo inside initServiceAuth.
type Repositories struct {
	Session       sessionpb.SessionDomainServiceServer
	User          userpb.UserDomainServiceServer
	SessionSwitch SessionSwitchAdapter
}

// Services groups infrastructure services. No Authorizer —
// identity-lifecycle use cases run before authz is established.
type Services struct {
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
	// SessionExpiry is the default time-to-live for a newly issued session.
	// Callers typically source this from PASSWORD_AUTH_SESSION_EXPIRY.
	// A zero value means IssueSession falls back to defaultSessionExpiry.
	SessionExpiry SessionExpiryConfig
}

// NewUseCases wires every auth use case from shared dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		AuthenticateSession: NewAuthenticateSessionUseCase(
			AuthenticateSessionRepositories{Session: repositories.Session, User: repositories.User},
			AuthenticateSessionServices{Translator: services.Translator},
		),
		IssueSession: NewIssueSessionUseCase(
			IssueSessionRepositories{Session: repositories.Session},
			IssueSessionServices{
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
				Expiry:      services.SessionExpiry,
			},
		),
		InvalidateSession: NewInvalidateSessionUseCase(
			InvalidateSessionRepositories{Session: repositories.Session},
			InvalidateSessionServices{
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		SwitchPrincipal: NewSwitchPrincipalUseCase(
			SwitchPrincipalRepositories{SessionSwitch: repositories.SessionSwitch},
			SwitchPrincipalServices{Translator: services.Translator},
		),
	}
}
