package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	enginepb "leapfor.xyz/esqyma/golang/v1/orchestration/engine"
)

// CreateWorkflowRepositories groups all repository dependencies
type CreateWorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// CreateWorkflowServices groups all business service dependencies
type CreateWorkflowServices struct {
	AuthorizationService  ports.AuthorizationService
	TransactionService    ports.TransactionService
	TranslationService    ports.TranslationService
	IDService             ports.IDService
	WorkflowEngineService ports.WorkflowEngineService
}

// CreateWorkflowUseCase handles the business logic for creating workflows
type CreateWorkflowUseCase struct {
	repositories CreateWorkflowRepositories
	services     CreateWorkflowServices
}

// NewCreateWorkflowUseCase creates use case with grouped dependencies
func NewCreateWorkflowUseCase(
	repositories CreateWorkflowRepositories,
	services CreateWorkflowServices,
) *CreateWorkflowUseCase {
	return &CreateWorkflowUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateWorkflowUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateWorkflowUseCase with grouped parameters instead
func NewCreateWorkflowUseCaseUngrouped(workflowRepo workflowpb.WorkflowDomainServiceServer) *CreateWorkflowUseCase {
	repositories := CreateWorkflowRepositories{
		Workflow: workflowRepo,
	}

	services := CreateWorkflowServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateWorkflowUseCase(repositories, services)
}

// Execute performs the create workflow operation
func (uc *CreateWorkflowUseCase) Execute(ctx context.Context, req *workflowpb.CreateWorkflowRequest) (*workflowpb.CreateWorkflowResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.request_required", "Request is required for workflows [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedWorkflow := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedWorkflow)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedWorkflow)
}

// executeWithTransaction executes workflow creation within a transaction
func (uc *CreateWorkflowUseCase) executeWithTransaction(ctx context.Context, enrichedWorkflow *workflowpb.Workflow) (*workflowpb.CreateWorkflowResponse, error) {
	var result *workflowpb.CreateWorkflowResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedWorkflow)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow.errors.creation_failed", "Workflow creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a workflow
func (uc *CreateWorkflowUseCase) executeCore(ctx context.Context, enrichedWorkflow *workflowpb.Workflow) (*workflowpb.CreateWorkflowResponse, error) {
	// Delegate to repository
	createRes, err := uc.repositories.Workflow.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: enrichedWorkflow,
	})
	if err != nil {
		return nil, err
	}

	// After creating, if there's a template ID, trigger the engine to start the workflow
	if uc.services.WorkflowEngineService != nil && enrichedWorkflow.WorkflowTemplateId != nil && *enrichedWorkflow.WorkflowTemplateId != "" {
		go func() {
			// We execute this in a goroutine to not block the response of the CreateWorkflow call.
			// The context of the initial request is likely to be cancelled, so we create a new background context.
			bgCtx := context.Background()

			inputJSON := ""
			if enrichedWorkflow.ContextJson != nil {
				inputJSON = *enrichedWorkflow.ContextJson
			}

			_, err := uc.services.WorkflowEngineService.StartWorkflowFromTemplate(bgCtx, &enginepb.StartWorkflowRequest{
				WorkflowTemplateId: *enrichedWorkflow.WorkflowTemplateId,
				InputJson:          inputJSON,
			})
			if err != nil {
				// We need a proper logger here, but for now, we'll just print.
				fmt.Printf("Error starting workflow from template in background: %v\n", err)
			}
		}()
	}

	return createRes, nil
}

// applyBusinessLogic applies business rules and returns enriched workflow
func (uc *CreateWorkflowUseCase) applyBusinessLogic(workflow *workflowpb.Workflow) *workflowpb.Workflow {
	now := time.Now()

	// Business logic: Generate Workflow ID if not provided
	if workflow.Id == "" {
		if uc.services.IDService != nil {
			workflow.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			workflow.Id = fmt.Sprintf("workflow-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new workflows
	workflow.Active = true

	// Business logic: Set default version if not provided
	if workflow.Version == nil || *workflow.Version == 0 {
		workflow.Version = &[]int32{1}[0]
	}

	// Business logic: Set creation audit fields
	workflow.DateCreated = &[]int64{now.UnixMilli()}[0]
	workflow.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	workflow.DateModified = &[]int64{now.UnixMilli()}[0]
	workflow.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return workflow
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkflowUseCase) validateBusinessRules(ctx context.Context, workflow *workflowpb.Workflow) error {
	// Business rule: Required data validation
	if workflow == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.data_required", "Workflow data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if workflow.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.name_required", "Workflow name is required [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(workflow.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.name_too_short", "Workflow name must be at least 2 characters long [DEFAULT]"))
	}

	if len(workflow.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.name_too_long", "Workflow name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation (alphanumeric, spaces, hyphens, underscores)
	if err := uc.validateWorkflowName(workflow.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.name_invalid", "Workflow name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if workflow.Description != nil && len(*workflow.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.description_too_long", "Workflow description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Workspace ID format validation if provided
	if workflow.WorkspaceId != nil && *workflow.WorkspaceId != "" {
		if len(*workflow.WorkspaceId) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.workspace_id_invalid", "Workspace ID is invalid [DEFAULT]"))
		}
	}

	return nil
}

// validateWorkflowName validates workflow name format
func (uc *CreateWorkflowUseCase) validateWorkflowName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid workflow name format")
	}
	return nil
}
