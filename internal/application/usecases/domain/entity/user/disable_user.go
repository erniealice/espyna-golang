package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// DisableUserRepositories groups all repository dependencies.
type DisableUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// DisableUserServices groups all business service dependencies. AuthService is
// the inward port that performs the provider-specific IdP effect (firebase
// disables the account + revokes tokens; password/mock no-op — the DB
// user.active guard is authoritative). It may be nil; the DB write is then
// the only effect.
type DisableUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	AuthService      infraports.AuthService
}

// DisableUserUseCase sets user.active=false (account-wide disable) and syncs
// the IdP via the AuthService port. See design §2 (disable semantics) and §5.
type DisableUserUseCase struct {
	repositories DisableUserRepositories
	services     DisableUserServices
}

// NewDisableUserUseCase creates the use case with grouped dependencies.
func NewDisableUserUseCase(
	repositories DisableUserRepositories,
	services DisableUserServices,
) *DisableUserUseCase {
	return &DisableUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute disables the user account.
func (uc *DisableUserUseCase) Execute(ctx context.Context, req *userpb.DisableUserRequest) (*userpb.DisableUserResponse, error) {
	// Authorization check — user:disable.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.User,
		Action: entityid.ActionDisable,
	}); err != nil {
		return nil, err
	}

	// Input validation.
	if req == nil || req.GetUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Set the global active flag to false — this is the authoritative effect
	// (the per-request user.active guard rejects the user on the next request).
	if _, err := uc.repositories.User.UpdateUser(ctx, &userpb.UpdateUserRequest{
		Data: &userpb.User{Id: req.GetUserId(), Active: false},
	}); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.disable_failed", "User disable failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Provider-side effect: disable at the IdP and revoke outstanding tokens.
	if uc.services.AuthService != nil {
		if err := uc.services.AuthService.DisableUserAtProvider(ctx, req.GetUserId()); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.disable_failed", "User disable failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
		if err := uc.services.AuthService.RevokeUserTokens(ctx, req.GetUserId()); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.disable_failed", "User disable failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
	}

	return &userpb.DisableUserResponse{Disabled: true, Success: true}, nil
}
