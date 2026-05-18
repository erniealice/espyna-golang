package phase_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type ListByJobRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type ListByJobServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByJobUseCase handles the business logic for listing phase outcome summaries by job
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
func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListPhaseOutcomeSummarysByJobRequest) (*pb.ListPhaseOutcomeSummarysByJobResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPhaseOutcomeSummary, ports.ActionList); err != nil {
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
func (uc *ListByJobUseCase) executeWithTransaction(ctx context.Context, req *pb.ListPhaseOutcomeSummarysByJobRequest) (*pb.ListPhaseOutcomeSummarysByJobResponse, error) {
	var result *pb.ListPhaseOutcomeSummarysByJobResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "phase_outcome_summary.errors.list_by_job_failed", "phase outcome summary listing by job failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing phase outcome summaries by job
func (uc *ListByJobUseCase) executeCore(ctx context.Context, req *pb.ListPhaseOutcomeSummarysByJobRequest) (*pb.ListPhaseOutcomeSummarysByJobResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.ListByJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.errors.list_by_job_failed", "failed to list phase outcome summaries by job: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByJobUseCase) validateInput(ctx context.Context, req *pb.ListPhaseOutcomeSummarysByJobRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.request_required", "request is required"))
	}
	if req.JobId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.job_id_required", "job ID is required"))
	}

	return nil
}
