package job_template_task

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type GetJobTemplateTaskListPageDataRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type GetJobTemplateTaskListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobTemplateTaskListPageDataUseCase handles the business logic for getting job template task list page data
type GetJobTemplateTaskListPageDataUseCase struct {
	repositories GetJobTemplateTaskListPageDataRepositories
	services     GetJobTemplateTaskListPageDataServices
}

// NewGetJobTemplateTaskListPageDataUseCase creates a new GetJobTemplateTaskListPageDataUseCase
func NewGetJobTemplateTaskListPageDataUseCase(
	repositories GetJobTemplateTaskListPageDataRepositories,
	services GetJobTemplateTaskListPageDataServices,
) *GetJobTemplateTaskListPageDataUseCase {
	return &GetJobTemplateTaskListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template task list page data operation
func (uc *GetJobTemplateTaskListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) (*pb.GetJobTemplateTaskListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplateTask, ports.ActionList); err != nil {
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
func (uc *GetJobTemplateTaskListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) (*pb.GetJobTemplateTaskListPageDataResponse, error) {
	var result *pb.GetJobTemplateTaskListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"job_template_task.errors.list_page_data_failed",
				"job template task list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting job template task list page data
func (uc *GetJobTemplateTaskListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) (*pb.GetJobTemplateTaskListPageDataResponse, error) {
	resp, err := uc.repositories.JobTemplateTask.GetJobTemplateTaskListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.errors.list_page_data_failed",
			"failed to retrieve job template task list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetJobTemplateTaskListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.validation.request_required",
			"request is required",
		))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetJobTemplateTaskListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
