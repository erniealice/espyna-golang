package task_outcome

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type GetTaskOutcomeItemPageDataRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type GetTaskOutcomeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetTaskOutcomeItemPageDataUseCase handles the business logic for getting task outcome item page data
type GetTaskOutcomeItemPageDataUseCase struct {
	repositories GetTaskOutcomeItemPageDataRepositories
	services     GetTaskOutcomeItemPageDataServices
}

// NewGetTaskOutcomeItemPageDataUseCase creates a new GetTaskOutcomeItemPageDataUseCase
func NewGetTaskOutcomeItemPageDataUseCase(
	repositories GetTaskOutcomeItemPageDataRepositories,
	services GetTaskOutcomeItemPageDataServices,
) *GetTaskOutcomeItemPageDataUseCase {
	return &GetTaskOutcomeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get task outcome item page data operation
func (uc *GetTaskOutcomeItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTaskOutcomeItemPageDataRequest,
) (*pb.GetTaskOutcomeItemPageDataResponse, error) {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetTaskOutcomeItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTaskOutcomeItemPageDataRequest,
) (*pb.GetTaskOutcomeItemPageDataResponse, error) {
	var result *pb.GetTaskOutcomeItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"task_outcome.errors.item_page_data_failed",
				"task outcome item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting task outcome item page data
func (uc *GetTaskOutcomeItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTaskOutcomeItemPageDataRequest,
) (*pb.GetTaskOutcomeItemPageDataResponse, error) {
	readReq := &pb.ReadTaskOutcomeRequest{
		Data: &pb.TaskOutcome{
			Id: req.TaskOutcomeId,
		},
	}

	readResp, err := uc.repositories.TaskOutcome.ReadTaskOutcome(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.errors.read_failed",
			"failed to retrieve task outcome: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.errors.not_found",
			"task outcome not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetTaskOutcomeItemPageDataResponse{
		TaskOutcome: item,
		Success:     true,
	}, nil
}

// validateInput validates the input request
func (uc *GetTaskOutcomeItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTaskOutcomeItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.validation.request_required",
			"request is required",
		))
	}

	if req.TaskOutcomeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome.validation.id_required",
			"task outcome ID is required",
		))
	}

	return nil
}
