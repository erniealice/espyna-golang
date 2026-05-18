package task_outcome

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type CreateTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type CreateTaskOutcomeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateTaskOutcomeUseCase handles the business logic for creating task outcomes
type CreateTaskOutcomeUseCase struct {
	repositories CreateTaskOutcomeRepositories
	services     CreateTaskOutcomeServices
}

// NewCreateTaskOutcomeUseCase creates a new CreateTaskOutcomeUseCase
func NewCreateTaskOutcomeUseCase(
	repositories CreateTaskOutcomeRepositories,
	services CreateTaskOutcomeServices,
) *CreateTaskOutcomeUseCase {
	return &CreateTaskOutcomeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create task outcome operation
func (uc *CreateTaskOutcomeUseCase) Execute(ctx context.Context, req *pb.CreateTaskOutcomeRequest) (*pb.CreateTaskOutcomeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcome, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateTaskOutcomeUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateTaskOutcomeRequest, enrichedData *pb.TaskOutcome) (*pb.CreateTaskOutcomeResponse, error) {
	var result *pb.CreateTaskOutcomeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for creating a task outcome
func (uc *CreateTaskOutcomeUseCase) executeCore(ctx context.Context, req *pb.CreateTaskOutcomeRequest, enrichedData *pb.TaskOutcome) (*pb.CreateTaskOutcomeResponse, error) {
	resp, err := uc.repositories.TaskOutcome.CreateTaskOutcome(ctx, &pb.CreateTaskOutcomeRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.creation_failed", "[ERR-DEFAULT] Task outcome creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateTaskOutcomeUseCase) applyBusinessLogic(data *pb.TaskOutcome) *pb.TaskOutcome {
	now := time.Now()

	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateTaskOutcomeUseCase) validateBusinessRules(ctx context.Context, data *pb.TaskOutcome) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	if data.JobTaskId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.job_task_id_required", "[ERR-DEFAULT] Job task ID is required"))
	}
	if data.CriteriaVersionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.criteria_id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}

	return nil
}
