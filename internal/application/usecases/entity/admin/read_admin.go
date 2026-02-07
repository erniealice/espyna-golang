package admin

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

// ReadAdminRepositories groups all repository dependencies
type ReadAdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// ReadAdminServices groups all business service dependencies
type ReadAdminServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadAdminUseCase handles the business logic for reading an admin
type ReadAdminUseCase struct {
	repositories ReadAdminRepositories
	services     ReadAdminServices
}

// NewReadAdminUseCase creates use case with grouped dependencies
func NewReadAdminUseCase(
	repositories ReadAdminRepositories,
	services ReadAdminServices,
) *ReadAdminUseCase {
	return &ReadAdminUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read admin operation
func (uc *ReadAdminUseCase) Execute(ctx context.Context, req *adminpb.ReadAdminRequest) (*adminpb.ReadAdminResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.request_required", ""))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.data_required", ""))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.id_required", ""))
	}

	// Call repository
	resp, err := uc.repositories.Admin.ReadAdmin(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}
