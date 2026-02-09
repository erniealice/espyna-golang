package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// ReadWorkflowRepositories groups all repository dependencies
type ReadWorkflowRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// ReadWorkflowServices groups all business service dependencies
type ReadWorkflowServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadWorkflowUseCase handles the business logic for reading workflows
type ReadWorkflowUseCase struct {
	repositories ReadWorkflowRepositories
	services     ReadWorkflowServices
}

// NewReadWorkflowUseCase creates use case with grouped dependencies
func NewReadWorkflowUseCase(
	repositories ReadWorkflowRepositories,
	services ReadWorkflowServices,
) *ReadWorkflowUseCase {
	return &ReadWorkflowUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadWorkflowUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadWorkflowUseCase with grouped parameters instead
func NewReadWorkflowUseCaseUngrouped(workflowRepo workflowpb.WorkflowDomainServiceServer) *ReadWorkflowUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadWorkflowRepositories{
		Workflow: workflowRepo,
	}

	services := ReadWorkflowServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadWorkflowUseCase(repositories, services)
}

// Execute performs the read workflow operation
func (uc *ReadWorkflowUseCase) Execute(ctx context.Context, req *workflowpb.ReadWorkflowRequest) (*workflowpb.ReadWorkflowResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"workflow", ports.ActionRead); err != nil {
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

	// Use transaction service if available (for consistent reads)
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes workflow read within a transaction
func (uc *ReadWorkflowUseCase) executeWithTransaction(ctx context.Context, workflow *workflowpb.Workflow) (*workflowpb.ReadWorkflowResponse, error) {
	var result *workflowpb.ReadWorkflowResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, workflow)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow.errors.read_failed", "Workflow read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading a workflow
func (uc *ReadWorkflowUseCase) executeCore(ctx context.Context, workflow *workflowpb.Workflow) (*workflowpb.ReadWorkflowResponse, error) {
	// Delegate to repository
	return uc.repositories.Workflow.ReadWorkflow(ctx, &workflowpb.ReadWorkflowRequest{
		Data: workflow,
	})
}

// validateBusinessRules enforces business constraints
func (uc *ReadWorkflowUseCase) validateBusinessRules(ctx context.Context, workflow *workflowpb.Workflow) error {
	// Business rule: Required data validation
	if workflow == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.data_required", "Workflow data is required [DEFAULT]"))
	}

	// Business rule: ID is required for reading
	if workflow.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.id_required", "Workflow ID is required for read operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateWorkflowID(workflow.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.id_invalid", "Workflow ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateWorkflowID validates workflow ID format
func (uc *ReadWorkflowUseCase) validateWorkflowID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if strings.TrimSpace(id) == "" {
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
