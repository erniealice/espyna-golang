package staff

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// ReadStaffRepositories groups all repository dependencies
type ReadStaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// ReadStaffServices groups all business service dependencies
type ReadStaffServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadStaffUseCase handles the business logic for reading a staff
type ReadStaffUseCase struct {
	repositories ReadStaffRepositories
	services     ReadStaffServices
}

// NewReadStaffUseCase creates use case with grouped dependencies
func NewReadStaffUseCase(
	repositories ReadStaffRepositories,
	services ReadStaffServices,
) *ReadStaffUseCase {
	return &ReadStaffUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadStaffUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadStaffUseCase with grouped parameters instead
func NewReadStaffUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *ReadStaffUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadStaffRepositories{
		Staff: staffRepo,
	}

	services := ReadStaffServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadStaffUseCase(repositories, services)
}

func (uc *ReadStaffUseCase) Execute(ctx context.Context, req *staffpb.ReadStaffRequest) (*staffpb.ReadStaffResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityStaff, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.request_required", "Request is required for staff [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.id_required", "Staff ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Staff.ReadStaff(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.not_found", "Staff with ID \"{staffId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{staffId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
