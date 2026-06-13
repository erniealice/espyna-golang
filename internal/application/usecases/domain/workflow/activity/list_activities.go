package activity

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
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
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListActivitiesUseCase(repositories, services)
}

// Execute performs the list activities operation
func (uc *ListActivitiesUseCase) Execute(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "activity",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Apply business logic defaults
	enrichedRequest := uc.applyBusinessLogic(req)

	// Use transaction service if available (for consistent reads)
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedRequest)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedRequest)
}

// executeWithTransaction executes activity listing within a transaction
func (uc *ListActivitiesUseCase) executeWithTransaction(ctx context.Context, req *activitypb.ListActivitiesRequest) (*activitypb.ListActivitiesResponse, error) {
	var result *activitypb.ListActivitiesResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "activity.errors.list_failed", "Activity listing failed [DEFAULT]")
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "activity.validation.pagination_limit_negative", "Pagination limit cannot be negative [DEFAULT]"))
	}

	if req.Pagination != nil && req.Pagination.Limit > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "activity.validation.pagination_limit_too_large", "Pagination limit cannot exceed 1000 [DEFAULT]"))
	}

	return nil
}
