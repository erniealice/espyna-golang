package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// UpdateWorkflowRepositories groups all repository dependencies
type UpdateWorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// UpdateWorkflowServices groups all business service dependencies
type UpdateWorkflowServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateWorkflowUseCase handles the business logic for updating workflows
type UpdateWorkflowUseCase struct {
	repositories UpdateWorkflowRepositories
	services     UpdateWorkflowServices
}

// NewUpdateWorkflowUseCase creates use case with grouped dependencies
func NewUpdateWorkflowUseCase(
	repositories UpdateWorkflowRepositories,
	services UpdateWorkflowServices,
) *UpdateWorkflowUseCase {
	return &UpdateWorkflowUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateWorkflowUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateWorkflowUseCase with grouped parameters instead
func NewUpdateWorkflowUseCaseUngrouped(workflowRepo workflowpb.WorkflowDomainServiceServer) *UpdateWorkflowUseCase {
	repositories := UpdateWorkflowRepositories{
		Workflow: workflowRepo,
	}

	services := UpdateWorkflowServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateWorkflowUseCase(repositories, services)
}

// Execute performs the update workflow operation
func (uc *UpdateWorkflowUseCase) Execute(ctx context.Context, req *workflowpb.UpdateWorkflowRequest) (*workflowpb.UpdateWorkflowResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"workflow", ports.ActionUpdate); err != nil {
		return nil, err
	}

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

// executeWithTransaction executes workflow update within a transaction
func (uc *UpdateWorkflowUseCase) executeWithTransaction(ctx context.Context, enrichedWorkflow *workflowpb.Workflow) (*workflowpb.UpdateWorkflowResponse, error) {
	var result *workflowpb.UpdateWorkflowResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedWorkflow)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow.errors.update_failed", "Workflow update failed [DEFAULT]")
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

// executeCore contains the core business logic for updating a workflow
func (uc *UpdateWorkflowUseCase) executeCore(ctx context.Context, enrichedWorkflow *workflowpb.Workflow) (*workflowpb.UpdateWorkflowResponse, error) {
	// Delegate to repository
	return uc.repositories.Workflow.UpdateWorkflow(ctx, &workflowpb.UpdateWorkflowRequest{
		Data: enrichedWorkflow,
	})
}

// applyBusinessLogic applies business rules and returns enriched workflow
func (uc *UpdateWorkflowUseCase) applyBusinessLogic(workflow *workflowpb.Workflow) *workflowpb.Workflow {
	now := time.Now()

	// Business logic: Update modification audit fields
	workflow.DateModified = &[]int64{now.UnixMilli()}[0]
	workflow.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return workflow
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkflowUseCase) validateBusinessRules(ctx context.Context, workflow *workflowpb.Workflow) error {
	// Business rule: Required data validation
	if workflow == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.data_required", "Workflow data is required [DEFAULT]"))
	}

	// Business rule: ID is required for updating
	if workflow.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.id_required", "Workflow ID is required for update operations [DEFAULT]"))
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

	// Business rule: Name format validation
	if err := uc.validateWorkflowName(workflow.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.name_invalid", "Workflow name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if workflow.Description != nil && len(*workflow.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.description_too_long", "Workflow description cannot exceed 1000 characters [DEFAULT]"))
	}

	return nil
}

// validateWorkflowName validates workflow name format
func (uc *UpdateWorkflowUseCase) validateWorkflowName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid workflow name format")
	}
	return nil
}
