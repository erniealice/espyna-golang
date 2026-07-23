package workflow_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflowTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// DeleteWorkflowTemplateRepositories groups all repository dependencies
type DeleteWorkflowTemplateRepositories struct {
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer               // Workspace repository for foreign key validation
}

// DeleteWorkflowTemplateServices groups all business service dependencies
type DeleteWorkflowTemplateServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
func NewDeleteWorkflowTemplateUseCaseUngrouped(workflowTemplateRepo workflowTemplatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *DeleteWorkflowTemplateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteWorkflowTemplateRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := DeleteWorkflowTemplateServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeleteWorkflowTemplateUseCase(repositories, services)
}

// Execute performs the delete workflow template operation
func (uc *DeleteWorkflowTemplateUseCase) Execute(ctx context.Context, req *workflowTemplatepb.DeleteWorkflowTemplateRequest) (*workflowTemplatepb.DeleteWorkflowTemplateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "workflow_template",
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.request_required", "Request is required for workflow templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes workflow template deletion within a transaction
func (uc *DeleteWorkflowTemplateUseCase) executeWithTransaction(ctx context.Context, workflowTemplate *workflowTemplatepb.WorkflowTemplate) (*workflowTemplatepb.DeleteWorkflowTemplateResponse, error) {
	var result *workflowTemplatepb.DeleteWorkflowTemplateResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, workflowTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "workflow_template.errors.deletion_failed", "Workflow template deletion failed [DEFAULT]")
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
func (uc *DeleteWorkflowTemplateUseCase) executeCore(ctx context.Context, workflowTemplate *workflowTemplatepb.WorkflowTemplate) (*workflowTemplatepb.DeleteWorkflowTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.DeleteWorkflowTemplate(ctx, &workflowTemplatepb.DeleteWorkflowTemplateRequest{
		Data: workflowTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *DeleteWorkflowTemplateUseCase) validateBusinessRules(ctx context.Context, workflowTemplate *workflowTemplatepb.WorkflowTemplate) error {
	// Business rule: Required data validation
	if workflowTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.data_required", "Workflow template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for deletion
	if workflowTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.id_required", "Workflow template ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowTemplateID(workflowTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.id_invalid", "Workflow template ID format is invalid [DEFAULT]"))
	}

	// Business rule: Cannot delete active workflow templates
	if workflowTemplate.Active {
		// Note: This check might be better performed at the repository level
		// where we have the full entity data from the database
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.cannot_delete_active", "Cannot delete active workflow templates [DEFAULT]"))
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
