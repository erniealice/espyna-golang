package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ListByJobPhaseRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ListByJobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByJobPhaseUseCase handles the business logic for listing outcomes by job phase
type ListByJobPhaseUseCase struct {
	repositories ListByJobPhaseRepositories
	services     ListByJobPhaseServices
}

// NewListByJobPhaseUseCase creates a new ListByJobPhaseUseCase
func NewListByJobPhaseUseCase(
	repositories ListByJobPhaseRepositories,
	services ListByJobPhaseServices,
) *ListByJobPhaseUseCase {
	return &ListByJobPhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by job phase operation
func (uc *ListByJobPhaseUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomesByJobPhaseRequest) (*pb.ListTaskOutcomesByJobPhaseResponse, error) {
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

// executeWithTransaction executes list by job phase within a transaction
func (uc *ListByJobPhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomesByJobPhaseRequest) (*pb.ListTaskOutcomesByJobPhaseResponse, error) {
	var result *pb.ListTaskOutcomesByJobPhaseResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "task_outcome.errors.list_by_job_phase_failed", "task outcome listing by job phase failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing outcomes by job phase
func (uc *ListByJobPhaseUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomesByJobPhaseRequest) (*pb.ListTaskOutcomesByJobPhaseResponse, error) {
	resp, err := uc.repositories.TaskOutcome.ListByJobPhase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.list_by_job_phase_failed", "failed to list task outcomes by job phase: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByJobPhaseUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomesByJobPhaseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.request_required", "request is required"))
	}
	if req.JobPhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.job_phase_id_required", "job phase ID is required"))
	}

	return nil
}
