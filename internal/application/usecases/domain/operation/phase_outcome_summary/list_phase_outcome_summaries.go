package phase_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type ListPhaseOutcomeSummariesRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type ListPhaseOutcomeSummariesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PhaseOutcomeSummary,
		Action: entityid.ActionList,
	}); err != nil {
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
func (uc *ListPhaseOutcomeSummariesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
	var result *pb.ListPhaseOutcomeSummarysResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "phase_outcome_summary.errors.list_failed", "phase outcome summary listing failed: %w"), err)
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
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.list_failed", "phase outcome summary listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListPhaseOutcomeSummariesUseCase) validateInput(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.request_required", "request is required"))
	}

	return nil
}
