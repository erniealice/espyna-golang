package job_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type GetByJobRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type GetByJobServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetByJobUseCase handles the business logic for getting job outcome summary by job
type GetByJobUseCase struct {
	repositories GetByJobRepositories
	services     GetByJobServices
}

// NewGetByJobUseCase creates a new GetByJobUseCase
func NewGetByJobUseCase(
	repositories GetByJobRepositories,
	services GetByJobServices,
) *GetByJobUseCase {
	return &GetByJobUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get by job operation
func (uc *GetByJobUseCase) Execute(ctx context.Context, req *pb.GetJobOutcomeSummaryByJobRequest) (*pb.GetJobOutcomeSummaryByJobResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.JobOutcomeSummary, entityid.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes get by job within a transaction
func (uc *GetByJobUseCase) executeWithTransaction(ctx context.Context, req *pb.GetJobOutcomeSummaryByJobRequest) (*pb.GetJobOutcomeSummaryByJobResponse, error) {
	var result *pb.GetJobOutcomeSummaryByJobResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "job_outcome_summary.errors.get_by_job_failed", "get job outcome summary by job failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting job outcome summary by job
func (uc *GetByJobUseCase) executeCore(ctx context.Context, req *pb.GetJobOutcomeSummaryByJobRequest) (*pb.GetJobOutcomeSummaryByJobResponse, error) {
	resp, err := uc.repositories.JobOutcomeSummary.GetByJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.get_by_job_failed", "failed to get job outcome summary by job: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetByJobUseCase) validateInput(ctx context.Context, req *pb.GetJobOutcomeSummaryByJobRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.request_required", "request is required"))
	}
	if req.JobId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.job_id_required", "job ID is required"))
	}

	return nil
}
