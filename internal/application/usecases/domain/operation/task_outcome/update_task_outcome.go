package task_outcome

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type UpdateTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type UpdateTaskOutcomeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateTaskOutcomeUseCase handles the business logic for updating task outcomes
type UpdateTaskOutcomeUseCase struct {
	repositories UpdateTaskOutcomeRepositories
	services     UpdateTaskOutcomeServices
}

// NewUpdateTaskOutcomeUseCase creates a new UpdateTaskOutcomeUseCase
func NewUpdateTaskOutcomeUseCase(
	repositories UpdateTaskOutcomeRepositories,
	services UpdateTaskOutcomeServices,
) *UpdateTaskOutcomeUseCase {
	return &UpdateTaskOutcomeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update task outcome operation
func (uc *UpdateTaskOutcomeUseCase) Execute(ctx context.Context, req *pb.UpdateTaskOutcomeRequest) (*pb.UpdateTaskOutcomeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.TaskOutcome, entityid.ActionUpdate); err != nil {
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
func (uc *UpdateTaskOutcomeUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateTaskOutcomeRequest, enrichedData *pb.TaskOutcome) (*pb.UpdateTaskOutcomeResponse, error) {
	var result *pb.UpdateTaskOutcomeResponse

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

// executeCore contains the core business logic for updating a task outcome
func (uc *UpdateTaskOutcomeUseCase) executeCore(ctx context.Context, req *pb.UpdateTaskOutcomeRequest, enrichedData *pb.TaskOutcome) (*pb.UpdateTaskOutcomeResponse, error) {
	_, err := uc.repositories.TaskOutcome.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{
		Data: &pb.TaskOutcome{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
	}

	resp, err := uc.repositories.TaskOutcome.UpdateTaskOutcome(ctx, &pb.UpdateTaskOutcomeRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.update_failed", "[ERR-DEFAULT] Task outcome update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateTaskOutcomeUseCase) applyBusinessLogic(data *pb.TaskOutcome) *pb.TaskOutcome {
	now := time.Now()
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return data
}

// validateInput validates the input request
func (uc *UpdateTaskOutcomeUseCase) validateInput(ctx context.Context, req *pb.UpdateTaskOutcomeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.validation.id_required", "[ERR-DEFAULT] Task outcome ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateTaskOutcomeUseCase) validateBusinessRules(ctx context.Context, data *pb.TaskOutcome) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.validation.id_required", "[ERR-DEFAULT] Task outcome ID is required"))
	}
	return nil
}
