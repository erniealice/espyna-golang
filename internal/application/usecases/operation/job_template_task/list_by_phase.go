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

type ListByPhaseRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type ListByPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByPhaseUseCase handles the business logic for listing tasks by phase
type ListByPhaseUseCase struct {
	repositories ListByPhaseRepositories
	services     ListByPhaseServices
}

// NewListByPhaseUseCase creates a new ListByPhaseUseCase
func NewListByPhaseUseCase(
	repositories ListByPhaseRepositories,
	services ListByPhaseServices,
) *ListByPhaseUseCase {
	return &ListByPhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by phase operation
func (uc *ListByPhaseUseCase) Execute(ctx context.Context, req *pb.ListJobTemplateTasksByPhaseRequest) (*pb.ListJobTemplateTasksByPhaseResponse, error) {
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

// executeWithTransaction executes list by phase within a transaction
func (uc *ListByPhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.ListJobTemplateTasksByPhaseRequest) (*pb.ListJobTemplateTasksByPhaseResponse, error) {
	var result *pb.ListJobTemplateTasksByPhaseResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "job_template_task.errors.list_by_phase_failed", "job template task listing by phase failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing tasks by phase
func (uc *ListByPhaseUseCase) executeCore(ctx context.Context, req *pb.ListJobTemplateTasksByPhaseRequest) (*pb.ListJobTemplateTasksByPhaseResponse, error) {
	resp, err := uc.repositories.JobTemplateTask.ListByPhase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.errors.list_by_phase_failed", "failed to list job template tasks by phase: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByPhaseUseCase) validateInput(ctx context.Context, req *pb.ListJobTemplateTasksByPhaseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.request_required", "request is required"))
	}
	if req.JobTemplatePhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_task.validation.phase_id_required", "job template phase ID is required"))
	}

	return nil
}
