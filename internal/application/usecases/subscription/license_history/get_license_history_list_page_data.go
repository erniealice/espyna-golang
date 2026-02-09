package licensehistory

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

// GetLicenseHistoryListPageDataRepositories groups all repository dependencies
type GetLicenseHistoryListPageDataRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer
}

// GetLicenseHistoryListPageDataServices groups all business service dependencies
type GetLicenseHistoryListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetLicenseHistoryListPageDataUseCase handles the business logic for getting license history list page data
type GetLicenseHistoryListPageDataUseCase struct {
	repositories GetLicenseHistoryListPageDataRepositories
	services     GetLicenseHistoryListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetLicenseHistoryListPageDataUseCase creates a new GetLicenseHistoryListPageDataUseCase
func NewGetLicenseHistoryListPageDataUseCase(
	repositories GetLicenseHistoryListPageDataRepositories,
	services GetLicenseHistoryListPageDataServices,
) *GetLicenseHistoryListPageDataUseCase {
	return &GetLicenseHistoryListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get license history list page data operation
func (uc *GetLicenseHistoryListPageDataUseCase) Execute(
	ctx context.Context,
	req *licensehistorypb.GetLicenseHistoryListPageDataRequest,
) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLicenseHistory, ports.ActionList); err != nil {
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

// executeWithTransaction executes license history list page data retrieval within a transaction
func (uc *GetLicenseHistoryListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *licensehistorypb.GetLicenseHistoryListPageDataRequest,
) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
	var result *licensehistorypb.GetLicenseHistoryListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"license_history.errors.list_page_data_failed",
				"license history list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting license history list page data
func (uc *GetLicenseHistoryListPageDataUseCase) executeCore(
	ctx context.Context,
	req *licensehistorypb.GetLicenseHistoryListPageDataRequest,
) (*licensehistorypb.GetLicenseHistoryListPageDataResponse, error) {
	// First, get all license history from the repository
	listReq := &licensehistorypb.ListLicenseHistoryRequest{
		LicenseId: req.LicenseId,
	}
	listResp, err := uc.repositories.LicenseHistory.ListLicenseHistory(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license_history.errors.list_failed",
			"failed to retrieve license history: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
			LicenseHistoryList: []*licensehistorypb.LicenseHistory{},
			Pagination:         emptyPagination,
			SearchResults:      []*commonpb.SearchResult{},
			Success:            true,
		}, nil
	}

	// Process the data with filtering, sorting, searching, and pagination
	result, err := uc.processor.ProcessListRequest(
		listResp.Data,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license_history.errors.processing_failed",
			"failed to process license history list data: %w",
		), err)
	}

	// Convert processed items back to license history protobuf format
	historyList := make([]*licensehistorypb.LicenseHistory, len(result.Items))
	for i, item := range result.Items {
		if history, ok := item.(*licensehistorypb.LicenseHistory); ok {
			historyList[i] = history
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"license_history.errors.type_conversion_failed",
				"failed to convert item to license history type",
			))
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &licensehistorypb.GetLicenseHistoryListPageDataResponse{
		LicenseHistoryList: historyList,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// validateInput validates the input request
func (uc *GetLicenseHistoryListPageDataUseCase) validateInput(
	ctx context.Context,
	req *licensehistorypb.GetLicenseHistoryListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license_history.validation.request_required",
			"request is required",
		))
	}

	return nil
}
