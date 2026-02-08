package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// DeleteWorkflowRepositories groups all repository dependencies
type DeleteWorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// DeleteWorkflowServices groups all business service dependencies
type DeleteWorkflowServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteWorkflowUseCase handles the business logic for deleting workflows
type DeleteWorkflowUseCase struct {
	repositories DeleteWorkflowRepositories
	services     DeleteWorkflowServices
}

// NewDeleteWorkflowUseCase creates use case with grouped dependencies
func NewDeleteWorkflowUseCase(
	repositories DeleteWorkflowRepositories,
	services DeleteWorkflowServices,
) *DeleteWorkflowUseCase {
	return &DeleteWorkflowUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteWorkflowUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteWorkflowUseCase with grouped parameters instead
func NewDeleteWorkflowUseCaseUngrouped(workflowRepo workflowpb.WorkflowDomainServiceServer) *DeleteWorkflowUseCase {
	repositories := DeleteWorkflowRepositories{
		Workflow: workflowRepo,
	}

	services := DeleteWorkflowServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteWorkflowUseCase(repositories, services)
}

// Execute performs the delete workflow operation
func (uc *DeleteWorkflowUseCase) Execute(ctx context.Context, req *workflowpb.DeleteWorkflowRequest) (*workflowpb.DeleteWorkflowResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.request_required", "Request is required for workflows [DEFAULT]"))
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

// executeWithTransaction executes workflow deletion within a transaction
func (uc *DeleteWorkflowUseCase) executeWithTransaction(ctx context.Context, workflow *workflowpb.Workflow) (*workflowpb.DeleteWorkflowResponse, error) {
	var result *workflowpb.DeleteWorkflowResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, workflow)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow.errors.deletion_failed", "Workflow deletion failed [DEFAULT]")
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

// executeCore contains the core business logic for deleting a workflow
func (uc *DeleteWorkflowUseCase) executeCore(ctx context.Context, workflow *workflowpb.Workflow) (*workflowpb.DeleteWorkflowResponse, error) {
	// Delegate to repository
	return uc.repositories.Workflow.DeleteWorkflow(ctx, &workflowpb.DeleteWorkflowRequest{
		Data: workflow,
	})
}

// validateBusinessRules enforces business constraints
func (uc *DeleteWorkflowUseCase) validateBusinessRules(ctx context.Context, workflow *workflowpb.Workflow) error {
	// Business rule: Required data validation
	if workflow == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.data_required", "Workflow data is required [DEFAULT]"))
	}

	// Business rule: ID is required for deleting
	if workflow.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.id_required", "Workflow ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowID(workflow.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.id_invalid", "Workflow ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateWorkflowID validates workflow ID format
func (uc *DeleteWorkflowUseCase) validateWorkflowID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if id == "" {
		return errors.New("workflow ID cannot be empty")
	}

	if len(id) < 3 {
		return errors.New("workflow ID must be at least 3 characters long")
	}

	if len(id) > 100 {
		return errors.New("workflow ID cannot exceed 100 characters")
	}

	// Allow alphanumeric characters, hyphens, and underscores
	idRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !idRegex.MatchString(id) {
		return errors.New("workflow ID contains invalid characters")
	}

	return nil
}
