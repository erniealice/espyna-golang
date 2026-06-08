package phase_outcome_summary

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type GetByJobPhaseRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type GetByJobPhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetByJobPhaseUseCase handles the business logic for getting phase outcome summary by job phase
type GetByJobPhaseUseCase struct {
	repositories GetByJobPhaseRepositories
	services     GetByJobPhaseServices
}

// NewGetByJobPhaseUseCase creates a new GetByJobPhaseUseCase
func NewGetByJobPhaseUseCase(
	repositories GetByJobPhaseRepositories,
	services GetByJobPhaseServices,
) *GetByJobPhaseUseCase {
	return &GetByJobPhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get by job phase operation
func (uc *GetByJobPhaseUseCase) Execute(ctx context.Context, req *pb.GetPhaseOutcomeSummaryByJobPhaseRequest) (*pb.GetPhaseOutcomeSummaryByJobPhaseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PhaseOutcomeSummary, entityid.ActionRead); err != nil {
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

// executeWithTransaction executes get by job phase within a transaction
func (uc *GetByJobPhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.GetPhaseOutcomeSummaryByJobPhaseRequest) (*pb.GetPhaseOutcomeSummaryByJobPhaseResponse, error) {
	var result *pb.GetPhaseOutcomeSummaryByJobPhaseResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "phase_outcome_summary.errors.get_by_job_phase_failed", "get phase outcome summary by job phase failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting phase outcome summary by job phase
func (uc *GetByJobPhaseUseCase) executeCore(ctx context.Context, req *pb.GetPhaseOutcomeSummaryByJobPhaseRequest) (*pb.GetPhaseOutcomeSummaryByJobPhaseResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.GetByJobPhase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.get_by_job_phase_failed", "failed to get phase outcome summary by job phase: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetByJobPhaseUseCase) validateInput(ctx context.Context, req *pb.GetPhaseOutcomeSummaryByJobPhaseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.request_required", "request is required"))
	}
	if req.JobPhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.job_phase_id_required", "job phase ID is required"))
	}

	return nil
}
