package group_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
)

// GetGroupAttributeItemPageDataUseCase handles the business logic for getting group attribute item page data
// GetGroupAttributeItemPageDataRepositories groups all repository dependencies
type GetGroupAttributeItemPageDataRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
}

// GetGroupAttributeItemPageDataServices groups all business service dependencies
type GetGroupAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetGroupAttributeItemPageDataUseCase handles the business logic for getting group attribute item page data
type GetGroupAttributeItemPageDataUseCase struct {
	repositories GetGroupAttributeItemPageDataRepositories
	services     GetGroupAttributeItemPageDataServices
}

// NewGetGroupAttributeItemPageDataUseCase creates use case with grouped dependencies
func NewGetGroupAttributeItemPageDataUseCase(
	repositories GetGroupAttributeItemPageDataRepositories,
	services GetGroupAttributeItemPageDataServices,
) *GetGroupAttributeItemPageDataUseCase {
	return &GetGroupAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetGroupAttributeItemPageDataUseCaseUngrouped creates a new GetGroupAttributeItemPageDataUseCase
// Deprecated: Use NewGetGroupAttributeItemPageDataUseCase with grouped parameters instead
func NewGetGroupAttributeItemPageDataUseCaseUngrouped(groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer) *GetGroupAttributeItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetGroupAttributeItemPageDataRepositories{
		GroupAttribute: groupAttributeRepo,
	}

	services := GetGroupAttributeItemPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetGroupAttributeItemPageDataUseCase(repositories, services)
}

// Execute performs the get group attribute item page data operation
func (uc *GetGroupAttributeItemPageDataUseCase) Execute(ctx context.Context, req *groupattributepb.GetGroupAttributeItemPageDataRequest) (*groupattributepb.GetGroupAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.GetGroupAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.item_page_data_failed", "Failed to retrieve group attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetGroupAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *groupattributepb.GetGroupAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "Request is required for group attributes [DEFAULT]"))
	}

	// Validate group attribute ID
	if strings.TrimSpace(req.GroupAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.id_required", "Group attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.GroupAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.id_too_short", "Group attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
