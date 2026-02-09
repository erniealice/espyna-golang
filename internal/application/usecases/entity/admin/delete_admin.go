package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// DeleteAdminRepositories groups all repository dependencies
type DeleteAdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// DeleteAdminServices groups all business service dependencies
type DeleteAdminServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteAdminUseCase handles the business logic for deleting an admin
type DeleteAdminUseCase struct {
	repositories DeleteAdminRepositories
	services     DeleteAdminServices
}

// NewDeleteAdminUseCase creates use case with grouped dependencies
func NewDeleteAdminUseCase(
	repositories DeleteAdminRepositories,
	services DeleteAdminServices,
) *DeleteAdminUseCase {
	return &DeleteAdminUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete admin operation
func (uc *DeleteAdminUseCase) Execute(ctx context.Context, req *adminpb.DeleteAdminRequest) (*adminpb.DeleteAdminResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityAdmin, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.request_required", ""))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.id_required", ""))
	}

	// Call repository
	resp, err := uc.repositories.Admin.DeleteAdmin(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.deletion_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
