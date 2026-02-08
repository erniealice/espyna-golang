package staff

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// UpdateStaffRepositories groups all repository dependencies
type UpdateStaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// UpdateStaffServices groups all business service dependencies
type UpdateStaffServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateStaffUseCase handles the business logic for updating a staff
type UpdateStaffUseCase struct {
	repositories UpdateStaffRepositories
	services     UpdateStaffServices
}

// NewUpdateStaffUseCase creates use case with grouped dependencies
func NewUpdateStaffUseCase(
	repositories UpdateStaffRepositories,
	services UpdateStaffServices,
) *UpdateStaffUseCase {
	return &UpdateStaffUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateStaffUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateStaffUseCase with grouped parameters instead
func NewUpdateStaffUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *UpdateStaffUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateStaffRepositories{
		Staff: staffRepo,
	}

	services := UpdateStaffServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateStaffUseCase(repositories, services)
}

func (uc *UpdateStaffUseCase) Execute(ctx context.Context, req *staffpb.UpdateStaffRequest) (*staffpb.UpdateStaffResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityStaff, ports.ActionUpdate)
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

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.email_required", "Staff email is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Staff.UpdateStaff(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.update_failed", "Staff update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
