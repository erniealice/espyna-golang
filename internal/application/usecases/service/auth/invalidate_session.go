package auth

import (
	"context"
	"errors"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// InvalidateSessionUseCase adapts the entity-layer
// usecases/auth.InvalidateSessionUseCase to the proto-shaped service/auth
// surface. See authenticate_session.go for the wrapper pattern.
type InvalidateSessionUseCase struct {
	inner    *entityauth.InvalidateSessionUseCase
	services Services
}

// NewInvalidateSessionUseCase wires the wrapper.
func NewInvalidateSessionUseCase(
	inner *entityauth.InvalidateSessionUseCase,
	services Services,
) *InvalidateSessionUseCase {
	return &InvalidateSessionUseCase{inner: inner, services: services}
}

// Execute terminates a session via the entity-layer use case and returns
// the boolean outcome as a proto Response.
func (uc *InvalidateSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.InvalidateSessionRequest,
) (*authpb.InvalidateSessionResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.validation.request_required", "Session invalidation request is required [DEFAULT]"))
	}
	if uc.inner == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}

	resp, err := uc.inner.Execute(ctx, &entityauth.InvalidateSessionRequest{
		Token:     req.GetToken(),
		SessionID: req.GetSessionId(),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return &authpb.InvalidateSessionResponse{Invalidated: false}, nil
	}

	return &authpb.InvalidateSessionResponse{Invalidated: resp.Invalidated}, nil
}
