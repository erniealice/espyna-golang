package stage

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// ListStagesRepositories groups all repository dependencies
type ListStagesRepositories struct {
	Stage stagepb.StageDomainServiceServer // Primary entity repository
}

// ListStagesServices groups all business service dependencies
type ListStagesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListStagesUseCase handles the business logic for listing stages
type ListStagesUseCase struct {
	repositories ListStagesRepositories
	services     ListStagesServices
}

// NewListStagesUseCase creates use case with grouped dependencies
func NewListStagesUseCase(
	repositories ListStagesRepositories,
	services ListStagesServices,
) *ListStagesUseCase {
	return &ListStagesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListStagesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListStagesUseCase with grouped parameters instead
func NewListStagesUseCaseUngrouped(stageRepo stagepb.StageDomainServiceServer) *ListStagesUseCase {
	repositories := ListStagesRepositories{
		Stage: stageRepo,
	}

	services := ListStagesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListStagesUseCase(repositories, services)
}

// Execute performs the list stages operation
func (uc *ListStagesUseCase) Execute(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"stage", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.request_required", "Request is required for stages [DEFAULT]"))
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

// executeWithTransaction executes stage listing within a transaction
func (uc *ListStagesUseCase) executeWithTransaction(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	var result *stagepb.ListStagesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage.errors.list_failed", "Stage listing failed [DEFAULT]")
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

// executeCore contains the core business logic for listing stages
func (uc *ListStagesUseCase) executeCore(ctx context.Context, req *stagepb.ListStagesRequest) (*stagepb.ListStagesResponse, error) {
	// Delegate to repository
	return uc.repositories.Stage.ListStages(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ListStagesUseCase) applyBusinessLogic(req *stagepb.ListStagesRequest) *stagepb.ListStagesRequest {
	// Create enriched request with new proto fields
	enrichedReq := &stagepb.ListStagesRequest{
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
func (uc *ListStagesUseCase) validateBusinessRules(ctx context.Context, req *stagepb.ListStagesRequest) error {
	// Business rule: Pagination validation if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.limit_negative", "Limit cannot be negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.limit_too_large", "Limit cannot exceed 1000 [DEFAULT]"))
		}
	}

	return nil
}
