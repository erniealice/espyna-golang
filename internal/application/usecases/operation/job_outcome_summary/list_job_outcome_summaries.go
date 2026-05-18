package job_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type ListJobOutcomeSummariesRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type ListJobOutcomeSummariesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobOutcomeSummary, ports.ActionList); err != nil {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListJobOutcomeSummariesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	var result *pb.ListJobOutcomeSummarysResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "job_outcome_summary.errors.list_failed", "job outcome summary listing failed: %w"), err)
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
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.errors.list_failed", "job outcome summary listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListJobOutcomeSummariesUseCase) validateInput(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.validation.request_required", "request is required"))
	}

	return nil
}
