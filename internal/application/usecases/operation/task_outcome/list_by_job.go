package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ListByJobRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ListByJobServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByJobUseCase handles the business logic for listing task outcomes by job
type ListByJobUseCase struct {
	repositories ListByJobRepositories
	services     ListByJobServices
}

// NewListByJobUseCase creates a new ListByJobUseCase
func NewListByJobUseCase(
	repositories ListByJobRepositories,
	services ListByJobServices,
) *ListByJobUseCase {
	return &ListByJobUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by job operation
func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomesByJobRequest) (*pb.ListTaskOutcomesByJobResponse, error) {
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

// executeWithTransaction executes list by job within a transaction
func (uc *ListByJobUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomesByJobRequest) (*pb.ListTaskOutcomesByJobResponse, error) {
	var result *pb.ListTaskOutcomesByJobResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "task_outcome.errors.list_by_job_failed", "task outcome listing by job failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing task outcomes by job
func (uc *ListByJobUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomesByJobRequest) (*pb.ListTaskOutcomesByJobResponse, error) {
	resp, err := uc.repositories.TaskOutcome.ListByJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.list_by_job_failed", "failed to list task outcomes by job: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByJobUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomesByJobRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.request_required", "request is required"))
	}
	if req.JobId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.job_id_required", "job ID is required"))
	}

	return nil
}
