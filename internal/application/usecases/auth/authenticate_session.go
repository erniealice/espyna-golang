package auth

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// AuthenticateSessionRequest carries the opaque session token presented by a caller.
type AuthenticateSessionRequest struct {
	Token string
}

// Identity is the authenticated principal materialised from a session + user lookup.
type Identity struct {
	UserID          string
	Email           string
	WorkspaceUserID string
	WorkspaceID     string
	Token           string
	ExpiresAtUnixMs int64
}

// AuthenticateSessionResponse wraps the materialised Identity plus flow metadata.
type AuthenticateSessionResponse struct {
	Identity Identity
}

// AuthenticateSessionRepositories groups proto repositories consulted during authentication.
type AuthenticateSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
	User    userpb.UserDomainServiceServer
}

// AuthenticateSessionServices groups application services. No AuthorizationService —
// this use case establishes identity; permission checks cannot run before it.
type AuthenticateSessionServices struct {
	TranslationService ports.TranslationService
}

// AuthenticateSessionUseCase resolves an opaque session token into an Identity.
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
	req *AuthenticateSessionRequest,
) (*AuthenticateSessionResponse, error) {
	if req == nil || req.Token == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.missing_token", "Session token is required [DEFAULT]"))
	}

	sessResp, err := uc.repositories.Session.ReadSession(ctx, &sessionpb.ReadSessionRequest{
		Data: &sessionpb.Session{Token: req.Token},
	})
	if err != nil || sessResp == nil || len(sessResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.session_invalid", "Invalid or expired session [DEFAULT]"))
	}
	sess := sessResp.Data[0]

	if !sess.Active {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.session_inactive", "Session has been invalidated [DEFAULT]"))
	}
	if sess.ExpiresAt > 0 && sess.ExpiresAt <= time.Now().UnixMilli() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.session_expired", "Session has expired [DEFAULT]"))
	}

	userResp, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: sess.UserId},
	})
	if err != nil || userResp == nil || len(userResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.session_user_missing", "Authenticated user not found [DEFAULT]"))
	}
	user := userResp.Data[0]

	identity := Identity{
		UserID:          sess.UserId,
		Email:           user.EmailAddress,
		Token:           sess.Token,
		ExpiresAtUnixMs: sess.ExpiresAt,
	}
	if sess.WorkspaceUserId != nil {
		identity.WorkspaceUserID = *sess.WorkspaceUserId
	}
	if sess.WorkspaceId != nil {
		identity.WorkspaceID = *sess.WorkspaceId
	}

	return &AuthenticateSessionResponse{Identity: identity}, nil
}
