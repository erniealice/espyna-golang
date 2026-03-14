package job_template_task

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type DeleteJobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type DeleteJobTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteJobTemplateTaskUseCase handles the business logic for deleting job template tasks
type DeleteJobTemplateTaskUseCase struct {
	repositories DeleteJobTemplateTaskRepositories
	services     DeleteJobTemplateTaskServices
}

// NewDeleteJobTemplateTaskUseCase creates a new DeleteJobTemplateTaskUseCase
func NewDeleteJobTemplateTaskUseCase(
	repositories DeleteJobTemplateTaskRepositories,
	services DeleteJobTemplateTaskServices,
) *DeleteJobTemplateTaskUseCase {
	return &DeleteJobTemplateTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete job template task operation
func (uc *DeleteJobTemplateTaskUseCase) Execute(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) (*pb.DeleteJobTemplateTaskResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplateTask, ports.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteJobTemplateTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) (*pb.DeleteJobTemplateTaskResponse, error) {
	var result *pb.DeleteJobTemplateTaskResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for deleting a job template task
func (uc *DeleteJobTemplateTaskUseCase) executeCore(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) (*pb.DeleteJobTemplateTaskResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.JobTemplateTask.ReadJobTemplateTask(ctx, &pb.ReadJobTemplateTaskRequest{
		Data: &pb.JobTemplateTask{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.not_found", "[ERR-DEFAULT] Job template task not found"))
	}

	resp, err := uc.repositories.JobTemplateTask.DeleteJobTemplateTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.deletion_failed", "[ERR-DEFAULT] Job template task deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteJobTemplateTaskUseCase) validateInput(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.data_required", "[ERR-DEFAULT] Job template task data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.id_required", "[ERR-DEFAULT] Job template task ID is required"))
	}
	return nil
}
