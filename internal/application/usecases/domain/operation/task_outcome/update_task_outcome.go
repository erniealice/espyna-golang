package task_outcome

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type UpdateTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type UpdateTaskOutcomeServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
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

	// Approval-aware (PostgreSQL) path: require a transaction and run the
	// cell-write lock protocol before the leaf write — NO nontransactional
	// fallback (codex FIX-FIRST 2).
	guard, guarded := isGuardedRepo(uc.repositories.TaskOutcome)
	txCapable := uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions()
	if txCapable && !guarded {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.guard_required", "[ERR-DEFAULT] task_outcome update requires the cell-write guard on a transactional provider"))
	}
	if guarded {
		if !txCapable {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.transaction_required", "[ERR-DEFAULT] task_outcome update requires a transaction (cell-write lock protocol)"))
		}
		return uc.executeGuarded(ctx, req, enrichedData, guard)
	}

	// Non-guarded provider (mock/firestore) — behaviour unchanged.
	if txCapable {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}
	return uc.executeCore(ctx, req, enrichedData)
}

// executeGuarded resolves the job_task that owns the target outcome under the
// transaction, runs the cell-write guard on that task's phase, then updates.
func (uc *UpdateTaskOutcomeUseCase) executeGuarded(ctx context.Context, req *pb.UpdateTaskOutcomeRequest, enrichedData *pb.TaskOutcome, guard cellWriteGuard) (*pb.UpdateTaskOutcomeResponse, error) {
	var result *pb.UpdateTaskOutcomeResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		existing, rerr := uc.repositories.TaskOutcome.ReadTaskOutcome(txCtx, &pb.ReadTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: req.Data.Id}})
		if rerr != nil || existing == nil || len(existing.GetData()) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
		}
		// Guard on the EXISTING owner's task so a reparent attempt still locks the
		// original phase.
		existingTaskID := existing.GetData()[0].GetJobTaskId()
		if err := guard.GuardCellWrite(txCtx, existingTaskID); err != nil {
			return err
		}
		// Membership immutability (codex P3 §A2): pin the anchors to the EXISTING
		// row so this update can never reparent the outcome into a different cell —
		// defense-in-depth ahead of the adapter's own strip (the guard only locked
		// the ORIGINAL phase, so persisting a caller-supplied new anchor would move
		// the leaf unguarded).
		enrichedData.JobTaskId = existingTaskID
		enrichedData.CriteriaVersionId = existing.GetData()[0].GetCriteriaVersionId()
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
