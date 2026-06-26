package session

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

// UseCases contains all session-entity use cases. Today only the admin
// bulk-revocation operation lives here; CRUD session management remains in
// service/auth (authentication-time).
type UseCases struct {
	RevokeUserSessions *RevokeUserSessionsUseCase
}

// SessionRepositories groups all repository dependencies for session use cases.
type SessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer // Primary entity repository
}

// SessionServices groups all business service dependencies for session use
// cases. AuthService is the inward IdP port (RevokeUserTokens); it may be nil.
type SessionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	AuthService      infraports.AuthService
}

// NewUseCases creates the collection of session use cases.
func NewUseCases(
	repositories SessionRepositories,
	services SessionServices,
) *UseCases {
	revokeRepos := RevokeUserSessionsRepositories{Session: repositories.Session}
	revokeServices := RevokeUserSessionsServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		AuthService:      services.AuthService,
	}

	return &UseCases{
		RevokeUserSessions: NewRevokeUserSessionsUseCase(revokeRepos, revokeServices),
	}
}
