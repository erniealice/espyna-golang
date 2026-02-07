package group

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// GetGroupItemPageDataUseCase handles the business logic for getting group item page data
// GetGroupItemPageDataRepositories groups all repository dependencies
type GetGroupItemPageDataRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// GetGroupItemPageDataServices groups all business service dependencies
type GetGroupItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetGroupItemPageDataUseCase handles the business logic for getting group item page data
type GetGroupItemPageDataUseCase struct {
	repositories GetGroupItemPageDataRepositories
	services     GetGroupItemPageDataServices
}

// NewGetGroupItemPageDataUseCase creates use case with grouped dependencies
func NewGetGroupItemPageDataUseCase(
	repositories GetGroupItemPageDataRepositories,
	services GetGroupItemPageDataServices,
) *GetGroupItemPageDataUseCase {
	return &GetGroupItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetGroupItemPageDataUseCaseUngrouped creates a new GetGroupItemPageDataUseCase
// Deprecated: Use NewGetGroupItemPageDataUseCase with grouped parameters instead
func NewGetGroupItemPageDataUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *GetGroupItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetGroupItemPageDataRepositories{
		Group: groupRepo,
	}

	services := GetGroupItemPageDataServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewGetGroupItemPageDataUseCase(repositories, services)
}

// Execute performs the get group item page data operation
func (uc *GetGroupItemPageDataUseCase) Execute(ctx context.Context, req *grouppb.GetGroupItemPageDataRequest) (*grouppb.GetGroupItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Group.GetGroupItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.item_page_data_failed", "Failed to retrieve group item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetGroupItemPageDataUseCase) validateInput(ctx context.Context, req *grouppb.GetGroupItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}

	// Validate group ID
	if strings.TrimSpace(req.GroupId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.id_required", "Group ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.GroupId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.id_too_short", "Group ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
