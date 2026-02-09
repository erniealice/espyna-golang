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

// CreateWorkspaceRepositories groups all repository dependencies
type CreateWorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// CreateWorkspaceServices groups all business service dependencies
type CreateWorkspaceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateWorkspaceUseCase handles the business logic for creating workspaces
type CreateWorkspaceUseCase struct {
	repositories CreateWorkspaceRepositories
	services     CreateWorkspaceServices
}

// NewCreateWorkspaceUseCase creates use case with grouped dependencies
func NewCreateWorkspaceUseCase(
	repositories CreateWorkspaceRepositories,
	services CreateWorkspaceServices,
) *CreateWorkspaceUseCase {
	return &CreateWorkspaceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateWorkspaceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateWorkspaceUseCase with grouped parameters instead
func NewCreateWorkspaceUseCaseUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *CreateWorkspaceUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateWorkspaceRepositories{
		Workspace: workspaceRepo,
	}

	services := CreateWorkspaceServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateWorkspaceUseCase(repositories, services)
}

func (uc *CreateWorkspaceUseCase) Execute(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspace, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workspace creation within a transaction
func (uc *CreateWorkspaceUseCase) executeWithTransaction(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	var result *workspacepb.CreateWorkspaceResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workspace.errors.creation_failed", "Workspace creation failed [DEFAULT]")
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
func (uc *CreateWorkspaceUseCase) executeCore(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichWorkspaceData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Workspace.CreateWorkspace(ctx, req)
}

// validateInput validates the input request
func (uc *CreateWorkspaceUseCase) validateInput(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.data_required", "Workspace data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_required", "Workspace name is required [DEFAULT]"))
	}
	return nil
}

// enrichWorkspaceData adds generated fields and audit information
func (uc *CreateWorkspaceUseCase) enrichWorkspaceData(workspace *workspacepb.Workspace) error {
	now := time.Now()

	// Generate Workspace ID if not provided
	if workspace.Id == "" {
		workspace.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	workspace.DateCreated = &[]int64{now.Unix()}[0]
	workspace.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	workspace.DateModified = &[]int64{now.Unix()}[0]
	workspace.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	workspace.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkspaceUseCase) validateBusinessRules(ctx context.Context, workspace *workspacepb.Workspace) error {
	// Validate name length
	if len(workspace.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_too_short", "Workspace name must be at least 2 characters long [DEFAULT]"))
	}

	if len(workspace.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.name_too_long", "Workspace name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if workspace.Description != "" && len(workspace.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace.validation.description_too_long", "Workspace description cannot exceed 500 characters [DEFAULT]"))
	}

	return nil
}
