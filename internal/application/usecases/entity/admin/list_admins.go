package admin

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

// ListAdminsRepositories groups all repository dependencies
type ListAdminsRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// ListAdminsServices groups all business service dependencies
type ListAdminsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAdminsUseCase handles the business logic for listing admins
type ListAdminsUseCase struct {
	repositories ListAdminsRepositories
	services     ListAdminsServices
}

// NewListAdminsUseCase creates use case with grouped dependencies
func NewListAdminsUseCase(
	repositories ListAdminsRepositories,
	services ListAdminsServices,
) *ListAdminsUseCase {
	return &ListAdminsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list admins operation
func (uc *ListAdminsUseCase) Execute(ctx context.Context, req *adminpb.ListAdminsRequest) (*adminpb.ListAdminsResponse, error) {
	// businessType := contextutil.ExtractBusinessTypeFromContext(ctx)

	// Input validation
	if req == nil {
		req = &adminpb.ListAdminsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Admin.ListAdmins(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.list_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
