package auth

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// AuthenticateSessionRepositories groups proto repositories consulted during
// authentication. Both are nil-tolerant — Execute fails closed with
// service_unavailable when either Session or User is nil (combined body-entry
// guard, codex auth-collapse R1 P1-1).
type AuthenticateSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
	User    userpb.UserDomainServiceServer
}

// AuthenticateSessionServices groups application services. No Authorizer —
// this use case establishes identity; per-action authorization cannot run
// before it.
type AuthenticateSessionServices struct {
	Translator ports.Translator
}

// AuthenticateSessionUseCase resolves an opaque session token into the
// authenticated principal, returned as a proto AuthIdentity.
type AuthenticateSessionUseCase struct {
	repositories AuthenticateSessionRepositories
	services     AuthenticateSessionServices
}

// NewAuthenticateSessionUseCase wires the use case from grouped dependencies.
func NewAuthenticateSessionUseCase(
	repositories AuthenticateSessionRepositories,
	services AuthenticateSessionServices,
) *AuthenticateSessionUseCase {
	return &AuthenticateSessionUseCase{repositories: repositories, services: services}
}

// Execute validates the session token, enforces expiry, and hydrates the
// identity from the user record. Read-only; never wrapped in a transaction.
func (uc *AuthenticateSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.AuthenticateSessionRequest,
) (*authpb.AuthenticateSessionResponse, error) {
	// Fail-closed at body entry when either repo is missing (Q2 lock).
	// Placement matters: this MUST run before any repository call so a
	// partially-wired auth subsystem cannot return session-specific errors
	// while the user-side is unwired. Codex round 1 P1-1.
	if uc.repositories.Session == nil || uc.repositories.User == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}
	// Split nil-request from empty-token to preserve the pre-collapse public
	// translator-key surface (codex round 1 P1-2): nil request returns
	// request_required; well-formed request with empty token returns missing_token.
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required", "Session authentication request is required [DEFAULT]"))
	}
	if req.GetToken() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.missing_token", "Session token is required [DEFAULT]"))
	}

	sessResp, err := uc.repositories.Session.ReadSession(ctx, &sessionpb.ReadSessionRequest{
		Data: &sessionpb.Session{Token: req.GetToken()},
	})
	if err != nil || sessResp == nil || len(sessResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.session_invalid", "Invalid or expired session [DEFAULT]"))
	}
	sess := sessResp.Data[0]

	if !sess.Active {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.session_inactive", "Session has been invalidated [DEFAULT]"))
	}
	if sess.ExpiresAt > 0 && sess.ExpiresAt <= time.Now().UnixMilli() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.session_expired", "Session has expired [DEFAULT]"))
	}

	userResp, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: sess.UserId},
	})
	if err != nil || userResp == nil || len(userResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.session_user_missing", "Authenticated user not found [DEFAULT]"))
	}
	user := userResp.Data[0]

	identity := &authpb.AuthIdentity{
		UserId:          sess.UserId,
		Email:           user.EmailAddress,
		Token:           sess.Token,
		ExpiresAtUnixMs: sess.ExpiresAt,
	}
	if sess.WorkspaceUserId != nil {
		identity.WorkspaceUserId = *sess.WorkspaceUserId
	}
	if sess.WorkspaceId != nil {
		identity.WorkspaceId = *sess.WorkspaceId
	}

	return &authpb.AuthenticateSessionResponse{Identity: identity}, nil
}
