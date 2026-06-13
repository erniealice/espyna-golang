package phase_outcome_summary

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

type UpdatePhaseOutcomeSummaryRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type UpdatePhaseOutcomeSummaryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdatePhaseOutcomeSummaryUseCase handles the business logic for updating phase outcome summaries
type UpdatePhaseOutcomeSummaryUseCase struct {
	repositories UpdatePhaseOutcomeSummaryRepositories
	services     UpdatePhaseOutcomeSummaryServices
}

// NewUpdatePhaseOutcomeSummaryUseCase creates a new UpdatePhaseOutcomeSummaryUseCase
func NewUpdatePhaseOutcomeSummaryUseCase(
	repositories UpdatePhaseOutcomeSummaryRepositories,
	services UpdatePhaseOutcomeSummaryServices,
) *UpdatePhaseOutcomeSummaryUseCase {
	return &UpdatePhaseOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update phase outcome summary operation
func (uc *UpdatePhaseOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest) (*pb.UpdatePhaseOutcomeSummaryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PhaseOutcomeSummary,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes update within a transaction
func (uc *UpdatePhaseOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest, enrichedData *pb.PhaseOutcomeSummary) (*pb.UpdatePhaseOutcomeSummaryResponse, error) {
	var result *pb.UpdatePhaseOutcomeSummaryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedData)
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

// executeCore contains the core business logic for updating a phase outcome summary
func (uc *UpdatePhaseOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest, enrichedData *pb.PhaseOutcomeSummary) (*pb.UpdatePhaseOutcomeSummaryResponse, error) {
	_, err := uc.repositories.PhaseOutcomeSummary.ReadPhaseOutcomeSummary(ctx, &pb.ReadPhaseOutcomeSummaryRequest{
		Data: &pb.PhaseOutcomeSummary{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.not_found", "[ERR-DEFAULT] Phase outcome summary not found"))
	}

	resp, err := uc.repositories.PhaseOutcomeSummary.UpdatePhaseOutcomeSummary(ctx, &pb.UpdatePhaseOutcomeSummaryRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.update_failed", "[ERR-DEFAULT] Phase outcome summary update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdatePhaseOutcomeSummaryUseCase) applyBusinessLogic(data *pb.PhaseOutcomeSummary) *pb.PhaseOutcomeSummary {
	now := time.Now()
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return data
}

// validateInput validates the input request
func (uc *UpdatePhaseOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest) error {
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

// validateBusinessRules enforces business constraints
func (uc *UpdatePhaseOutcomeSummaryUseCase) validateBusinessRules(ctx context.Context, data *pb.PhaseOutcomeSummary) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.data_required", "[ERR-DEFAULT] Phase outcome summary data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.id_required", "[ERR-DEFAULT] Phase outcome summary ID is required"))
	}
	return nil
}
