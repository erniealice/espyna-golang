package job_outcome_summary

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

type CreateJobOutcomeSummaryRepositories struct {
	JobOutcomeSummary pb.JobOutcomeSummaryDomainServiceServer
}

type CreateJobOutcomeSummaryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateJobOutcomeSummaryUseCase handles the business logic for creating job outcome summaries
type CreateJobOutcomeSummaryUseCase struct {
	repositories CreateJobOutcomeSummaryRepositories
	services     CreateJobOutcomeSummaryServices
}

// NewCreateJobOutcomeSummaryUseCase creates a new CreateJobOutcomeSummaryUseCase
func NewCreateJobOutcomeSummaryUseCase(
	repositories CreateJobOutcomeSummaryRepositories,
	services CreateJobOutcomeSummaryServices,
) *CreateJobOutcomeSummaryUseCase {
	return &CreateJobOutcomeSummaryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create job outcome summary operation
func (uc *CreateJobOutcomeSummaryUseCase) Execute(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest) (*pb.CreateJobOutcomeSummaryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobOutcomeSummary,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.data_required", "[ERR-DEFAULT] Job outcome summary data is required"))
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
func (uc *CreateJobOutcomeSummaryUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest, enrichedData *pb.JobOutcomeSummary) (*pb.CreateJobOutcomeSummaryResponse, error) {
	var result *pb.CreateJobOutcomeSummaryResponse
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

// executeCore contains the core business logic for creating a job outcome summary
func (uc *CreateJobOutcomeSummaryUseCase) executeCore(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest, enrichedData *pb.JobOutcomeSummary) (*pb.CreateJobOutcomeSummaryResponse, error) {
	resp, err := uc.repositories.JobOutcomeSummary.CreateJobOutcomeSummary(ctx, &pb.CreateJobOutcomeSummaryRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.errors.creation_failed", "[ERR-DEFAULT] Job outcome summary creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateJobOutcomeSummaryUseCase) applyBusinessLogic(data *pb.JobOutcomeSummary) *pb.JobOutcomeSummary {
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
func (uc *CreateJobOutcomeSummaryUseCase) validateBusinessRules(ctx context.Context, data *pb.JobOutcomeSummary) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.data_required", "[ERR-DEFAULT] Job outcome summary data is required"))
	}
	if data.JobId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_outcome_summary.validation.job_id_required", "[ERR-DEFAULT] Job ID is required"))
	}

	return nil
}
