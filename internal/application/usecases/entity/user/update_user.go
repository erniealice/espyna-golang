package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// UpdateUserRepositories groups all repository dependencies
type UpdateUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// UpdateUserServices groups all business service dependencies
type UpdateUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateUserUseCase(repositories, services)
}

// Execute performs the update user operation
func (uc *UpdateUserUseCase) Execute(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.request_required", "Request is required for users [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.email_required", "User email is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.User.UpdateUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.update_failed", "User update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
