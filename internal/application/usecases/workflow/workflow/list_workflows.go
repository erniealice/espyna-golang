package workflow

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

// ListWorkflowsRepositories groups all repository dependencies
type ListWorkflowsRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer // Primary entity repository
}

// ListWorkflowsServices groups all business service dependencies
type ListWorkflowsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListWorkflowsUseCase handles the business logic for listing workflows
type ListWorkflowsUseCase struct {
	repositories ListWorkflowsRepositories
	services     ListWorkflowsServices
}

// NewListWorkflowsUseCase creates use case with grouped dependencies
func NewListWorkflowsUseCase(
	repositories ListWorkflowsRepositories,
	services ListWorkflowsServices,
) *ListWorkflowsUseCase {
	return &ListWorkflowsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListWorkflowsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListWorkflowsUseCase with grouped parameters instead
func NewListWorkflowsUseCaseUngrouped(workflowRepo workflowpb.WorkflowDomainServiceServer) *ListWorkflowsUseCase {
	repositories := ListWorkflowsRepositories{
		Workflow: workflowRepo,
	}

	services := ListWorkflowsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListWorkflowsUseCase(repositories, services)
}

// Execute performs the list workflows operation
func (uc *ListWorkflowsUseCase) Execute(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.request_required", "Request is required for workflows [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Apply business logic defaults
	enrichedRequest := uc.applyBusinessLogic(req)

	// Use transaction service if available (for consistent reads)
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedRequest)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedRequest)
}

// executeWithTransaction executes workflow listing within a transaction
func (uc *ListWorkflowsUseCase) executeWithTransaction(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
	var result *workflowpb.ListWorkflowsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow.errors.list_failed", "Workflow listing failed [DEFAULT]")
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

// executeCore contains the core business logic for listing workflows
func (uc *ListWorkflowsUseCase) executeCore(ctx context.Context, req *workflowpb.ListWorkflowsRequest) (*workflowpb.ListWorkflowsResponse, error) {
	// Delegate to repository
	return uc.repositories.Workflow.ListWorkflows(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListWorkflowsUseCase) applyBusinessLogic(req *workflowpb.ListWorkflowsRequest) *workflowpb.ListWorkflowsRequest {
	// Create enriched request with new proto fields
	enrichedReq := &workflowpb.ListWorkflowsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// Business logic: Set default pagination if not provided
	if enrichedReq.Pagination == nil {
		enrichedReq.Pagination = &commonpb.PaginationRequest{Limit: 20}
	} else if enrichedReq.Pagination.Limit <= 0 {
		enrichedReq.Pagination.Limit = 20
	} else if enrichedReq.Pagination.Limit > 100 {
		enrichedReq.Pagination.Limit = 100
	}

	return enrichedReq
}

// validateBusinessRules enforces business constraints
func (uc *ListWorkflowsUseCase) validateBusinessRules(ctx context.Context, req *workflowpb.ListWorkflowsRequest) error {
	// Business rule: Pagination validation if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.limit_negative", "Limit cannot be negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow.validation.limit_too_large", "Limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	return nil
}
