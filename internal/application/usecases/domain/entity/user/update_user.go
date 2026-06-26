package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// UpdateUserRepositories groups all repository dependencies
type UpdateUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// UpdateUserServices groups all business service dependencies
type UpdateUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
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

	// Read the current row first, ALWAYS, for two reasons:
	//  1. PRESERVE the password hash — a field update that carries an empty
	//     PasswordHash must NOT blank the credential. The edit form (and most
	//     callers) send no password on a plain edit, so without this an innocuous
	//     "change the name" edit would wipe the user's password (login lockout).
	//     Never blank a credential via a field update.
	//  2. Detect an email change so we can sync it to the IdP (firebase) after the
	//     DB write, to avoid login lockout / account takeover.
	emailChanged := false
	existing, readErr := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: req.Data.Id},
	})
	if readErr == nil && existing != nil && len(existing.GetData()) > 0 {
		cur := existing.GetData()[0]
		if req.Data.GetPasswordHash() == "" && cur.GetPasswordHash() != "" {
			req.Data.PasswordHash = cur.GetPasswordHash()
		}
		if uc.services.AuthService != nil && cur.GetEmailAddress() != req.Data.EmailAddress {
			emailChanged = true
		}
	}

	// Call repository
	resp, err := uc.repositories.User.UpdateUser(ctx, req)
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

	return resp, nil
}
