package user

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// ListUsersRepositories groups all repository dependencies
type ListUsersRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// ListUsersServices groups all business service dependencies
type ListUsersServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListUsersUseCase handles the business logic for listing users
type ListUsersUseCase struct {
	repositories ListUsersRepositories
	services     ListUsersServices
}

// NewListUsersUseCase creates use case with grouped dependencies
func NewListUsersUseCase(
	repositories ListUsersRepositories,
	services ListUsersServices,
) *ListUsersUseCase {
	return &ListUsersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListUsersUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListUsersUseCase with grouped parameters instead
func NewListUsersUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *ListUsersUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListUsersRepositories{
		User: userRepo,
	}

	services := ListUsersServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListUsersUseCase(repositories, services)
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// Input validation
	if req == nil {
		req = &userpb.ListUsersRequest{}
	}

	// Call repository
	resp, err := uc.repositories.User.ListUsers(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.list_failed", "Failed to retrieve users [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
