package activity

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
)

// ListActivitiesRepositories groups all repository dependencies
type ListActivitiesRepositories struct {
	Activity activitypb.ActivityDomainServiceServer // Primary entity repository
}

// ListActivitiesServices groups all business service dependencies
type ListActivitiesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListActivitiesUseCase handles the business logic for listing activities
type ListActivitiesUseCase struct {
	repositories ListActivitiesRepositories
	services     ListActivitiesServices
}

// NewListActivitiesUseCase creates use case with grouped dependencies
func NewListActivitiesUseCase(
	repositories ListActivitiesRepositories,
	services ListActivitiesServices,
) *ListActivitiesUseCase {
	return &ListActivitiesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListActivitiesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListActivitiesUseCase with grouped parameters instead
func NewListActivitiesUseCaseUngrouped(activityRepo activitypb.ActivityDomainServiceServer) *ListActivitiesUseCase {
	repositories := ListActivitiesRepositories{
		Activity: activityRepo,
	}

	services := ListActivitiesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListActivitiesUseCase(repositories, services)
}

// Execute performs the list activities operation
func (uc *ListActivitiesUseCase) Execute(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
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

// executeWithTransaction executes activity listing within a transaction
func (uc *ListActivitiesUseCase) executeWithTransaction(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	var result *activitypb.ListActivitiesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity.errors.list_failed", "Activity listing failed [DEFAULT]")
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

// executeCore contains the core business logic for listing activities
func (uc *ListActivitiesUseCase) executeCore(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	// Delegate to repository
	return uc.repositories.Activity.ListActivities(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListActivitiesUseCase) applyBusinessLogic(req *activitypb.ListActivitiesRequest) *activitypb.ListActivitiesRequest {
	// Create a copy to avoid modifying the original request
	enrichedReq := &activitypb.ListActivitiesRequest{
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
func (uc *ListActivitiesUseCase) validateBusinessRules(ctx context.Context, req *activitypb.ListActivitiesRequest) error {
	// Business rule: Pagination limit validation if provided
	if req.Pagination != nil && req.Pagination.Limit < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.pagination_limit_negative", "Pagination limit cannot be negative [DEFAULT]"))
	}

	if req.Pagination != nil && req.Pagination.Limit > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.pagination_limit_too_large", "Pagination limit cannot exceed 1000 [DEFAULT]"))
	}

	return nil
}
