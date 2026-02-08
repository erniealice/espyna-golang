package staff

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// GetStaffItemPageDataRepositories groups all repository dependencies
type GetStaffItemPageDataRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// GetStaffItemPageDataServices groups all business service dependencies
type GetStaffItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetStaffItemPageDataUseCase handles the business logic for getting staff item page data
type GetStaffItemPageDataUseCase struct {
	repositories GetStaffItemPageDataRepositories
	services     GetStaffItemPageDataServices
}

// NewGetStaffItemPageDataUseCase creates use case with grouped dependencies
func NewGetStaffItemPageDataUseCase(
	repositories GetStaffItemPageDataRepositories,
	services GetStaffItemPageDataServices,
) *GetStaffItemPageDataUseCase {
	return &GetStaffItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetStaffItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetStaffItemPageDataUseCase with grouped parameters instead
func NewGetStaffItemPageDataUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *GetStaffItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetStaffItemPageDataRepositories{
		Staff: staffRepo,
	}

	services := GetStaffItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetStaffItemPageDataUseCase(repositories, services)
}

func (uc *GetStaffItemPageDataUseCase) Execute(ctx context.Context, req *staffpb.GetStaffItemPageDataRequest) (*staffpb.GetStaffItemPageDataResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.request_required", "Request is required for staff item page data [DEFAULT]"))
	}

	if req.StaffId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.id_required", "Staff ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Staff.GetStaffItemPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	// Check if staff was found
	if resp.Staff == nil || resp.Staff.Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.not_found", "Staff with ID \"{staffId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{staffId}", req.StaffId)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
