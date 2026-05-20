package auth

import (
	"context"
	"errors"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// IssueSessionUseCase adapts the entity-layer
// usecases/auth.IssueSessionUseCase to the proto-shaped service/auth
// surface. See authenticate_session.go for the wrapper pattern.
type IssueSessionUseCase struct {
	inner    *entityauth.IssueSessionUseCase
	services Services
}

// NewIssueSessionUseCase wires the wrapper.
func NewIssueSessionUseCase(
	inner *entityauth.IssueSessionUseCase,
	services Services,
) *IssueSessionUseCase {
	return &IssueSessionUseCase{inner: inner, services: services}
}

// Execute mints a new session via the entity-layer use case and returns the
// resulting token + metadata as a proto Response.
func (uc *IssueSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.IssueSessionRequest,
) (*authpb.IssueSessionResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.validation.request_required", "Session issuance request is required [DEFAULT]"))
	}
	if uc.inner == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}

	resp, err := uc.inner.Execute(ctx, &entityauth.IssueSessionRequest{
		UserID:          req.GetUserId(),
		WorkspaceUserID: req.GetWorkspaceUserId(),
		WorkspaceID:     req.GetWorkspaceId(),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.errors.issue_session_failed", "Failed to issue session [DEFAULT]"))
	}

	return &authpb.IssueSessionResponse{
		Token:           resp.Token,
		SessionId:       resp.SessionID,
		ExpiresAtUnixMs: resp.ExpiresAtUnixMs,
		WorkspaceUserId: resp.WorkspaceUserID,
		WorkspaceId:     resp.WorkspaceID,
	}, nil
}
