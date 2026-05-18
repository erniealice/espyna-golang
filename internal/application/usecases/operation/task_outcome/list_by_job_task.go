package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ListByJobTaskRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ListByJobTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByJobTaskUseCase handles the business logic for listing outcomes by job task
type ListByJobTaskUseCase struct {
	repositories ListByJobTaskRepositories
	services     ListByJobTaskServices
}

// NewListByJobTaskUseCase creates a new ListByJobTaskUseCase
func NewListByJobTaskUseCase(
	repositories ListByJobTaskRepositories,
	services ListByJobTaskServices,
) *ListByJobTaskUseCase {
	return &ListByJobTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by job task operation
func (uc *ListByJobTaskUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomesByJobTaskRequest) (*pb.ListTaskOutcomesByJobTaskResponse, error) {
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

// executeWithTransaction executes list by job task within a transaction
func (uc *ListByJobTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomesByJobTaskRequest) (*pb.ListTaskOutcomesByJobTaskResponse, error) {
	var result *pb.ListTaskOutcomesByJobTaskResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "task_outcome.errors.list_by_job_task_failed", "task outcome listing by job task failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing outcomes by job task
func (uc *ListByJobTaskUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomesByJobTaskRequest) (*pb.ListTaskOutcomesByJobTaskResponse, error) {
	resp, err := uc.repositories.TaskOutcome.ListByJobTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.list_by_job_task_failed", "failed to list task outcomes by job task: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByJobTaskUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomesByJobTaskRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.request_required", "request is required"))
	}
	if req.JobTaskId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.job_task_id_required", "job task ID is required"))
	}

	return nil
}
