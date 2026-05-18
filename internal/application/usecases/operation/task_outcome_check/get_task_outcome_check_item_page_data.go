package task_outcome_check

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type GetTaskOutcomeCheckItemPageDataRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type GetTaskOutcomeCheckItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetTaskOutcomeCheckItemPageDataUseCase handles the business logic for getting task outcome check item page data
type GetTaskOutcomeCheckItemPageDataUseCase struct {
	repositories GetTaskOutcomeCheckItemPageDataRepositories
	services     GetTaskOutcomeCheckItemPageDataServices
}

// NewGetTaskOutcomeCheckItemPageDataUseCase creates a new GetTaskOutcomeCheckItemPageDataUseCase
func NewGetTaskOutcomeCheckItemPageDataUseCase(
	repositories GetTaskOutcomeCheckItemPageDataRepositories,
	services GetTaskOutcomeCheckItemPageDataServices,
) *GetTaskOutcomeCheckItemPageDataUseCase {
	return &GetTaskOutcomeCheckItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get task outcome check item page data operation
func (uc *GetTaskOutcomeCheckItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckItemPageDataRequest,
) (*pb.GetTaskOutcomeCheckItemPageDataResponse, error) {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetTaskOutcomeCheckItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckItemPageDataRequest,
) (*pb.GetTaskOutcomeCheckItemPageDataResponse, error) {
	var result *pb.GetTaskOutcomeCheckItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"task_outcome_check.errors.item_page_data_failed",
				"task outcome check item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting task outcome check item page data
func (uc *GetTaskOutcomeCheckItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckItemPageDataRequest,
) (*pb.GetTaskOutcomeCheckItemPageDataResponse, error) {
	readReq := &pb.ReadTaskOutcomeCheckRequest{
		Data: &pb.TaskOutcomeCheck{
			Id: req.TaskOutcomeCheckId,
		},
	}

	readResp, err := uc.repositories.TaskOutcomeCheck.ReadTaskOutcomeCheck(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome_check.errors.read_failed",
			"failed to retrieve task outcome check: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome_check.errors.not_found",
			"task outcome check not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetTaskOutcomeCheckItemPageDataResponse{
		TaskOutcomeCheck: item,
		Success:          true,
	}, nil
}

// validateInput validates the input request
func (uc *GetTaskOutcomeCheckItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome_check.validation.request_required",
			"request is required",
		))
	}

	if req.TaskOutcomeCheckId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"task_outcome_check.validation.id_required",
			"task outcome check ID is required",
		))
	}

	return nil
}
