package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type GetOutcomeCriteriaListPageDataRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type GetOutcomeCriteriaListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetOutcomeCriteriaListPageDataUseCase handles the business logic for getting outcome criteria list page data
type GetOutcomeCriteriaListPageDataUseCase struct {
	repositories GetOutcomeCriteriaListPageDataRepositories
	services     GetOutcomeCriteriaListPageDataServices
}

// NewGetOutcomeCriteriaListPageDataUseCase creates a new GetOutcomeCriteriaListPageDataUseCase
func NewGetOutcomeCriteriaListPageDataUseCase(
	repositories GetOutcomeCriteriaListPageDataRepositories,
	services GetOutcomeCriteriaListPageDataServices,
) *GetOutcomeCriteriaListPageDataUseCase {
	return &GetOutcomeCriteriaListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get outcome criteria list page data operation
func (uc *GetOutcomeCriteriaListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.OutcomeCriteria, entityid.ActionList); err != nil {
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
func (uc *GetOutcomeCriteriaListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {
	var result *pb.GetOutcomeCriteriaListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"outcome_criteria.errors.list_page_data_failed",
				"outcome criteria list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting outcome criteria list page data
func (uc *GetOutcomeCriteriaListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.GetOutcomeCriteriaListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.errors.list_page_data_failed",
			"failed to retrieve outcome criteria list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetOutcomeCriteriaListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.validation.request_required",
			"request is required",
		))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetOutcomeCriteriaListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"outcome_criteria.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
