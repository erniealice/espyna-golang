package workspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// UpdateWorkspaceRepositories groups all repository dependencies
type UpdateWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// UpdateWorkspaceServices groups all business service dependencies
type UpdateWorkspaceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateWorkspaceUseCase handles the business logic for updating a workspace
type UpdateWorkspaceUseCase struct {
	repositories UpdateWorkspaceRepositories
	services     UpdateWorkspaceServices
}

// NewUpdateWorkspaceUseCase creates use case with grouped dependencies
func NewUpdateWorkspaceUseCase(
	repositories UpdateWorkspaceRepositories,
	services UpdateWorkspaceServices,
) *UpdateWorkspaceUseCase {
	return &UpdateWorkspaceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateWorkspaceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateWorkspaceUseCase with grouped parameters instead
func NewUpdateWorkspaceUseCaseUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *UpdateWorkspaceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateWorkspaceRepositories{
		Workspace: workspaceRepo,
	}

	services := UpdateWorkspaceServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateWorkspaceUseCase(repositories, services)
}

// Execute performs the update workspace operation
func (uc *UpdateWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspace, ports.ActionUpdate); err != nil {
		return nil, err
	}

		if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.id_required", "Workspace ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_required", "Workspace name is required [DEFAULT]"))
	}

	// Validate business rules (including name length)
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Enrich workspace data with audit information
	if err := uc.enrichWorkspaceData(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Workspace.UpdateWorkspace(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.errors.update_failed", "Workspace update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateWorkspaceUseCase) validateInput(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.data_required", "Workspace data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.id_required", "Workspace ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_required", "Workspace name is required [DEFAULT]"))
	}
	return nil
}

// enrichWorkspaceData adds audit information for updates
func (uc *UpdateWorkspaceUseCase) enrichWorkspaceData(workspace *workspacepb.Workspace) error {
	now := time.Now()

	// Set workspace audit fields for modification
	workspace.DateModified = &[]int64{now.Unix()}[0]
	workspace.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkspaceUseCase) validateBusinessRules(ctx context.Context, workspace *workspacepb.Workspace) error {
	if len(workspace.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_too_short", "Workspace name must be at least 2 characters long [DEFAULT]"))
	}

	if len(workspace.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_too_long", "Workspace name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if workspace.Description != "" && len(workspace.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.description_too_long", "Workspace description cannot exceed 1000 characters [DEFAULT]"))
	}

	return nil
}
