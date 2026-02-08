package user

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// GetUserListPageDataRepositories groups all repository dependencies
type GetUserListPageDataRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// GetUserListPageDataServices groups all business service dependencies
type GetUserListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetUserListPageDataUseCase handles the business logic for getting user list page data
type GetUserListPageDataUseCase struct {
	repositories GetUserListPageDataRepositories
	services     GetUserListPageDataServices
}

// NewGetUserListPageDataUseCase creates use case with grouped dependencies
func NewGetUserListPageDataUseCase(
	repositories GetUserListPageDataRepositories,
	services GetUserListPageDataServices,
) *GetUserListPageDataUseCase {
	return &GetUserListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetUserListPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetUserListPageDataUseCase with grouped parameters instead
func NewGetUserListPageDataUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *GetUserListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetUserListPageDataRepositories{
		User: userRepo,
	}

	services := GetUserListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetUserListPageDataUseCase(repositories, services)
}

func (uc *GetUserListPageDataUseCase) Execute(ctx context.Context, req *userpb.GetUserListPageDataRequest) (*userpb.GetUserListPageDataResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.request_required", "Request is required for user list page data [DEFAULT]"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 100 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 100 [DEFAULT]"))
		}
		if req.Pagination.Limit < 1 {
			req.Pagination.Limit = 10 // Set default limit
		}
	}

	// Call repository
	resp, err := uc.repositories.User.GetUserListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
