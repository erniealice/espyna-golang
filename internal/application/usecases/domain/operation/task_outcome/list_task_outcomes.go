package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ListTaskOutcomesRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ListTaskOutcomesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListTaskOutcomesUseCase handles the business logic for listing task outcomes
type ListTaskOutcomesUseCase struct {
	repositories ListTaskOutcomesRepositories
	services     ListTaskOutcomesServices
}

// NewListTaskOutcomesUseCase creates a new ListTaskOutcomesUseCase
func NewListTaskOutcomesUseCase(
	repositories ListTaskOutcomesRepositories,
	services ListTaskOutcomesServices,
) *ListTaskOutcomesUseCase {
	return &ListTaskOutcomesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list task outcomes operation
func (uc *ListTaskOutcomesUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomesRequest) (*pb.ListTaskOutcomesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcome, ports.ActionList); err != nil {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListTaskOutcomesUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomesRequest) (*pb.ListTaskOutcomesResponse, error) {
	var result *pb.ListTaskOutcomesResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "task_outcome.errors.list_failed", "task outcome listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing task outcomes
func (uc *ListTaskOutcomesUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomesRequest) (*pb.ListTaskOutcomesResponse, error) {
	resp, err := uc.repositories.TaskOutcome.ListTaskOutcomes(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.list_failed", "task outcome listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListTaskOutcomesUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.request_required", "request is required"))
	}

	return nil
}
