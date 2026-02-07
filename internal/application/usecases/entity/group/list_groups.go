package group

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// ListGroupsRepositories groups all repository dependencies
type ListGroupsRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// ListGroupsServices groups all business service dependencies
type ListGroupsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListGroupsUseCase handles the business logic for listing groups
type ListGroupsUseCase struct {
	repositories ListGroupsRepositories
	services     ListGroupsServices
}

// NewListGroupsUseCase creates use case with grouped dependencies
func NewListGroupsUseCase(
	repositories ListGroupsRepositories,
	services ListGroupsServices,
) *ListGroupsUseCase {
	return &ListGroupsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListGroupsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListGroupsUseCase with grouped parameters instead
func NewListGroupsUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *ListGroupsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListGroupsRepositories{
		Group: groupRepo,
	}

	services := ListGroupsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListGroupsUseCase(repositories, services)
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context, req *grouppb.ListGroupsRequest) (*grouppb.ListGroupsResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Group.ListGroups(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.list_failed", "Failed to retrieve groups [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListGroupsUseCase) validateInput(ctx context.Context, req *grouppb.ListGroupsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListGroupsUseCase) validateBusinessRules(ctx context.Context, req *grouppb.ListGroupsRequest) error {
	// No additional business rules for listing groups
	// Pagination is not supported in current protobuf definition
	return nil
}
