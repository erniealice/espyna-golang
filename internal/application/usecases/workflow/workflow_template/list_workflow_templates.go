package workflow_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// ListWorkflowTemplatesRepositories groups all repository dependencies
type ListWorkflowTemplatesRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer // Primary entity repository
	Workspace        workspacepb.WorkspaceDomainServiceServer                // Workspace repository for foreign key validation
}

// ListWorkflowTemplatesServices groups all business service dependencies
type ListWorkflowTemplatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListWorkflowTemplatesUseCase handles the business logic for listing workflow templates
type ListWorkflowTemplatesUseCase struct {
	repositories ListWorkflowTemplatesRepositories
	services     ListWorkflowTemplatesServices
}

// NewListWorkflowTemplatesUseCase creates use case with grouped dependencies
func NewListWorkflowTemplatesUseCase(
	repositories ListWorkflowTemplatesRepositories,
	services ListWorkflowTemplatesServices,
) *ListWorkflowTemplatesUseCase {
	return &ListWorkflowTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListWorkflowTemplatesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListWorkflowTemplatesUseCase with grouped parameters instead
func NewListWorkflowTemplatesUseCaseUngrouped(workflowTemplateRepo workflow_templatepb.WorkflowTemplateDomainServiceServer, workspaceRepo workspacepb.WorkspaceDomainServiceServer) *ListWorkflowTemplatesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListWorkflowTemplatesRepositories{
		WorkflowTemplate: workflowTemplateRepo,
		Workspace:        workspaceRepo,
	}

	services := ListWorkflowTemplatesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListWorkflowTemplatesUseCase(repositories, services)
}

// Execute performs the list workflow templates operation
func (uc *ListWorkflowTemplatesUseCase) Execute(ctx context.Context, req *workflow_templatepb.ListWorkflowTemplatesRequest) (*workflow_templatepb.ListWorkflowTemplatesResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.request_required", "Request is required for workflow templates [DEFAULT]"))
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

// executeWithTransaction executes workflow template listing within a transaction
func (uc *ListWorkflowTemplatesUseCase) executeWithTransaction(ctx context.Context, req *workflow_templatepb.ListWorkflowTemplatesRequest) (*workflow_templatepb.ListWorkflowTemplatesResponse, error) {
	var result *workflow_templatepb.ListWorkflowTemplatesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workflow_template.errors.list_failed", "Workflow template listing failed [DEFAULT]")
			return errors.New(translatedError + ": " + err.Error())
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing workflow templates
func (uc *ListWorkflowTemplatesUseCase) executeCore(ctx context.Context, req *workflow_templatepb.ListWorkflowTemplatesRequest) (*workflow_templatepb.ListWorkflowTemplatesResponse, error) {
	// Delegate to repository
	return uc.repositories.WorkflowTemplate.ListWorkflowTemplates(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListWorkflowTemplatesUseCase) applyBusinessLogic(req *workflow_templatepb.ListWorkflowTemplatesRequest) *workflow_templatepb.ListWorkflowTemplatesRequest {
	// Create enriched request with new proto fields
	enrichedReq := &workflow_templatepb.ListWorkflowTemplatesRequest{
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
func (uc *ListWorkflowTemplatesUseCase) validateBusinessRules(ctx context.Context, req *workflow_templatepb.ListWorkflowTemplatesRequest) error {
	// Business rule: Pagination validation if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.limit_negative", "Limit cannot be negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workflow_template.validation.limit_too_large", "Limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	return nil
}
