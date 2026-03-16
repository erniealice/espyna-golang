package criteria_option

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type GetCriteriaOptionListPageDataRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type GetCriteriaOptionListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetCriteriaOptionListPageDataUseCase handles the business logic for getting criteria option list page data
type GetCriteriaOptionListPageDataUseCase struct {
	repositories GetCriteriaOptionListPageDataRepositories
	services     GetCriteriaOptionListPageDataServices
}

// NewGetCriteriaOptionListPageDataUseCase creates a new GetCriteriaOptionListPageDataUseCase
func NewGetCriteriaOptionListPageDataUseCase(
	repositories GetCriteriaOptionListPageDataRepositories,
	services GetCriteriaOptionListPageDataServices,
) *GetCriteriaOptionListPageDataUseCase {
	return &GetCriteriaOptionListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get criteria option list page data operation
func (uc *GetCriteriaOptionListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) (*pb.GetCriteriaOptionListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaOption, ports.ActionList); err != nil {
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

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetCriteriaOptionListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) (*pb.GetCriteriaOptionListPageDataResponse, error) {
	var result *pb.GetCriteriaOptionListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"criteria_option.errors.list_page_data_failed",
				"criteria option list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting criteria option list page data
func (uc *GetCriteriaOptionListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) (*pb.GetCriteriaOptionListPageDataResponse, error) {
	resp, err := uc.repositories.CriteriaOption.GetCriteriaOptionListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_option.errors.list_page_data_failed",
			"failed to retrieve criteria option list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetCriteriaOptionListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_option.validation.request_required",
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
func (uc *GetCriteriaOptionListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_option.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
