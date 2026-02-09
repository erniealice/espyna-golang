package user

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// GetUserItemPageDataRepositories groups all repository dependencies
type GetUserItemPageDataRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// GetUserItemPageDataServices groups all business service dependencies
type GetUserItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetUserItemPageDataUseCase handles the business logic for getting user item page data
type GetUserItemPageDataUseCase struct {
	repositories GetUserItemPageDataRepositories
	services     GetUserItemPageDataServices
}

// NewGetUserItemPageDataUseCase creates use case with grouped dependencies
func NewGetUserItemPageDataUseCase(
	repositories GetUserItemPageDataRepositories,
	services GetUserItemPageDataServices,
) *GetUserItemPageDataUseCase {
	return &GetUserItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetUserItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetUserItemPageDataUseCase with grouped parameters instead
func NewGetUserItemPageDataUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *GetUserItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetUserItemPageDataRepositories{
		User: userRepo,
	}

	services := GetUserItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetUserItemPageDataUseCase(repositories, services)
}

func (uc *GetUserItemPageDataUseCase) Execute(ctx context.Context, req *userpb.GetUserItemPageDataRequest) (*userpb.GetUserItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityUser, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.request_required", "Request is required for user item page data [DEFAULT]"))
	}

	if req.UserId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.User.GetUserItemPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	// Check if user was found
	if resp.User == nil || resp.User.Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.UserId)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
