package task_outcome_check

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type UpdateTaskOutcomeCheckRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type UpdateTaskOutcomeCheckServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateTaskOutcomeCheckUseCase handles the business logic for updating task outcome checks
type UpdateTaskOutcomeCheckUseCase struct {
	repositories UpdateTaskOutcomeCheckRepositories
	services     UpdateTaskOutcomeCheckServices
}

// NewUpdateTaskOutcomeCheckUseCase creates a new UpdateTaskOutcomeCheckUseCase
func NewUpdateTaskOutcomeCheckUseCase(
	repositories UpdateTaskOutcomeCheckRepositories,
	services UpdateTaskOutcomeCheckServices,
) *UpdateTaskOutcomeCheckUseCase {
	return &UpdateTaskOutcomeCheckUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update task outcome check operation
func (uc *UpdateTaskOutcomeCheckUseCase) Execute(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest) (*pb.UpdateTaskOutcomeCheckResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcomeCheck, ports.ActionUpdate); err != nil {
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
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes update within a transaction
func (uc *UpdateTaskOutcomeCheckUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest, enrichedData *pb.TaskOutcomeCheck) (*pb.UpdateTaskOutcomeCheckResponse, error) {
	var result *pb.UpdateTaskOutcomeCheckResponse

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

// executeCore contains the core business logic for updating a task outcome check
func (uc *UpdateTaskOutcomeCheckUseCase) executeCore(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest, enrichedData *pb.TaskOutcomeCheck) (*pb.UpdateTaskOutcomeCheckResponse, error) {
	_, err := uc.repositories.TaskOutcomeCheck.ReadTaskOutcomeCheck(ctx, &pb.ReadTaskOutcomeCheckRequest{
		Data: &pb.TaskOutcomeCheck{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.errors.not_found", "[ERR-DEFAULT] Task outcome check not found"))
	}

	resp, err := uc.repositories.TaskOutcomeCheck.UpdateTaskOutcomeCheck(ctx, &pb.UpdateTaskOutcomeCheckRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.errors.update_failed", "[ERR-DEFAULT] Task outcome check update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateTaskOutcomeCheckUseCase) applyBusinessLogic(data *pb.TaskOutcomeCheck) *pb.TaskOutcomeCheck {
	return data
}

// validateInput validates the input request
func (uc *UpdateTaskOutcomeCheckUseCase) validateInput(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.data_required", "[ERR-DEFAULT] Task outcome check data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.id_required", "[ERR-DEFAULT] Task outcome check ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateTaskOutcomeCheckUseCase) validateBusinessRules(ctx context.Context, data *pb.TaskOutcomeCheck) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.data_required", "[ERR-DEFAULT] Task outcome check data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.id_required", "[ERR-DEFAULT] Task outcome check ID is required"))
	}
	return nil
}
