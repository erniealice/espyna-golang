package task_outcome

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type DeleteTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type DeleteTaskOutcomeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteTaskOutcomeUseCase handles the business logic for deleting task outcomes
type DeleteTaskOutcomeUseCase struct {
	repositories DeleteTaskOutcomeRepositories
	services     DeleteTaskOutcomeServices
}

// NewDeleteTaskOutcomeUseCase creates a new DeleteTaskOutcomeUseCase
func NewDeleteTaskOutcomeUseCase(
	repositories DeleteTaskOutcomeRepositories,
	services DeleteTaskOutcomeServices,
) *DeleteTaskOutcomeUseCase {
	return &DeleteTaskOutcomeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete task outcome operation
func (uc *DeleteTaskOutcomeUseCase) Execute(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
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
func (uc *DeleteTaskOutcomeUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	var result *pb.DeleteTaskOutcomeResponse

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

// executeCore contains the core business logic for deleting a task outcome
func (uc *DeleteTaskOutcomeUseCase) executeCore(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	_, err := uc.repositories.TaskOutcome.ReadTaskOutcome(ctx, &pb.ReadTaskOutcomeRequest{
		Data: &pb.TaskOutcome{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
	}

	resp, err := uc.repositories.TaskOutcome.DeleteTaskOutcome(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.deletion_failed", "[ERR-DEFAULT] Task outcome deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteTaskOutcomeUseCase) validateInput(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) error {
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
