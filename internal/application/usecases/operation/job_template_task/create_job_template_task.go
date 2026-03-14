package job_template_task

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type CreateJobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type CreateJobTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateJobTemplateTaskUseCase handles the business logic for creating job template tasks
type CreateJobTemplateTaskUseCase struct {
	repositories CreateJobTemplateTaskRepositories
	services     CreateJobTemplateTaskServices
}

// NewCreateJobTemplateTaskUseCase creates a new CreateJobTemplateTaskUseCase
func NewCreateJobTemplateTaskUseCase(
	repositories CreateJobTemplateTaskRepositories,
	services CreateJobTemplateTaskServices,
) *CreateJobTemplateTaskUseCase {
	return &CreateJobTemplateTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create job template task operation
func (uc *CreateJobTemplateTaskUseCase) Execute(ctx context.Context, req *pb.CreateJobTemplateTaskRequest) (*pb.CreateJobTemplateTaskResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplateTask, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.data_required", "[ERR-DEFAULT] Job template task data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateJobTemplateTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateJobTemplateTaskRequest, enrichedData *pb.JobTemplateTask) (*pb.CreateJobTemplateTaskResponse, error) {
	var result *pb.CreateJobTemplateTaskResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedData)
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

// executeCore contains the core business logic for creating a job template task
func (uc *CreateJobTemplateTaskUseCase) executeCore(ctx context.Context, req *pb.CreateJobTemplateTaskRequest, enrichedData *pb.JobTemplateTask) (*pb.CreateJobTemplateTaskResponse, error) {
	resp, err := uc.repositories.JobTemplateTask.CreateJobTemplateTask(ctx, &pb.CreateJobTemplateTaskRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.creation_failed", "[ERR-DEFAULT] Job template task creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateJobTemplateTaskUseCase) applyBusinessLogic(data *pb.JobTemplateTask) *pb.JobTemplateTask {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new tasks
	data.Active = true

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateJobTemplateTaskUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplateTask) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.data_required", "[ERR-DEFAULT] Job template task data is required"))
	}
	if data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.name_required", "[ERR-DEFAULT] Job template task name is required"))
	}
	if data.JobTemplatePhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.phase_id_required", "[ERR-DEFAULT] Job template phase ID is required"))
	}
	if len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.name_too_long", "[ERR-DEFAULT] Job template task name is too long"))
	}

	return nil
}
