package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	"google.golang.org/protobuf/proto"
)

// EnableUserRepositories groups all repository dependencies.
type EnableUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// EnableUserServices groups all business service dependencies. AuthService is
// the inward port that re-enables the account at the IdP (firebase:
// UpdateUser{Disabled:false}; password/mock no-op). It may be nil.
type EnableUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	AuthService      infraports.AuthService
}

// EnableUserUseCase sets user.active=true and re-enables the IdP account.
type EnableUserUseCase struct {
	repositories EnableUserRepositories
	services     EnableUserServices
}

// NewEnableUserUseCase creates the use case with grouped dependencies.
func NewEnableUserUseCase(
	repositories EnableUserRepositories,
	services EnableUserServices,
) *EnableUserUseCase {
	return &EnableUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute re-enables the user account.
func (uc *EnableUserUseCase) Execute(ctx context.Context, req *userpb.EnableUserRequest) (*userpb.EnableUserResponse, error) {
	// Authorization check — user:enable.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.User,
		Action: entityid.ActionEnable,
	}); err != nil {
		return nil, err
	}

	// Input validation.
	if req == nil || req.GetUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Read existing user and preserve trusted profile fields.
	existing, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: req.GetUserId()},
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.enable_failed", "User enable failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	if existing == nil || len(existing.GetData()) == 0 || existing.GetData()[0] == nil || existing.GetData()[0].GetId() == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.GetUserId())
		return nil, errors.New(translatedError)
	}

	updateData := proto.Clone(existing.GetData()[0]).(*userpb.User)
	updateData.Active = true

	// Set the global active flag back to true.
	if _, err := uc.repositories.User.UpdateUser(ctx, &userpb.UpdateUserRequest{
		Data: updateData,
	}); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.enable_failed", "User enable failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Provider-side effect: re-enable at the IdP.
	if uc.services.AuthService != nil {
		if err := uc.services.AuthService.EnableUserAtProvider(ctx, req.GetUserId()); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.enable_failed", "User enable failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}
	}

	return &userpb.EnableUserResponse{Enabled: true, Success: true}, nil
}
