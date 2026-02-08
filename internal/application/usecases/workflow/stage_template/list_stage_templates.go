package stage_template

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

// ListStageTemplatesRepositories groups all repository dependencies
type ListStageTemplatesRepositories struct {
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Primary entity repository
}

// ListStageTemplatesServices groups all business service dependencies
type ListStageTemplatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListStageTemplatesUseCase handles the business logic for listing stage templates
type ListStageTemplatesUseCase struct {
	repositories ListStageTemplatesRepositories
	services     ListStageTemplatesServices
}

// NewListStageTemplatesUseCase creates use case with grouped dependencies
func NewListStageTemplatesUseCase(
	repositories ListStageTemplatesRepositories,
	services ListStageTemplatesServices,
) *ListStageTemplatesUseCase {
	return &ListStageTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListStageTemplatesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListStageTemplatesUseCase with grouped parameters instead
func NewListStageTemplatesUseCaseUngrouped(stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *ListStageTemplatesUseCase {
	repositories := ListStageTemplatesRepositories{
		StageTemplate: stageTemplateRepo,
	}

	services := ListStageTemplatesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListStageTemplatesUseCase(repositories, services)
}

// Execute performs the list stage templates operation
func (uc *ListStageTemplatesUseCase) Execute(ctx context.Context, req *stageTemplatepb.ListStageTemplatesRequest) (*stageTemplatepb.ListStageTemplatesResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.request_required", "Request is required for stage templates [DEFAULT]"))
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

// executeWithTransaction executes stage template listing within a transaction
func (uc *ListStageTemplatesUseCase) executeWithTransaction(ctx context.Context, req *stageTemplatepb.ListStageTemplatesRequest) (*stageTemplatepb.ListStageTemplatesResponse, error) {
	var result *stageTemplatepb.ListStageTemplatesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage_template.errors.list_failed", "Stage template listing failed [DEFAULT]")
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

// executeCore contains the core business logic for listing stage templates
func (uc *ListStageTemplatesUseCase) executeCore(ctx context.Context, req *stageTemplatepb.ListStageTemplatesRequest) (*stageTemplatepb.ListStageTemplatesResponse, error) {
	// Delegate to repository
	return uc.repositories.StageTemplate.ListStageTemplates(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListStageTemplatesUseCase) applyBusinessLogic(req *stageTemplatepb.ListStageTemplatesRequest) *stageTemplatepb.ListStageTemplatesRequest {
	enrichedReq := &stageTemplatepb.ListStageTemplatesRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	}

	// Set default pagination if not provided
	if enrichedReq.Pagination == nil {
		enrichedReq.Pagination = &commonpb.PaginationRequest{
			Limit: 20,
		}
	} else if enrichedReq.Pagination.Limit <= 0 {
		enrichedReq.Pagination.Limit = 20
	} else if enrichedReq.Pagination.Limit > 100 {
		enrichedReq.Pagination.Limit = 100
	}

	return enrichedReq
}

// validateBusinessRules enforces business constraints
func (uc *ListStageTemplatesUseCase) validateBusinessRules(ctx context.Context, req *stageTemplatepb.ListStageTemplatesRequest) error {
	// Business rule: Pagination limit validation if provided
	if req.Pagination != nil && req.Pagination.Limit < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.page_size_negative", "Page size cannot be negative [DEFAULT]"))
	}

	if req.Pagination != nil && req.Pagination.Limit > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.page_size_too_large", "Page size cannot exceed 1000 [DEFAULT]"))
	}

	return nil
}
