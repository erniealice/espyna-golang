package criteria_threshold

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
)

type GetCriteriaThresholdListPageDataRepositories struct {
	CriteriaThreshold pb.CriteriaThresholdDomainServiceServer
}

type GetCriteriaThresholdListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetCriteriaThresholdListPageDataUseCase handles the business logic for getting criteria threshold list page data
type GetCriteriaThresholdListPageDataUseCase struct {
	repositories GetCriteriaThresholdListPageDataRepositories
	services     GetCriteriaThresholdListPageDataServices
}

// NewGetCriteriaThresholdListPageDataUseCase creates a new GetCriteriaThresholdListPageDataUseCase
func NewGetCriteriaThresholdListPageDataUseCase(
	repositories GetCriteriaThresholdListPageDataRepositories,
	services GetCriteriaThresholdListPageDataServices,
) *GetCriteriaThresholdListPageDataUseCase {
	return &GetCriteriaThresholdListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get criteria threshold list page data operation
func (uc *GetCriteriaThresholdListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetCriteriaThresholdListPageDataRequest,
) (*pb.GetCriteriaThresholdListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaThreshold, ports.ActionList); err != nil {
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
func (uc *GetCriteriaThresholdListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetCriteriaThresholdListPageDataRequest,
) (*pb.GetCriteriaThresholdListPageDataResponse, error) {
	var result *pb.GetCriteriaThresholdListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"criteria_threshold.errors.list_page_data_failed",
				"criteria threshold list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting criteria threshold list page data
func (uc *GetCriteriaThresholdListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetCriteriaThresholdListPageDataRequest,
) (*pb.GetCriteriaThresholdListPageDataResponse, error) {
	resp, err := uc.repositories.CriteriaThreshold.GetCriteriaThresholdListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_threshold.errors.list_page_data_failed",
			"failed to retrieve criteria threshold list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetCriteriaThresholdListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetCriteriaThresholdListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_threshold.validation.request_required",
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
func (uc *GetCriteriaThresholdListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"criteria_threshold.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
