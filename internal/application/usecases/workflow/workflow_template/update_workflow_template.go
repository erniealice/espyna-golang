package workflow_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// UpdateWorkflowTemplateRepositories groups all repository dependencies
type UpdateWorkflowTemplateRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// UpdateWorkflowTemplateServices groups all business service dependencies
type UpdateWorkflowTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateWorkflowTemplateUseCase handles the business logic for updating workflow templates
type UpdateWorkflowTemplateUseCase struct {
	repositories UpdateWorkflowTemplateRepositories
	services     UpdateWorkflowTemplateServices
}

// NewUpdateWorkflowTemplateUseCase creates use case with grouped dependencies
func NewUpdateWorkflowTemplateUseCase(
	repositories UpdateWorkflowTemplateRepositories,
	services UpdateWorkflowTemplateServices,
) *UpdateWorkflowTemplateUseCase {
	return &UpdateWorkflowTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateWorkflowTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateWorkflowTemplateUseCase with grouped parameters instead
func NewUpdateWorkflowTemplateUseCaseUngrouped(workflowTemplateRepo workflow_templatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *UpdateWorkflowTemplateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateWorkflowTemplateRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := UpdateWorkflowTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateWorkflowTemplateUseCase(repositories, services)
}

// Execute performs the update workflow template operation
func (uc *UpdateWorkflowTemplateUseCase) Execute(ctx context.Context, req *workflow_templatepb.UpdateWorkflowTemplateRequest) (*workflow_templatepb.UpdateWorkflowTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"workflow_template", ports.ActionUpdate); err != nil {
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

	// Business enrichment
	enrichedWorkflowTemplate := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedWorkflowTemplate)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedWorkflowTemplate)
}

// executeWithTransaction executes workflow template update within a transaction
func (uc *UpdateWorkflowTemplateUseCase) executeWithTransaction(ctx context.Context, enrichedWorkflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.UpdateWorkflowTemplateResponse, error) {
	var result *workflow_templatepb.UpdateWorkflowTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedWorkflowTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow_template.errors.update_failed", "Workflow template update failed [DEFAULT]")
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

// executeCore contains the core business logic for updating a workflow template
func (uc *UpdateWorkflowTemplateUseCase) executeCore(ctx context.Context, enrichedWorkflowTemplate *workflow_templatepb.WorkflowTemplate) (*workflow_templatepb.UpdateWorkflowTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.UpdateWorkflowTemplate(ctx, &workflow_templatepb.UpdateWorkflowTemplateRequest{
		Data: enrichedWorkflowTemplate,
	})
}

// applyBusinessLogic applies business rules and returns enriched workflow template
func (uc *UpdateWorkflowTemplateUseCase) applyBusinessLogic(workflowTemplate *workflow_templatepb.WorkflowTemplate) *workflow_templatepb.WorkflowTemplate {
	now := time.Now()

	// Business logic: Set modification audit fields
	workflowTemplate.DateModified = &[]int64{now.UnixMilli()}[0]
	workflowTemplate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Business logic: Increment version if not explicitly set
	if workflowTemplate.Version == nil || *workflowTemplate.Version == 0 {
		// Note: In a real implementation, you might want to fetch the current version and increment it
		// For now, we'll set it to 1 as a default
		workflowTemplate.Version = &[]int32{1}[0]
	}

	return workflowTemplate
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkflowTemplateUseCase) validateBusinessRules(ctx context.Context, workflowTemplate *workflow_templatepb.WorkflowTemplate) error {
	// Business rule: Required data validation
	if workflowTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.data_required", "Workflow template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for updating
	if workflowTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.id_required", "Workflow template ID is required for update operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowTemplateID(workflowTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.id_invalid", "Workflow template ID format is invalid [DEFAULT]"))
	}

	// Business rule: Name length constraints if provided
	if workflowTemplate.Name != "" {
		if len(workflowTemplate.Name) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_too_short", "Workflow template name must be at least 2 characters long [DEFAULT]"))
		}

		if len(workflowTemplate.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_too_long", "Workflow template name cannot exceed 100 characters [DEFAULT]"))
		}

		// Business rule: Name format validation if provided
		if err := uc.validateWorkflowTemplateName(workflowTemplate.Name); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.name_invalid", "Workflow template name contains invalid characters [DEFAULT]"))
		}
	}

	// Business rule: Description length constraints if provided
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

	// Note: business_type is typically set from context (HTTP middleware) during creation
	// Updates to business_type are not validated since it's controlled by the application context

	// Business rule: Configuration JSON validation if provided
	if workflowTemplate.ConfigurationJson != nil && strings.TrimSpace(*workflowTemplate.ConfigurationJson) != "" {
		if err := uc.validateConfigurationJSON(*workflowTemplate.ConfigurationJson); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.configuration_json_invalid", "Invalid configuration JSON format [DEFAULT]"))
		}
	}

	// Business rule: Workspace ID foreign key validation if provided
	if workflowTemplate.WorkspaceId != nil && *workflowTemplate.WorkspaceId != "" {
		if err := uc.validateWorkspaceExists(ctx, *workflowTemplate.WorkspaceId); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.workspace_not_found", "Workspace not found [DEFAULT]"))
		}
	}

	return nil
}

// validateWorkflowTemplateID validates workflow template ID format
func (uc *UpdateWorkflowTemplateUseCase) validateWorkflowTemplateID(id string) error {
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

// validateWorkflowTemplateName validates workflow template name format
func (uc *UpdateWorkflowTemplateUseCase) validateWorkflowTemplateName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid workflow template name format")
	}
	return nil
}

// validateConfigurationJSON validates basic JSON format (simplified validation)
func (uc *UpdateWorkflowTemplateUseCase) validateConfigurationJSON(jsonStr string) error {
	// Basic validation: check if it starts with { and ends with } (object) or [ and ] (array)
	trimmed := strings.TrimSpace(jsonStr)
	if !(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) &&
		!(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return errors.New("invalid JSON format")
	}

	// Note: In a production environment, you might want to use json.Valid() or more sophisticated validation
	return nil
}

// validateWorkspaceExists checks if the workspace exists
func (uc *UpdateWorkflowTemplateUseCase) validateWorkspaceExists(ctx context.Context, workspaceId string) error {
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
