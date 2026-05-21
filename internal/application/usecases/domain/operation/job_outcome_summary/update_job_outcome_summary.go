package job_outcome_summary

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type UpdateJobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type UpdateJobOutcomeSummaryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateJobOutcomeSummaryUseCase handles the business logic for updating job outcome summaries
type UpdateJobOutcomeSummaryUseCase struct {
	repositories UpdateJobOutcomeSummaryRepositories
	services     UpdateJobOutcomeSummaryServices
}

// NewUpdateJobOutcomeSummaryUseCase creates a new UpdateJobOutcomeSummaryUseCase
func NewUpdateJobOutcomeSummaryUseCase(
	repositories UpdateJobOutcomeSummaryRepositories,
	services UpdateJobOutcomeSummaryServices,
) *UpdateJobOutcomeSummaryUseCase {
	return &UpdateJobOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update job outcome summary operation
func (uc *UpdateJobOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityJobOutcomeSummary, ports.ActionUpdate); err != nil {
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
func (uc *UpdateJobOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest, enrichedData *pb.JobOutcomeSummary) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	var result *pb.UpdateJobOutcomeSummaryResponse

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

// executeCore contains the core business logic for updating a job outcome summary
func (uc *UpdateJobOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest, enrichedData *pb.JobOutcomeSummary) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	_, err := uc.repositories.JobOutcomeSummary.ReadJobOutcomeSummary(ctx, &pb.ReadJobOutcomeSummaryRequest{
		Data: &pb.JobOutcomeSummary{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.not_found", "[ERR-DEFAULT] Job outcome summary not found"))
	}

	resp, err := uc.repositories.JobOutcomeSummary.UpdateJobOutcomeSummary(ctx, &pb.UpdateJobOutcomeSummaryRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.update_failed", "[ERR-DEFAULT] Job outcome summary update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateJobOutcomeSummaryUseCase) applyBusinessLogic(data *pb.JobOutcomeSummary) *pb.JobOutcomeSummary {
	now := time.Now()
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return data
}

// validateInput validates the input request
func (uc *UpdateJobOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.data_required", "[ERR-DEFAULT] Job outcome summary data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.id_required", "[ERR-DEFAULT] Job outcome summary ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateJobOutcomeSummaryUseCase) validateBusinessRules(ctx context.Context, data *pb.JobOutcomeSummary) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.data_required", "[ERR-DEFAULT] Job outcome summary data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.id_required", "[ERR-DEFAULT] Job outcome summary ID is required"))
	}
	return nil
}
