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

// UpdateUserRepositories groups all repository dependencies
type UpdateUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// UpdateUserServices groups all business service dependencies
type UpdateUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// AuthService syncs an email change to the IdP (firebase: UpdateUser{Email},
	// which prevents lockout/account-takeover; password/mock no-op since the DB
	// is authoritative). May be nil; the email sync is then skipped.
	AuthService infraports.AuthService
}

// UpdateUserUseCase handles the business logic for updating a user
type UpdateUserUseCase struct {
	repositories UpdateUserRepositories
	services     UpdateUserServices
}

// NewUpdateUserUseCase creates use case with grouped dependencies
func NewUpdateUserUseCase(
	repositories UpdateUserRepositories,
	services UpdateUserServices,
) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateUserUseCase with grouped parameters instead
func NewUpdateUserUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *UpdateUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateUserRepositories{
		User: userRepo,
	}

	services := UpdateUserServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUpdateUserUseCase(repositories, services)
}

// Execute performs the update user operation
func (uc *UpdateUserUseCase) Execute(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.User,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.request_required", "Request is required for users [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.email_required", "User email is required [DEFAULT]"))
	}

	// Read the current row first and abort if missing, so that sensitive fields
	// come only from trusted data and email-sync decisions reflect real state.
	existing, readErr := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: req.Data.Id},
	})
	if readErr != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.update_failed", "User update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, readErr)
	}

	if existing == nil || len(existing.GetData()) == 0 || existing.GetData()[0] == nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	current := existing.GetData()[0]
	if current.GetId() == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	updateReq := proto.Clone(req).(*userpb.UpdateUserRequest)
	updateData := updateReq.GetData()

	updateData.PasswordHash = current.GetPasswordHash()
	updateData.PasswordResetToken = current.PasswordResetToken
	updateData.PasswordResetExpires = current.PasswordResetExpires
	updateData.FailedLoginAttempts = current.FailedLoginAttempts
	updateData.LockedUntil = current.LockedUntil

	// Detect an email change so we can sync it to the provider (IdP) after DB write.
	emailChanged := current.GetEmailAddress() != updateData.GetEmailAddress()

	// Call repository
	resp, err := uc.repositories.User.UpdateUser(ctx, updateReq)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.update_failed", "User update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Provider-side effect: keep the IdP email in sync with the DB.
	if emailChanged && uc.services.AuthService != nil {
		if syncErr := uc.services.AuthService.UpdateEmailAtProvider(ctx, req.Data.Id, req.Data.EmailAddress); syncErr != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.update_failed", "User update failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, syncErr)
		}
	}

	resp.Data = redactPublicUserResponseData(resp.Data)
	return resp, nil
}
