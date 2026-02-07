package group

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// UpdateGroupRepositories groups all repository dependencies
type UpdateGroupRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// UpdateGroupServices groups all business service dependencies
type UpdateGroupServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateGroupUseCase handles the business logic for updating groups
type UpdateGroupUseCase struct {
	repositories UpdateGroupRepositories
	services     UpdateGroupServices
}

// NewUpdateGroupUseCase creates use case with grouped dependencies
func NewUpdateGroupUseCase(
	repositories UpdateGroupRepositories,
	services UpdateGroupServices,
) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateGroupUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateGroupUseCase with grouped parameters instead
func NewUpdateGroupUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *UpdateGroupUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateGroupRepositories{
		Group: groupRepo,
	}

	services := UpdateGroupServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateGroupUseCase(repositories, services)
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichGroupData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Group.UpdateGroup(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.update_failed", "Group update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateGroupUseCase) validateInput(ctx context.Context, req *grouppb.UpdateGroupRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.data_required", "Group data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.id_required", "Group ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.name_required", "Group name is required [DEFAULT]"))
	}
	return nil
}

// enrichGroupData adds audit information for updates
func (uc *UpdateGroupUseCase) enrichGroupData(group *grouppb.Group) error {
	now := time.Now()

	// Set group audit fields for modification
	group.DateModified = &[]int64{now.UnixMilli()}[0]
	group.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateGroupUseCase) validateBusinessRules(ctx context.Context, group *grouppb.Group) error {
	// Validate name length
	if len(group.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.name_too_short", "Group name must be at least 2 characters long [DEFAULT]"))
	}

	if len(group.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.name_too_long", "Group name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if group.Description != "" && len(group.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.description_too_long", "Group description cannot exceed 500 characters [DEFAULT]"))
	}

	return nil
}
