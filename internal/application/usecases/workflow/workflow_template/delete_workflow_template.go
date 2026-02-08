package workflow_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// DeleteWorkflowTemplateRepositories groups all repository dependencies
type DeleteWorkflowTemplateRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// DeleteWorkflowTemplateServices groups all business service dependencies
type DeleteWorkflowTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteWorkflowTemplateUseCase handles the business logic for deleting workflow templates
type DeleteWorkflowTemplateUseCase struct {
	repositories DeleteWorkflowTemplateRepositories
	services     DeleteWorkflowTemplateServices
}

// NewDeleteWorkflowTemplateUseCase creates use case with grouped dependencies
func NewDeleteWorkflowTemplateUseCase(
	repositories DeleteWorkflowTemplateRepositories,
	services DeleteWorkflowTemplateServices,
) *DeleteWorkflowTemplateUseCase {
	return &DeleteWorkflowTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteWorkflowTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteWorkflowTemplateUseCase with grouped parameters instead
func NewDeleteWorkflowTemplateUseCaseUngrouped(workflowTemplateRepo workflow_templatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *DeleteWorkflowTemplateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteWorkflowTemplateRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := DeleteWorkflowTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteWorkflowTemplateUseCase(repositories, services)
}

// Execute performs the delete workflow template operation
func (uc *DeleteWorkflowTemplateUseCase) Execute(ctx context.Context, req *workflow_templatepb.DeleteWorkflowTemplateRequest) (*workflow_templatepb.DeleteWorkflowTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.request_required", "Request is required for workflow templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes workflow template deletion within a transaction
func (uc *DeleteWorkflowTemplateUseCase) executeWithTransaction(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.DeleteWorkflowTemplateResponse, error) {
	var result *workflow_templatepb.DeleteWorkflowTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, workflowTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow_template.errors.deletion_failed", "Workflow template deletion failed [DEFAULT]")
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

// executeCore contains the core business logic for deleting a workflow template
func (uc *DeleteWorkflowTemplateUseCase) executeCore(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.DeleteWorkflowTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.DeleteWorkflowTemplate(ctx, &workflow_templatepb.DeleteWorkflowTemplateRequest{
		Data: workflowTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *DeleteWorkflowTemplateUseCase) validateBusinessRules(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) error {
	// Business rule: Required data validation
	if workflowTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.data_required", "Workflow template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for deletion
	if workflowTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.id_required", "Workflow template ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowTemplateID(workflowTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.id_invalid", "Workflow template ID format is invalid [DEFAULT]"))
	}

	// Business rule: Cannot delete active workflow templates
	if workflowTemplate.Active {
		// Note: This check might be better performed at the repository level
		// where we have the full entity data from the database
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.cannot_delete_active", "Cannot delete active workflow templates [DEFAULT]"))
	}

	return nil
}

// validateWorkflowTemplateID validates workflow template ID format
func (uc *DeleteWorkflowTemplateUseCase) validateWorkflowTemplateID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if strings.TrimSpace(id) == "" {
		return errors.New("workflow template ID cannot be empty")
	}

	if len(id) < 3 {
		return errors.New("workflow template ID must be at least 3 characters long")
	}

	if len(id) > 100 {
		return errors.New("workflow template ID cannot exceed 100 characters")
	}

	// Allow alphanumeric characters, hyphens, and underscores
	idRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !idRegex.MatchString(id) {
		return errors.New("workflow template ID contains invalid characters")
	}

	return nil
}
