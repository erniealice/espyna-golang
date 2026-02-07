package activity_template

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

// ListActivityTemplatesRepositories groups all repository dependencies
type ListActivityTemplatesRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
}

// ListActivityTemplatesServices groups all business service dependencies
type ListActivityTemplatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListActivityTemplatesUseCase handles the business logic for listing activity templates
type ListActivityTemplatesUseCase struct {
	repositories ListActivityTemplatesRepositories
	services     ListActivityTemplatesServices
}

// NewListActivityTemplatesUseCase creates use case with grouped dependencies
func NewListActivityTemplatesUseCase(
	repositories ListActivityTemplatesRepositories,
	services ListActivityTemplatesServices,
) *ListActivityTemplatesUseCase {
	return &ListActivityTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListActivityTemplatesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListActivityTemplatesUseCase with grouped parameters instead
func NewListActivityTemplatesUseCaseUngrouped(activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer) *ListActivityTemplatesUseCase {
	repositories := ListActivityTemplatesRepositories{
		ActivityTemplate: activityTemplateRepo,
	}

	services := ListActivityTemplatesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListActivityTemplatesUseCase(repositories, services)
}

// Execute performs the list activity templates operation
func (uc *ListActivityTemplatesUseCase) Execute(ctx context.Context, req *activityTemplatepb.ListActivityTemplatesRequest) (*activityTemplatepb.ListActivityTemplatesResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.request_required", "Request is required for activity templates [DEFAULT]"))
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

// executeWithTransaction executes activity template listing within a transaction
func (uc *ListActivityTemplatesUseCase) executeWithTransaction(ctx context.Context, req *activityTemplatepb.ListActivityTemplatesRequest) (*activityTemplatepb.ListActivityTemplatesResponse, error) {
	var result *activityTemplatepb.ListActivityTemplatesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity_template.errors.list_failed", "Activity template listing failed [DEFAULT]")
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

// executeCore contains the core business logic for listing activity templates
func (uc *ListActivityTemplatesUseCase) executeCore(ctx context.Context, req *activityTemplatepb.ListActivityTemplatesRequest) (*activityTemplatepb.ListActivityTemplatesResponse, error) {
	// Delegate to repository
	return uc.repositories.ActivityTemplate.ListActivityTemplates(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListActivityTemplatesUseCase) applyBusinessLogic(req *activityTemplatepb.ListActivityTemplatesRequest) *activityTemplatepb.ListActivityTemplatesRequest {
	enrichedReq := &activityTemplatepb.ListActivityTemplatesRequest{
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
func (uc *ListActivityTemplatesUseCase) validateBusinessRules(ctx context.Context, req *activityTemplatepb.ListActivityTemplatesRequest) error {
	// Business rule: Pagination limit validation if provided
	if req.Pagination != nil && req.Pagination.Limit < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.page_size_negative", "Page size cannot be negative [DEFAULT]"))
	}

	if req.Pagination != nil && req.Pagination.Limit > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.page_size_too_large", "Page size cannot exceed 1000 [DEFAULT]"))
	}

	return nil
}
