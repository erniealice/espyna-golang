package group

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// CreateGroupRepositories groups all repository dependencies
type CreateGroupRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// CreateGroupServices groups all business service dependencies
type CreateGroupServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateGroupUseCase handles the business logic for creating groups
type CreateGroupUseCase struct {
	repositories CreateGroupRepositories
	services     CreateGroupServices
}

// NewCreateGroupUseCase creates use case with grouped dependencies
func NewCreateGroupUseCase(
	repositories CreateGroupRepositories,
	services CreateGroupServices,
) *CreateGroupUseCase {
	return &CreateGroupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateGroupUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateGroupUseCase with grouped parameters instead
func NewCreateGroupUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *CreateGroupUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateGroupRepositories{
		Group: groupRepo,
	}

	services := CreateGroupServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateGroupUseCase(repositories, services)
}

// Execute performs the create group operation
func (uc *CreateGroupUseCase) Execute(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes group creation within a transaction
func (uc *CreateGroupUseCase) executeWithTransaction(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	var result *grouppb.CreateGroupResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "group.errors.creation_failed", "Group creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateGroupUseCase) executeCore(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichGroupData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Group.CreateGroup(ctx, req)
}

// validateInput validates the input request
func (uc *CreateGroupUseCase) validateInput(ctx context.Context, req *grouppb.CreateGroupRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.data_required", "Group data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.name_required", "Group name is required [DEFAULT]"))
	}
	return nil
}

// enrichGroupData adds generated fields and audit information
func (uc *CreateGroupUseCase) enrichGroupData(group *grouppb.Group) error {
	now := time.Now()

	// Generate Group ID if not provided
	if group.Id == "" {
		group.Id = uc.services.IDService.GenerateID()
	}

	// Set group audit fields
	group.DateCreated = &[]int64{now.UnixMilli()}[0]
	group.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	group.DateModified = &[]int64{now.UnixMilli()}[0]
	group.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	group.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateGroupUseCase) validateBusinessRules(ctx context.Context, group *grouppb.Group) error {
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
