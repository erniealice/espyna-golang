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
// service_unavailable when Session is nil.
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
	if uc.repositories.Session == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}
	if req == nil || req.GetToken() == "" {
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

	if uc.repositories.User == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
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
