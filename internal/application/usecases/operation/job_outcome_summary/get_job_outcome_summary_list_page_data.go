package job_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type GetJobOutcomeSummaryListPageDataRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type GetJobOutcomeSummaryListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobOutcomeSummaryListPageDataUseCase handles the business logic for getting job outcome summary list page data
type GetJobOutcomeSummaryListPageDataUseCase struct {
	repositories GetJobOutcomeSummaryListPageDataRepositories
	services     GetJobOutcomeSummaryListPageDataServices
}

// NewGetJobOutcomeSummaryListPageDataUseCase creates a new GetJobOutcomeSummaryListPageDataUseCase
func NewGetJobOutcomeSummaryListPageDataUseCase(
	repositories GetJobOutcomeSummaryListPageDataRepositories,
	services GetJobOutcomeSummaryListPageDataServices,
) *GetJobOutcomeSummaryListPageDataUseCase {
	return &GetJobOutcomeSummaryListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job outcome summary list page data operation
func (uc *GetJobOutcomeSummaryListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobOutcomeSummary, ports.ActionList); err != nil {
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
func (uc *GetJobOutcomeSummaryListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	var result *pb.GetJobOutcomeSummaryListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"job_outcome_summary.errors.list_page_data_failed",
				"job outcome summary list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting job outcome summary list page data
func (uc *GetJobOutcomeSummaryListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	resp, err := uc.repositories.JobOutcomeSummary.GetJobOutcomeSummaryListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.errors.list_page_data_failed",
			"failed to retrieve job outcome summary list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetJobOutcomeSummaryListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.validation.request_required",
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
func (uc *GetJobOutcomeSummaryListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_outcome_summary.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
