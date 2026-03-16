package job_outcome_summary

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type DeleteJobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type DeleteJobOutcomeSummaryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteJobOutcomeSummaryUseCase handles the business logic for deleting job outcome summaries
type DeleteJobOutcomeSummaryUseCase struct {
	repositories DeleteJobOutcomeSummaryRepositories
	services     DeleteJobOutcomeSummaryServices
}

// NewDeleteJobOutcomeSummaryUseCase creates a new DeleteJobOutcomeSummaryUseCase
func NewDeleteJobOutcomeSummaryUseCase(
	repositories DeleteJobOutcomeSummaryRepositories,
	services DeleteJobOutcomeSummaryServices,
) *DeleteJobOutcomeSummaryUseCase {
	return &DeleteJobOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete job outcome summary operation
func (uc *DeleteJobOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobOutcomeSummary, ports.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteJobOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	var result *pb.DeleteJobOutcomeSummaryResponse

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

// executeCore contains the core business logic for deleting a job outcome summary
func (uc *DeleteJobOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	_, err := uc.repositories.JobOutcomeSummary.ReadJobOutcomeSummary(ctx, &pb.ReadJobOutcomeSummaryRequest{
		Data: &pb.JobOutcomeSummary{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.errors.not_found", "[ERR-DEFAULT] Job outcome summary not found"))
	}

	resp, err := uc.repositories.JobOutcomeSummary.DeleteJobOutcomeSummary(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.errors.deletion_failed", "[ERR-DEFAULT] Job outcome summary deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteJobOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.validation.data_required", "[ERR-DEFAULT] Job outcome summary data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_outcome_summary.validation.id_required", "[ERR-DEFAULT] Job outcome summary ID is required"))
	}
	return nil
}
