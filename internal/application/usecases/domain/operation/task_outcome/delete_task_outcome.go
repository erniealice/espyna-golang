package task_outcome

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type DeleteTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type DeleteTaskOutcomeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
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

	// Approval-aware (PostgreSQL) path: require a transaction and run the
	// cell-write lock protocol before the soft-delete — NO nontransactional
	// fallback (codex FIX-FIRST 2).
	guard, guarded := isGuardedRepo(uc.repositories.TaskOutcome)
	txCapable := uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions()
	if txCapable && !guarded {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.guard_required", "[ERR-DEFAULT] task_outcome delete requires the cell-write guard on a transactional provider"))
	}
	if guarded {
		if !txCapable {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.transaction_required", "[ERR-DEFAULT] task_outcome delete requires a transaction (cell-write lock protocol)"))
		}
		return uc.executeGuarded(ctx, req, guard)
	}

	// Non-guarded provider (mock/firestore) — behaviour unchanged.
	if txCapable {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeGuarded resolves the job_task that owns the target outcome under the
// transaction, runs the cell-write guard, then soft-deletes.
func (uc *DeleteTaskOutcomeUseCase) executeGuarded(ctx context.Context, req *pb.DeleteTaskOutcomeRequest, guard cellWriteGuard) (*pb.DeleteTaskOutcomeResponse, error) {
	var result *pb.DeleteTaskOutcomeResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		existing, rerr := uc.repositories.TaskOutcome.ReadTaskOutcome(txCtx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: req.Data.Id}})
		if rerr != nil || existing == nil || len(existing.GetData()) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
		}
		if err := guard.GuardCellWrite(txCtx, existing.GetData()[0].GetJobTaskId()); err != nil {
			return err
		}
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
