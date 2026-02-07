package staff_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	staffattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff_attribute"
)

// GetStaffAttributeItemPageDataUseCase handles the business logic for getting staff attribute item page data
// GetStaffAttributeItemPageDataRepositories groups all repository dependencies
type GetStaffAttributeItemPageDataRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
}

// GetStaffAttributeItemPageDataServices groups all business service dependencies
type GetStaffAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetStaffAttributeItemPageDataUseCase handles the business logic for getting staff attribute item page data
type GetStaffAttributeItemPageDataUseCase struct {
	repositories GetStaffAttributeItemPageDataRepositories
	services     GetStaffAttributeItemPageDataServices
}

// NewGetStaffAttributeItemPageDataUseCase creates use case with grouped dependencies
func NewGetStaffAttributeItemPageDataUseCase(
	repositories GetStaffAttributeItemPageDataRepositories,
	services GetStaffAttributeItemPageDataServices,
) *GetStaffAttributeItemPageDataUseCase {
	return &GetStaffAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetStaffAttributeItemPageDataUseCaseUngrouped creates a new GetStaffAttributeItemPageDataUseCase
// Deprecated: Use NewGetStaffAttributeItemPageDataUseCase with grouped parameters instead
func NewGetStaffAttributeItemPageDataUseCaseUngrouped(staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer) *GetStaffAttributeItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetStaffAttributeItemPageDataRepositories{
		StaffAttribute: staffAttributeRepo,
	}

	services := GetStaffAttributeItemPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetStaffAttributeItemPageDataUseCase(repositories, services)
}

// Execute performs the get staff attribute item page data operation
func (uc *GetStaffAttributeItemPageDataUseCase) Execute(ctx context.Context, req *staffattributepb.GetStaffAttributeItemPageDataRequest) (*staffattributepb.GetStaffAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.GetStaffAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.item_page_data_failed", "Failed to retrieve staff attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetStaffAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *staffattributepb.GetStaffAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", "Request is required for staff attributes [DEFAULT]"))
	}

	// Validate staff attribute ID
	if strings.TrimSpace(req.StaffAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.id_required", "Staff attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.StaffAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.id_too_short", "Staff attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
