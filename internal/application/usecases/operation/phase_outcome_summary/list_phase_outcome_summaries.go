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

type ListPhaseOutcomeSummariesRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type ListPhaseOutcomeSummariesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPhaseOutcomeSummariesUseCase handles the business logic for listing phase outcome summaries
type ListPhaseOutcomeSummariesUseCase struct {
	repositories ListPhaseOutcomeSummariesRepositories
	services     ListPhaseOutcomeSummariesServices
}

// NewListPhaseOutcomeSummariesUseCase creates a new ListPhaseOutcomeSummariesUseCase
func NewListPhaseOutcomeSummariesUseCase(
	repositories ListPhaseOutcomeSummariesRepositories,
	services ListPhaseOutcomeSummariesServices,
) *ListPhaseOutcomeSummariesUseCase {
	return &ListPhaseOutcomeSummariesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list phase outcome summaries operation
func (uc *ListPhaseOutcomeSummariesUseCase) Execute(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListPhaseOutcomeSummariesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
	var result *pb.ListPhaseOutcomeSummarysResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "phase_outcome_summary.errors.list_failed", "phase outcome summary listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing phase outcome summaries
func (uc *ListPhaseOutcomeSummariesUseCase) executeCore(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.ListPhaseOutcomeSummarys(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.errors.list_failed", "phase outcome summary listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListPhaseOutcomeSummariesUseCase) validateInput(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.request_required", "request is required"))
	}

	return nil
}
