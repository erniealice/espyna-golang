package phase_outcome_summary

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type DeletePhaseOutcomeSummaryRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type DeletePhaseOutcomeSummaryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeletePhaseOutcomeSummaryUseCase handles the business logic for deleting phase outcome summaries
type DeletePhaseOutcomeSummaryUseCase struct {
	repositories DeletePhaseOutcomeSummaryRepositories
	services     DeletePhaseOutcomeSummaryServices
}

// NewDeletePhaseOutcomeSummaryUseCase creates a new DeletePhaseOutcomeSummaryUseCase
func NewDeletePhaseOutcomeSummaryUseCase(
	repositories DeletePhaseOutcomeSummaryRepositories,
	services DeletePhaseOutcomeSummaryServices,
) *DeletePhaseOutcomeSummaryUseCase {
	return &DeletePhaseOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete phase outcome summary operation
func (uc *DeletePhaseOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) (*pb.DeletePhaseOutcomeSummaryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PhaseOutcomeSummary,
		Action: entityid.ActionDelete,
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeletePhaseOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) (*pb.DeletePhaseOutcomeSummaryResponse, error) {
	var result *pb.DeletePhaseOutcomeSummaryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for deleting a phase outcome summary
func (uc *DeletePhaseOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) (*pb.DeletePhaseOutcomeSummaryResponse, error) {
	_, err := uc.repositories.PhaseOutcomeSummary.ReadPhaseOutcomeSummary(ctx, &pb.ReadPhaseOutcomeSummaryRequest{
		Data: &pb.PhaseOutcomeSummary{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.not_found", "[ERR-DEFAULT] Phase outcome summary not found"))
	}

	resp, err := uc.repositories.PhaseOutcomeSummary.DeletePhaseOutcomeSummary(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.deletion_failed", "[ERR-DEFAULT] Phase outcome summary deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeletePhaseOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.data_required", "[ERR-DEFAULT] Phase outcome summary data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.id_required", "[ERR-DEFAULT] Phase outcome summary ID is required"))
	}
	return nil
}
