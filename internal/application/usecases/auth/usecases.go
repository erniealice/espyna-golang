// Package auth contains use cases that establish or terminate user identity —
// cookie/session authentication, session issuance, and session invalidation.
// Future home for login, register, and password-reset use cases currently
// embedded in the `password` auth adapter.
//
// Invariant: every file under usecases/auth/ must either establish identity
// (login, authenticate_session, issue_session, register, request_password_reset,
// execute_password_reset) or terminate an established session
// (invalidate_session, rotate_session). This is why the authcheck coverage
// test skips this directory — these use cases run BEFORE authorization can
// be applied or AFTER it has been revoked. Authenticated business operations
// that are merely auth-adjacent (e.g. "admin revokes another user's
// sessions") belong in usecases/entity/session/ with authcheck wired in.
package auth

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// UseCases aggregates every identity-lifecycle use case in this package.
type UseCases struct {
	AuthenticateSession *AuthenticateSessionUseCase
	IssueSession        *IssueSessionUseCase
	InvalidateSession   *InvalidateSessionUseCase
}

// Repositories groups proto-level domain services needed by auth flows.
type Repositories struct {
	Session sessionpb.SessionDomainServiceServer
	User    userpb.UserDomainServiceServer
}

// Services groups infrastructure services. Note: no Authorizer —
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
	}
}
