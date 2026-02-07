package staff

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

// DeleteStaffRepositories groups all repository dependencies
type DeleteStaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// DeleteStaffServices groups all business service dependencies
type DeleteStaffServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteStaffUseCase handles the business logic for deleting a staff
type DeleteStaffUseCase struct {
	repositories DeleteStaffRepositories
	services     DeleteStaffServices
}

// NewDeleteStaffUseCase creates use case with grouped dependencies
func NewDeleteStaffUseCase(
	repositories DeleteStaffRepositories,
	services DeleteStaffServices,
) *DeleteStaffUseCase {
	return &DeleteStaffUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteStaffUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteStaffUseCase with grouped parameters instead
func NewDeleteStaffUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *DeleteStaffUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteStaffRepositories{
		Staff: staffRepo,
	}

	services := DeleteStaffServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteStaffUseCase(repositories, services)
}

func (uc *DeleteStaffUseCase) Execute(ctx context.Context, req *staffpb.DeleteStaffRequest) (*staffpb.DeleteStaffResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityStaff, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.request_required", "Request is required for staff [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.id_required", "Staff ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Staff.DeleteStaff(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
