package task_outcome_check

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type ListByTaskOutcomeRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type ListByTaskOutcomeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByTaskOutcomeUseCase handles the business logic for listing checks by task outcome
type ListByTaskOutcomeUseCase struct {
	repositories ListByTaskOutcomeRepositories
	services     ListByTaskOutcomeServices
}

// NewListByTaskOutcomeUseCase creates a new ListByTaskOutcomeUseCase
func NewListByTaskOutcomeUseCase(
	repositories ListByTaskOutcomeRepositories,
	services ListByTaskOutcomeServices,
) *ListByTaskOutcomeUseCase {
	return &ListByTaskOutcomeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by task outcome operation
func (uc *ListByTaskOutcomeUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomeChecksByTaskOutcomeRequest) (*pb.ListTaskOutcomeChecksByTaskOutcomeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcomeCheck, ports.ActionList); err != nil {
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

// executeWithTransaction executes list by task outcome within a transaction
func (uc *ListByTaskOutcomeUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomeChecksByTaskOutcomeRequest) (*pb.ListTaskOutcomeChecksByTaskOutcomeResponse, error) {
	var result *pb.ListTaskOutcomeChecksByTaskOutcomeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "task_outcome_check.errors.list_by_task_outcome_failed", "task outcome check listing by task outcome failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing checks by task outcome
func (uc *ListByTaskOutcomeUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomeChecksByTaskOutcomeRequest) (*pb.ListTaskOutcomeChecksByTaskOutcomeResponse, error) {
	resp, err := uc.repositories.TaskOutcomeCheck.ListByTaskOutcome(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.errors.list_by_task_outcome_failed", "failed to list task outcome checks by task outcome: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByTaskOutcomeUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomeChecksByTaskOutcomeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.request_required", "request is required"))
	}
	if req.TaskOutcomeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.task_outcome_id_required", "task outcome ID is required"))
	}

	return nil
}
