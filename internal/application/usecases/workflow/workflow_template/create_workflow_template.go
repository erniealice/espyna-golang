package workflow_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// CreateWorkflowTemplateRepositories groups all repository dependencies
type CreateWorkflowTemplateRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// CreateWorkflowTemplateServices groups all business service dependencies
type CreateWorkflowTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateWorkflowTemplateUseCase handles the business logic for creating workflow templates
type CreateWorkflowTemplateUseCase struct {
	repositories CreateWorkflowTemplateRepositories
	services     CreateWorkflowTemplateServices
}

// NewCreateWorkflowTemplateUseCase creates use case with grouped dependencies
func NewCreateWorkflowTemplateUseCase(
	repositories CreateWorkflowTemplateRepositories,
	services CreateWorkflowTemplateServices,
) *CreateWorkflowTemplateUseCase {
	return &CreateWorkflowTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateWorkflowTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateWorkflowTemplateUseCase with grouped parameters instead
func NewCreateWorkflowTemplateUseCaseUngrouped(workflowTemplateRepo workflow_templatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *CreateWorkflowTemplateUseCase {
	repositories := CreateWorkflowTemplateRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := CreateWorkflowTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateWorkflowTemplateUseCase(repositories, services)
}

// Execute performs the create workflow template operation
func (uc *CreateWorkflowTemplateUseCase) Execute(ctx context.Context, req *workflow_templatepb.CreateWorkflowTemplateRequest) (*workflow_templatepb.CreateWorkflowTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"workflow_template", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.request_required", "Request is required for workflow templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment (ctx provides business_type from middleware)
	enrichedWorkflowTemplate := uc.applyBusinessLogic(ctx, req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedWorkflowTemplate)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedWorkflowTemplate)
}

// executeWithTransaction executes workflow template creation within a transaction
func (uc *CreateWorkflowTemplateUseCase) executeWithTransaction(ctx context.Context, enrichedWorkflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.CreateWorkflowTemplateResponse, error) {
	var result *workflow_templatepb.CreateWorkflowTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedWorkflowTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow_template.errors.creation_failed", "Workflow template creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a workflow template
func (uc *CreateWorkflowTemplateUseCase) executeCore(ctx context.Context, enrichedWorkflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.CreateWorkflowTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.CreateWorkflowTemplate(ctx, &workflow_templatepb.CreateWorkflowTemplateRequest{
		Data: enrichedWorkflowTemplate,
	})
}

// applyBusinessLogic applies business rules and returns enriched workflow template
// Business type is extracted from context (set by HTTP middleware based on tenant/app)
func (uc *CreateWorkflowTemplateUseCase) applyBusinessLogic(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) *workflow_templatepb.WorkflowTemplate {
	now := time.Now()

	// Business logic: Generate Workflow Template ID if not provided
	if workflowTemplate.Id == "" {
		if uc.services.IDService != nil {
			workflowTemplate.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			workflowTemplate.Id = fmt.Sprintf("workflow-template-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new workflow templates
	workflowTemplate.Active = true

	// Business logic: Set default status if not provided
	if workflowTemplate.Status == "" {
		workflowTemplate.Status = "draft"
	}

	// Business logic: Set business_type from context (defaults to "education")
	// Context is set by HTTP middleware based on tenant/app configuration
	workflowTemplate.BusinessType = contextutil.ExtractBusinessTypeFromContext(ctx)

	// Business logic: Set default version if not provided
	if workflowTemplate.Version == nil || *workflowTemplate.Version == 0 {
		workflowTemplate.Version = &[]int32{1}[0]
	}

	// Business logic: Set creation audit fields
	workflowTemplate.DateCreated = &[]int64{now.UnixMilli()}[0]
	workflowTemplate.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	workflowTemplate.DateModified = &[]int64{now.UnixMilli()}[0]
	workflowTemplate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return workflowTemplate
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkflowTemplateUseCase) validateBusinessRules(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) error {
	// Business rule: Required data validation
	if workflowTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.data_required", "Workflow template data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if workflowTemplate.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_required", "Workflow template name is required [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(workflowTemplate.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_too_short", "Workflow template name must be at least 2 characters long [DEFAULT]"))
	}

	if len(workflowTemplate.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_too_long", "Workflow template name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation (alphanumeric, spaces, hyphens, underscores)
	if err := uc.validateWorkflowTemplateName(workflowTemplate.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_invalid", "Workflow template name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if workflowTemplate.Description != nil && len(*workflowTemplate.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.description_too_long", "Workflow template description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Status validation if provided
	if workflowTemplate.Status != "" {
		validStatuses := []string{"draft", "active", "inactive", "archived"}
		isValidStatus := false
		for _, status := range validStatuses {
			if workflowTemplate.Status == status {
				isValidStatus = true
				break
			}
		}
		if !isValidStatus {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.status_invalid", "Invalid workflow template status [DEFAULT]"))
		}
	}

	// Note: business_type is set from context (HTTP middleware) in applyBusinessLogic
	// No validation needed here since it's controlled by the application context

	// Business rule: Workspace ID foreign key validation if provided
	if workflowTemplate.WorkspaceId != nil && *workflowTemplate.WorkspaceId != "" {
		if err := uc.validateWorkspaceExists(ctx, *workflowTemplate.WorkspaceId); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.workspace_not_found", "Workspace not found [DEFAULT]"))
		}
	}

	return nil
}

// validateWorkflowTemplateName validates workflow template name format
func (uc *CreateWorkflowTemplateUseCase) validateWorkflowTemplateName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid workflow template name format")
	}
	return nil
}

// validateWorkspaceExists checks if the workspace exists
func (uc *CreateWorkflowTemplateUseCase) validateWorkspaceExists(ctx context.Context, workspaceId string) error {
	if uc.repositories.Workspace == nil {
		// If workspace repository is not available, skip validation
		return nil
	}

	// Check if workspace exists and is active
	req := &workspacepb.ReadWorkspaceRequest{
		Data: &workspacepb.Workspace{
			Id: workspaceId,
		},
	}

	resp, err := uc.repositories.Workspace.ReadWorkspace(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to validate workspace existence: %w", err)
	}

	if resp == nil || len(resp.Data) == 0 {
		return errors.New("workspace not found")
	}

	// Check if workspace is active
	if len(resp.Data) > 0 && !resp.Data[0].Active {
		return errors.New("workspace is not active")
	}

	return nil
}
