package job_template_task

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type ReadJobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type ReadJobTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadJobTemplateTaskUseCase handles the business logic for reading job template tasks
type ReadJobTemplateTaskUseCase struct {
	repositories ReadJobTemplateTaskRepositories
	services     ReadJobTemplateTaskServices
}

// NewReadJobTemplateTaskUseCase creates a new ReadJobTemplateTaskUseCase
func NewReadJobTemplateTaskUseCase(
	repositories ReadJobTemplateTaskRepositories,
	services ReadJobTemplateTaskServices,
) *ReadJobTemplateTaskUseCase {
	return &ReadJobTemplateTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read job template task operation
func (uc *ReadJobTemplateTaskUseCase) Execute(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) (*pb.ReadJobTemplateTaskResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplateTask, ports.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadJobTemplateTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) (*pb.ReadJobTemplateTaskResponse, error) {
	var result *pb.ReadJobTemplateTaskResponse

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

// executeCore contains the core business logic for reading a job template task
func (uc *ReadJobTemplateTaskUseCase) executeCore(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) (*pb.ReadJobTemplateTaskResponse, error) {
	resp, err := uc.repositories.JobTemplateTask.ReadJobTemplateTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.not_found", "[ERR-DEFAULT] Job template task not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.not_found", "[ERR-DEFAULT] Job template task not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadJobTemplateTaskUseCase) validateInput(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) error {
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
