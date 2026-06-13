package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// InvalidateSessionRepositories is the write-side of session termination.
type InvalidateSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
}

// InvalidateSessionServices groups infrastructure deps. No Authorizer —
// per the package invariant this operates on the session established
// earlier in the request lifecycle.
type InvalidateSessionServices struct {
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// InvalidateSessionUseCase marks a session inactive (logout semantics).
type InvalidateSessionUseCase struct {
	repositories InvalidateSessionRepositories
	services     InvalidateSessionServices
}

// NewInvalidateSessionUseCase wires the use case.
func NewInvalidateSessionUseCase(
	repositories InvalidateSessionRepositories,
	services InvalidateSessionServices,
) *InvalidateSessionUseCase {
	return &InvalidateSessionUseCase{repositories: repositories, services: services}
}

// Execute terminates the session addressed by the request.
func (uc *InvalidateSessionUseCase) Execute(
	ctx context.Context,
	req *authpb.InvalidateSessionRequest,
) (*authpb.InvalidateSessionResponse, error) {
	if uc.repositories.Session == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable", "Auth service is not available [DEFAULT]"))
	}
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *InvalidateSessionUseCase) executeWithTransaction(
	ctx context.Context,
	req *authpb.InvalidateSessionRequest,
) (*authpb.InvalidateSessionResponse, error) {
	var out *authpb.InvalidateSessionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(
				txCtx, uc.services.Translator,
				"auth.errors.invalidate_session_failed", "Failed to invalidate session [DEFAULT]")
			return fmt.Errorf("%s: %w", translated, err)
		}
		out = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (uc *InvalidateSessionUseCase) executeCore(
	ctx context.Context,
	req *authpb.InvalidateSessionRequest,
) (*authpb.InvalidateSessionResponse, error) {
	sessionID := req.GetSessionId()
	if sessionID == "" {
		// Resolve by token.
		lookup, err := uc.repositories.Session.ReadSession(ctx, &sessionpb.ReadSessionRequest{
			Data: &sessionpb.Session{Token: req.GetToken()},
		})
		if err != nil || lookup == nil || len(lookup.Data) == 0 {
			// Unknown token — treat as already-invalid, not an error.
			return &authpb.InvalidateSessionResponse{Invalidated: false}, nil
		}
		sessionID = lookup.Data[0].Id
		if sessionID == "" {
			return &authpb.InvalidateSessionResponse{Invalidated: false}, nil
		}
	}

	_, err := uc.repositories.Session.UpdateSession(ctx, &sessionpb.UpdateSessionRequest{
		Data: &sessionpb.Session{Id: sessionID, Active: false},
	})
	if err != nil {
		return nil, err
	}
	return &authpb.InvalidateSessionResponse{Invalidated: true}, nil
}

func (uc *InvalidateSessionUseCase) validateInput(ctx context.Context, req *authpb.InvalidateSessionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required", "Session invalidation request is required [DEFAULT]"))
	}
	if req.GetToken() == "" && req.GetSessionId() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.session_identifier_required",
			"Session token or session ID is required [DEFAULT]"))
	}
	return nil
}
