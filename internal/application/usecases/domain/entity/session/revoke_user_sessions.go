package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

// RevokeUserSessionsRepositories is the write-side of bulk session revocation.
type RevokeUserSessionsRepositories struct {
	Session sessionpb.SessionDomainServiceServer // Primary entity repository
}

// RevokeUserSessionsServices groups all business service dependencies.
// AuthService is the inward port that revokes the user's outstanding refresh
// tokens at the IdP (firebase: RevokeRefreshTokens; password/mock no-op — the
// session rows + user.active guard are authoritative). It may be nil.
type RevokeUserSessionsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	AuthService      infraports.AuthService
}

// RevokeUserSessionsUseCase invalidates all active sessions for a user and
// revokes their provider-side tokens. See design §5 (session:revoke).
type RevokeUserSessionsUseCase struct {
	repositories RevokeUserSessionsRepositories
	services     RevokeUserSessionsServices
}

// NewRevokeUserSessionsUseCase creates the use case with grouped dependencies.
func NewRevokeUserSessionsUseCase(
	repositories RevokeUserSessionsRepositories,
	services RevokeUserSessionsServices,
) *RevokeUserSessionsUseCase {
	return &RevokeUserSessionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute revokes every active session for the user.
func (uc *RevokeUserSessionsUseCase) Execute(ctx context.Context, req *sessionpb.RevokeUserSessionsRequest) (*sessionpb.RevokeUserSessionsResponse, error) {
	// Authorization check — session:revoke.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Session,
		Action: entityid.ActionRevoke,
	}); err != nil {
		return nil, err
	}

	// Input validation.
	if req == nil || req.GetUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "session.validation.user_id_required", "User ID is required [DEFAULT]"))
	}

	if uc.repositories.Session == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "session.errors.service_unavailable", "Session service is not available [DEFAULT]"))
	}

	// Invalidate all active session rows for the user (the DB-authoritative
	// effect). Query by user_id + active, then mark each inactive — mirrors the
	// password adapter's InvalidateAllUserSessions but stays provider-agnostic.
	lookup, err := uc.repositories.Session.ReadSession(ctx, &sessionpb.ReadSessionRequest{
		Data: &sessionpb.Session{UserId: req.GetUserId(), Active: true},
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "session.errors.revoke_failed", "Session revocation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	if lookup != nil {
		for _, s := range lookup.GetData() {
			if s == nil || s.GetId() == "" {
				continue
			}
			if _, updErr := uc.repositories.Session.UpdateSession(ctx, &sessionpb.UpdateSessionRequest{
				Data: &sessionpb.Session{Id: s.GetId(), Active: false},
			}); updErr != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "session.errors.revoke_failed", "Session revocation failed [DEFAULT]")
				return nil, fmt.Errorf("%s: %w", translatedError, updErr)
			}
		}
	}

	// Provider-side effect: revoke outstanding refresh tokens at the IdP.
	if uc.services.AuthService != nil {
		if err := uc.services.AuthService.RevokeUserTokens(ctx, req.GetUserId()); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "session.errors.revoke_failed", "Session revocation failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
	}

	return &sessionpb.RevokeUserSessionsResponse{Revoked: true, Success: true}, nil
}
