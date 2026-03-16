package phase_outcome_summary

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type ReadPhaseOutcomeSummaryRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type ReadPhaseOutcomeSummaryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPhaseOutcomeSummaryUseCase handles the business logic for reading phase outcome summaries
type ReadPhaseOutcomeSummaryUseCase struct {
	repositories ReadPhaseOutcomeSummaryRepositories
	services     ReadPhaseOutcomeSummaryServices
}

// NewReadPhaseOutcomeSummaryUseCase creates a new ReadPhaseOutcomeSummaryUseCase
func NewReadPhaseOutcomeSummaryUseCase(
	repositories ReadPhaseOutcomeSummaryRepositories,
	services ReadPhaseOutcomeSummaryServices,
) *ReadPhaseOutcomeSummaryUseCase {
	return &ReadPhaseOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read phase outcome summary operation
func (uc *ReadPhaseOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) (*pb.ReadPhaseOutcomeSummaryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPhaseOutcomeSummary, ports.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadPhaseOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) (*pb.ReadPhaseOutcomeSummaryResponse, error) {
	var result *pb.ReadPhaseOutcomeSummaryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for reading a phase outcome summary
func (uc *ReadPhaseOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) (*pb.ReadPhaseOutcomeSummaryResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.ReadPhaseOutcomeSummary(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.errors.not_found", "[ERR-DEFAULT] Phase outcome summary not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.errors.not_found", "[ERR-DEFAULT] Phase outcome summary not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPhaseOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.data_required", "[ERR-DEFAULT] Phase outcome summary data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "phase_outcome_summary.validation.id_required", "[ERR-DEFAULT] Phase outcome summary ID is required"))
	}
	return nil
}
