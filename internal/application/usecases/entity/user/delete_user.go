package user

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// DeleteUserRepositories groups all repository dependencies
type DeleteUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// DeleteUserServices groups all business service dependencies
type DeleteUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteUserUseCase handles the business logic for deleting a user
type DeleteUserUseCase struct {
	repositories DeleteUserRepositories
	services     DeleteUserServices
}

// NewDeleteUserUseCase creates use case with grouped dependencies
func NewDeleteUserUseCase(
	repositories DeleteUserRepositories,
	services DeleteUserServices,
) *DeleteUserUseCase {
	return &DeleteUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteUserUseCase with grouped parameters instead
func NewDeleteUserUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *DeleteUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteUserRepositories{
		User: userRepo,
	}

	services := DeleteUserServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteUserUseCase(repositories, services)
}

func (uc *DeleteUserUseCase) Execute(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.request_required", "Request is required for users [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.User.DeleteUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.deletion_failed", "User deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
