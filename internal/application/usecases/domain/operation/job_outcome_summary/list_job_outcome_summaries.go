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

type ListJobOutcomeSummariesRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type ListJobOutcomeSummariesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListJobOutcomeSummariesUseCase handles the business logic for listing job outcome summaries
type ListJobOutcomeSummariesUseCase struct {
	repositories ListJobOutcomeSummariesRepositories
	services     ListJobOutcomeSummariesServices
}

// NewListJobOutcomeSummariesUseCase creates a new ListJobOutcomeSummariesUseCase
func NewListJobOutcomeSummariesUseCase(
	repositories ListJobOutcomeSummariesRepositories,
	services ListJobOutcomeSummariesServices,
) *ListJobOutcomeSummariesUseCase {
	return &ListJobOutcomeSummariesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list job outcome summaries operation
func (uc *ListJobOutcomeSummariesUseCase) Execute(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.JobOutcomeSummary, entityid.ActionList); err != nil {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListJobOutcomeSummariesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	var result *pb.ListJobOutcomeSummarysResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "job_outcome_summary.errors.list_failed", "job outcome summary listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing job outcome summaries
func (uc *ListJobOutcomeSummariesUseCase) executeCore(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	resp, err := uc.repositories.JobOutcomeSummary.ListJobOutcomeSummarys(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.list_failed", "job outcome summary listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListJobOutcomeSummariesUseCase) validateInput(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.request_required", "request is required"))
	}

	return nil
}
