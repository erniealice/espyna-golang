package workflow_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// ReadWorkflowTemplateRepositories groups all repository dependencies
type ReadWorkflowTemplateRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// ReadWorkflowTemplateServices groups all business service dependencies
type ReadWorkflowTemplateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadWorkflowTemplateUseCase handles the business logic for reading workflow templates
type ReadWorkflowTemplateUseCase struct {
	repositories ReadWorkflowTemplateRepositories
	services     ReadWorkflowTemplateServices
}

// NewReadWorkflowTemplateUseCase creates use case with grouped dependencies
func NewReadWorkflowTemplateUseCase(
	repositories ReadWorkflowTemplateRepositories,
	services ReadWorkflowTemplateServices,
) *ReadWorkflowTemplateUseCase {
	return &ReadWorkflowTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadWorkflowTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadWorkflowTemplateUseCase with grouped parameters instead
func NewReadWorkflowTemplateUseCaseUngrouped(workflowTemplateRepo workflow_templatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *ReadWorkflowTemplateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadWorkflowTemplateRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := ReadWorkflowTemplateServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewReadWorkflowTemplateUseCase(repositories, services)
}

// Execute performs the read workflow template operation
func (uc *ReadWorkflowTemplateUseCase) Execute(ctx context.Context, req *workflow_templatepb.ReadWorkflowTemplateRequest) (*workflow_templatepb.ReadWorkflowTemplateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "workflow_template",
		Action: entityid.ActionRead,
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

	// Use transaction service if available (for consistent reads)
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes workflow template read within a transaction
func (uc *ReadWorkflowTemplateUseCase) executeWithTransaction(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.ReadWorkflowTemplateResponse, error) {
	var result *workflow_templatepb.ReadWorkflowTemplateResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, workflowTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "workflow_template.errors.read_failed", "Workflow template read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading a workflow template
func (uc *ReadWorkflowTemplateUseCase) executeCore(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.ReadWorkflowTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.ReadWorkflowTemplate(ctx, &workflow_templatepb.ReadWorkflowTemplateRequest{
		Data: workflowTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *ReadWorkflowTemplateUseCase) validateBusinessRules(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) error {
	// Business rule: Required data validation
	if workflowTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.data_required", "Workflow template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for reading
	if workflowTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.id_required", "Workflow template ID is required for read operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowTemplateID(workflowTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workflow_template.validation.id_invalid", "Workflow template ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateWorkflowTemplateID validates workflow template ID format
func (uc *ReadWorkflowTemplateUseCase) validateWorkflowTemplateID(id string) error {
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
