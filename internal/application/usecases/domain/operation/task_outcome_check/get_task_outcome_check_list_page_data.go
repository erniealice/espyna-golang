package task_outcome_check

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type GetTaskOutcomeCheckListPageDataRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type GetTaskOutcomeCheckListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetTaskOutcomeCheckListPageDataUseCase handles the business logic for getting task outcome check list page data
type GetTaskOutcomeCheckListPageDataUseCase struct {
	repositories GetTaskOutcomeCheckListPageDataRepositories
	services     GetTaskOutcomeCheckListPageDataServices
}

// NewGetTaskOutcomeCheckListPageDataUseCase creates a new GetTaskOutcomeCheckListPageDataUseCase
func NewGetTaskOutcomeCheckListPageDataUseCase(
	repositories GetTaskOutcomeCheckListPageDataRepositories,
	services GetTaskOutcomeCheckListPageDataServices,
) *GetTaskOutcomeCheckListPageDataUseCase {
	return &GetTaskOutcomeCheckListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get task outcome check list page data operation
func (uc *GetTaskOutcomeCheckListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckListPageDataRequest,
) (*pb.GetTaskOutcomeCheckListPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcomeCheck,
		Action: entityid.ActionList,
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

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetTaskOutcomeCheckListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckListPageDataRequest,
) (*pb.GetTaskOutcomeCheckListPageDataResponse, error) {
	var result *pb.GetTaskOutcomeCheckListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"task_outcome_check.errors.list_page_data_failed",
				"task outcome check list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting task outcome check list page data
func (uc *GetTaskOutcomeCheckListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckListPageDataRequest,
) (*pb.GetTaskOutcomeCheckListPageDataResponse, error) {
	resp, err := uc.repositories.TaskOutcomeCheck.GetTaskOutcomeCheckListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"task_outcome_check.errors.list_page_data_failed",
			"failed to retrieve task outcome check list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetTaskOutcomeCheckListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"task_outcome_check.validation.request_required",
			"request is required",
		))
	}

	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetTaskOutcomeCheckListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"task_outcome_check.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
