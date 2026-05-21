package task_outcome

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ReadTaskOutcomeRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ReadTaskOutcomeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadTaskOutcomeUseCase handles the business logic for reading task outcomes
type ReadTaskOutcomeUseCase struct {
	repositories ReadTaskOutcomeRepositories
	services     ReadTaskOutcomeServices
}

// NewReadTaskOutcomeUseCase creates a new ReadTaskOutcomeUseCase
func NewReadTaskOutcomeUseCase(
	repositories ReadTaskOutcomeRepositories,
	services ReadTaskOutcomeServices,
) *ReadTaskOutcomeUseCase {
	return &ReadTaskOutcomeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read task outcome operation
func (uc *ReadTaskOutcomeUseCase) Execute(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcome, ports.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadTaskOutcomeUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	var result *pb.ReadTaskOutcomeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for reading a task outcome
func (uc *ReadTaskOutcomeUseCase) executeCore(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	resp, err := uc.repositories.TaskOutcome.ReadTaskOutcome(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.errors.not_found", "[ERR-DEFAULT] Task outcome not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadTaskOutcomeUseCase) validateInput(ctx context.Context, req *pb.ReadTaskOutcomeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.data_required", "[ERR-DEFAULT] Task outcome data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome.validation.id_required", "[ERR-DEFAULT] Task outcome ID is required"))
	}
	return nil
}
