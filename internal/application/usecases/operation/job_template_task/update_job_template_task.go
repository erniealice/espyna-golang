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

type UpdateJobTemplateTaskRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type UpdateJobTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateJobTemplateTaskUseCase handles the business logic for updating job template tasks
type UpdateJobTemplateTaskUseCase struct {
	repositories UpdateJobTemplateTaskRepositories
	services     UpdateJobTemplateTaskServices
}

// NewUpdateJobTemplateTaskUseCase creates a new UpdateJobTemplateTaskUseCase
func NewUpdateJobTemplateTaskUseCase(
	repositories UpdateJobTemplateTaskRepositories,
	services UpdateJobTemplateTaskServices,
) *UpdateJobTemplateTaskUseCase {
	return &UpdateJobTemplateTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update job template task operation
func (uc *UpdateJobTemplateTaskUseCase) Execute(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest) (*pb.UpdateJobTemplateTaskResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplateTask, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
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

// executeWithTransaction executes update within a transaction
func (uc *UpdateJobTemplateTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest, enrichedData *pb.JobTemplateTask) (*pb.UpdateJobTemplateTaskResponse, error) {
	var result *pb.UpdateJobTemplateTaskResponse

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

// executeCore contains the core business logic for updating a job template task
func (uc *UpdateJobTemplateTaskUseCase) executeCore(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest, enrichedData *pb.JobTemplateTask) (*pb.UpdateJobTemplateTaskResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.JobTemplateTask.ReadJobTemplateTask(ctx, &pb.ReadJobTemplateTaskRequest{
		Data: &pb.JobTemplateTask{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.not_found", "[ERR-DEFAULT] Job template task not found"))
	}

	resp, err := uc.repositories.JobTemplateTask.UpdateJobTemplateTask(ctx, &pb.UpdateJobTemplateTaskRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.update_failed", "[ERR-DEFAULT] Job template task update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateJobTemplateTaskUseCase) applyBusinessLogic(data *pb.JobTemplateTask) *pb.JobTemplateTask {
	now := time.Now()

	// Business logic: Update modification audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateInput validates the input request
func (uc *UpdateJobTemplateTaskUseCase) validateInput(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest) error {
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

// validateBusinessRules enforces business constraints
func (uc *UpdateJobTemplateTaskUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplateTask) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.data_required", "[ERR-DEFAULT] Job template task data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.id_required", "[ERR-DEFAULT] Job template task ID is required"))
	}
	// Validate Name only if provided (partial update support)
	if data.Name != "" && len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.name_too_long", "[ERR-DEFAULT] Job template task name is too long"))
	}

	return nil
}
