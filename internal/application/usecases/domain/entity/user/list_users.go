package user

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ListUsersRepositories groups all repository dependencies
type ListUsersRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// ListUsersServices groups all business service dependencies
type ListUsersServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListUsersUseCase(repositories, services)
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.User, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &userpb.ListUsersRequest{}
	}

	// Call repository
	resp, err := uc.repositories.User.ListUsers(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.list_failed", "Failed to retrieve users [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
