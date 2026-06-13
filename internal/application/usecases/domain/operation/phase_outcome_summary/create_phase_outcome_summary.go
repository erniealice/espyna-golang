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

type CreatePhaseOutcomeSummaryRepositories struct {
	PhaseOutcomeSummary pb.PhaseOutcomeSummaryDomainServiceServer
}

type CreatePhaseOutcomeSummaryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreatePhaseOutcomeSummaryUseCase handles the business logic for creating phase outcome summaries
type CreatePhaseOutcomeSummaryUseCase struct {
	repositories CreatePhaseOutcomeSummaryRepositories
	services     CreatePhaseOutcomeSummaryServices
}

// NewCreatePhaseOutcomeSummaryUseCase creates a new CreatePhaseOutcomeSummaryUseCase
func NewCreatePhaseOutcomeSummaryUseCase(
	repositories CreatePhaseOutcomeSummaryRepositories,
	services CreatePhaseOutcomeSummaryServices,
) *CreatePhaseOutcomeSummaryUseCase {
	return &CreatePhaseOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create phase outcome summary operation
func (uc *CreatePhaseOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.CreatePhaseOutcomeSummaryRequest) (*pb.CreatePhaseOutcomeSummaryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PhaseOutcomeSummary,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.data_required", "[ERR-DEFAULT] Phase outcome summary data is required"))
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

// executeWithTransaction executes creation within a transaction
func (uc *CreatePhaseOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.CreatePhaseOutcomeSummaryRequest, enrichedData *pb.PhaseOutcomeSummary) (*pb.CreatePhaseOutcomeSummaryResponse, error) {
	var result *pb.CreatePhaseOutcomeSummaryResponse
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

// executeCore contains the core business logic for creating a phase outcome summary
func (uc *CreatePhaseOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.CreatePhaseOutcomeSummaryRequest, enrichedData *pb.PhaseOutcomeSummary) (*pb.CreatePhaseOutcomeSummaryResponse, error) {
	resp, err := uc.repositories.PhaseOutcomeSummary.CreatePhaseOutcomeSummary(ctx, &pb.CreatePhaseOutcomeSummaryRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.errors.creation_failed", "[ERR-DEFAULT] Phase outcome summary creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreatePhaseOutcomeSummaryUseCase) applyBusinessLogic(data *pb.PhaseOutcomeSummary) *pb.PhaseOutcomeSummary {
	now := time.Now()

	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreatePhaseOutcomeSummaryUseCase) validateBusinessRules(ctx context.Context, data *pb.PhaseOutcomeSummary) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.data_required", "[ERR-DEFAULT] Phase outcome summary data is required"))
	}
	if data.JobPhaseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "phase_outcome_summary.validation.job_phase_id_required", "[ERR-DEFAULT] Job phase ID is required"))
	}

	return nil
}
