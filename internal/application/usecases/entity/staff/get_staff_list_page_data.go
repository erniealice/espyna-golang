package staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// GetStaffListPageDataRepositories groups all repository dependencies
type GetStaffListPageDataRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// GetStaffListPageDataServices groups all business service dependencies
type GetStaffListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetStaffListPageDataUseCase handles the business logic for getting staff list page data
type GetStaffListPageDataUseCase struct {
	repositories GetStaffListPageDataRepositories
	services     GetStaffListPageDataServices
}

// NewGetStaffListPageDataUseCase creates use case with grouped dependencies
func NewGetStaffListPageDataUseCase(
	repositories GetStaffListPageDataRepositories,
	services GetStaffListPageDataServices,
) *GetStaffListPageDataUseCase {
	return &GetStaffListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetStaffListPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetStaffListPageDataUseCase with grouped parameters instead
func NewGetStaffListPageDataUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *GetStaffListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetStaffListPageDataRepositories{
		Staff: staffRepo,
	}

	services := GetStaffListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetStaffListPageDataUseCase(repositories, services)
}

func (uc *GetStaffListPageDataUseCase) Execute(ctx context.Context, req *staffpb.GetStaffListPageDataRequest) (*staffpb.GetStaffListPageDataResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.request_required", "Request is required for staff list page data [DEFAULT]"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 100 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.pagination_limit_exceeded", "Pagination limit cannot exceed 100 [DEFAULT]"))
		}
		if req.Pagination.Limit < 1 {
			req.Pagination.Limit = 10 // Set default limit
		}
	}

	// Call repository
	resp, err := uc.repositories.Staff.GetStaffListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
