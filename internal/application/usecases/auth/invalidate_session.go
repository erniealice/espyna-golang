package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

// InvalidateSessionRequest identifies the session to terminate. Exactly one of
// Token or SessionID must be populated.
type InvalidateSessionRequest struct {
	Token     string
	SessionID string
}

// InvalidateSessionResponse reports whether a session was actually terminated.
// Invalidating an unknown/expired token is a no-op (not an error).
type InvalidateSessionResponse struct {
	Invalidated bool
}

// InvalidateSessionRepositories is the write-side of session termination.
type InvalidateSessionRepositories struct {
	Session sessionpb.SessionDomainServiceServer
}

// InvalidateSessionServices groups infrastructure deps. No AuthorizationService —
// per the usecases/auth invariant, this operates on the session established
// earlier in the request lifecycle.
type InvalidateSessionServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
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
	req *InvalidateSessionRequest,
) (*InvalidateSessionResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *InvalidateSessionUseCase) executeWithTransaction(
	ctx context.Context,
	req *InvalidateSessionRequest,
) (*InvalidateSessionResponse, error) {
	var out *InvalidateSessionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(
				txCtx, uc.services.TranslationService,
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
	req *InvalidateSessionRequest,
) (*InvalidateSessionResponse, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		// Resolve by token.
		lookup, err := uc.repositories.Session.ReadSession(ctx, &sessionpb.ReadSessionRequest{
			Data: &sessionpb.Session{Token: req.Token},
		})
		if err != nil || lookup == nil || len(lookup.Data) == 0 {
			// Unknown token — treat as already-invalid, not an error.
			return &InvalidateSessionResponse{Invalidated: false}, nil
		}
		sessionID = lookup.Data[0].Id
		if sessionID == "" {
			return &InvalidateSessionResponse{Invalidated: false}, nil
		}
	}

	_, err := uc.repositories.Session.UpdateSession(ctx, &sessionpb.UpdateSessionRequest{
		Data: &sessionpb.Session{Id: sessionID, Active: false},
	})
	if err != nil {
		return nil, err
	}
	return &InvalidateSessionResponse{Invalidated: true}, nil
}

func (uc *InvalidateSessionUseCase) validateInput(ctx context.Context, req *InvalidateSessionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.validation.request_required", "Session invalidation request is required [DEFAULT]"))
	}
	if req.Token == "" && req.SessionID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"auth.validation.session_identifier_required",
			"Session token or session ID is required [DEFAULT]"))
	}
	return nil
}
