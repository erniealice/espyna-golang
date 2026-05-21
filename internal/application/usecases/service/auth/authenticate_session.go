package auth

import (
	"context"
	"errors"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// AuthenticateSessionUseCase adapts the entity-layer
// usecases/auth.AuthenticateSessionUseCase to the proto-shaped
// service/auth.AuthenticateSessionRequest/Response surface. It is the
// service-driven (Q7) wrapper consumed by apps that previously reached for
// `consumer.AuthenticateSessionRequest` via the auth_aliases.go visibility
// bridge.
//
// Translation flow: the proto-shaped request is rewritten into the
// entity-layer struct request, the inner use case is invoked, and the
// resulting Identity is rewritten back into the proto-shaped response.
//
// No authcheck.Check call here: this is identity ESTABLISHMENT. Per the
// invariant on usecases/auth/usecases.go the entity-layer use case also
// runs before any per-action authorization is possible.
type AuthenticateSessionUseCase struct {
	inner    *entityauth.AuthenticateSessionUseCase
	services Services
}

// NewAuthenticateSessionUseCase wires the wrapper. inner may be nil; in that
// case Execute returns a translated "service unavailable" error so the
// caller degrades gracefully.
func NewAuthenticateSessionUseCase(
	inner *entityauth.AuthenticateSessionUseCase,
	services Services,
) *AuthenticateSessionUseCase {
	return &AuthenticateSessionUseCase{inner: inner, services: services}
}

// Execute validates the proto-shaped session token and returns the
// authenticated Identity as a proto Response.
func (uc *AuthenticateSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.AuthenticateSessionRequest,
) (*authpb.AuthenticateSessionResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required", "Session authentication request is required [DEFAULT]"))
	}
	if uc.inner == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}

	resp, err := uc.inner.Execute(ctx, &entityauth.AuthenticateSessionRequest{Token: req.GetToken()})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.session_invalid", "Invalid or expired session [DEFAULT]"))
	}

	return &authpb.AuthenticateSessionResponse{
		Identity: &authpb.AuthIdentity{
			UserId:          resp.Identity.UserID,
			Email:           resp.Identity.Email,
			WorkspaceUserId: resp.Identity.WorkspaceUserID,
			WorkspaceId:     resp.Identity.WorkspaceID,
			Token:           resp.Identity.Token,
			ExpiresAtUnixMs: resp.Identity.ExpiresAtUnixMs,
		},
	}, nil
}
