package staff

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// ListStaffsRepositories groups all repository dependencies
type ListStaffsRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// ListStaffsServices groups all business service dependencies
type ListStaffsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListStaffsUseCase handles the business logic for listing staffs
type ListStaffsUseCase struct {
	repositories ListStaffsRepositories
	services     ListStaffsServices
}

// NewListStaffsUseCase creates use case with grouped dependencies
func NewListStaffsUseCase(
	repositories ListStaffsRepositories,
	services ListStaffsServices,
) *ListStaffsUseCase {
	return &ListStaffsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListStaffsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListStaffsUseCase with grouped parameters instead
func NewListStaffsUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *ListStaffsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListStaffsRepositories{
		Staff: staffRepo,
	}

	services := ListStaffsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListStaffsUseCase(repositories, services)
}

func (uc *ListStaffsUseCase) Execute(ctx context.Context, req *staffpb.ListStaffsRequest) (*staffpb.ListStaffsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityStaff, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &staffpb.ListStaffsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.Staff.ListStaffs(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.list_failed", "Failed to retrieve staff [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
