package job_template_task

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type GetJobTemplateTaskItemPageDataRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type GetJobTemplateTaskItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetJobTemplateTaskItemPageDataUseCase handles the business logic for getting job template task item page data
type GetJobTemplateTaskItemPageDataUseCase struct {
	repositories GetJobTemplateTaskItemPageDataRepositories
	services     GetJobTemplateTaskItemPageDataServices
}

// NewGetJobTemplateTaskItemPageDataUseCase creates a new GetJobTemplateTaskItemPageDataUseCase
func NewGetJobTemplateTaskItemPageDataUseCase(
	repositories GetJobTemplateTaskItemPageDataRepositories,
	services GetJobTemplateTaskItemPageDataServices,
) *GetJobTemplateTaskItemPageDataUseCase {
	return &GetJobTemplateTaskItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template task item page data operation
func (uc *GetJobTemplateTaskItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobTemplateTaskItemPageDataRequest,
) (*pb.GetJobTemplateTaskItemPageDataResponse, error) {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetJobTemplateTaskItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobTemplateTaskItemPageDataRequest,
) (*pb.GetJobTemplateTaskItemPageDataResponse, error) {
	var result *pb.GetJobTemplateTaskItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"job_template_task.errors.item_page_data_failed",
				"job template task item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting job template task item page data
func (uc *GetJobTemplateTaskItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobTemplateTaskItemPageDataRequest,
) (*pb.GetJobTemplateTaskItemPageDataResponse, error) {
	// Retrieve the entity via Read
	readReq := &pb.ReadJobTemplateTaskRequest{
		Data: &pb.JobTemplateTask{
			Id: req.JobTemplateTaskId,
		},
	}

	readResp, err := uc.repositories.JobTemplateTask.ReadJobTemplateTask(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.errors.read_failed",
			"failed to retrieve job template task: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.errors.not_found",
			"job template task not found",
		))
	}

	task := readResp.Data[0]

	return &pb.GetJobTemplateTaskItemPageDataResponse{
		JobTemplateTask: task,
		Success:         true,
	}, nil
}

// validateInput validates the input request
func (uc *GetJobTemplateTaskItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobTemplateTaskItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.validation.request_required",
			"request is required",
		))
	}

	if req.JobTemplateTaskId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"job_template_task.validation.id_required",
			"job template task ID is required",
		))
	}

	return nil
}
