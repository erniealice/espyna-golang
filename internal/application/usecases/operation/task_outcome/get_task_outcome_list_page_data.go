package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type GetTaskOutcomeListPageDataRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type GetTaskOutcomeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetTaskOutcomeListPageDataUseCase handles the business logic for getting task outcome list page data
type GetTaskOutcomeListPageDataUseCase struct {
	repositories GetTaskOutcomeListPageDataRepositories
	services     GetTaskOutcomeListPageDataServices
}

// NewGetTaskOutcomeListPageDataUseCase creates a new GetTaskOutcomeListPageDataUseCase
func NewGetTaskOutcomeListPageDataUseCase(
	repositories GetTaskOutcomeListPageDataRepositories,
	services GetTaskOutcomeListPageDataServices,
) *GetTaskOutcomeListPageDataUseCase {
	return &GetTaskOutcomeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get task outcome list page data operation
func (uc *GetTaskOutcomeListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTaskOutcomeListPageDataRequest,
) (*pb.GetTaskOutcomeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcome, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetTaskOutcomeListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTaskOutcomeListPageDataRequest,
) (*pb.GetTaskOutcomeListPageDataResponse, error) {
	var result *pb.GetTaskOutcomeListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"task_outcome.errors.list_page_data_failed",
				"task outcome list page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting task outcome list page data
func (uc *GetTaskOutcomeListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTaskOutcomeListPageDataRequest,
) (*pb.GetTaskOutcomeListPageDataResponse, error) {
	resp, err := uc.repositories.TaskOutcome.GetTaskOutcomeListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.errors.list_page_data_failed",
			"failed to retrieve task outcome list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetTaskOutcomeListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTaskOutcomeListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.validation.request_required",
			"request is required",
		))
	}

	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetTaskOutcomeListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
