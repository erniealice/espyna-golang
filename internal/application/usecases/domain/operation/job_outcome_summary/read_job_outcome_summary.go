package job_outcome_summary

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type ReadJobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type ReadJobOutcomeSummaryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadJobOutcomeSummaryUseCase handles the business logic for reading job outcome summaries
type ReadJobOutcomeSummaryUseCase struct {
	repositories ReadJobOutcomeSummaryRepositories
	services     ReadJobOutcomeSummaryServices
}

// NewReadJobOutcomeSummaryUseCase creates a new ReadJobOutcomeSummaryUseCase
func NewReadJobOutcomeSummaryUseCase(
	repositories ReadJobOutcomeSummaryRepositories,
	services ReadJobOutcomeSummaryServices,
) *ReadJobOutcomeSummaryUseCase {
	return &ReadJobOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read job outcome summary operation
func (uc *ReadJobOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobOutcomeSummary,
		Action: entityid.ActionRead,
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadJobOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	var result *pb.ReadJobOutcomeSummaryResponse

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

// executeCore contains the core business logic for reading a job outcome summary
func (uc *ReadJobOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	resp, err := uc.repositories.JobOutcomeSummary.ReadJobOutcomeSummary(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.not_found", "[ERR-DEFAULT] Job outcome summary not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.not_found", "[ERR-DEFAULT] Job outcome summary not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadJobOutcomeSummaryUseCase) validateInput(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) error {
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
