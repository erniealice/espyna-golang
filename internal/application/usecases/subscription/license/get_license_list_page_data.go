package license

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

// GetLicenseListPageDataRepositories groups all repository dependencies
type GetLicenseListPageDataRepositories struct {
	License licensepb.LicenseDomainServiceServer
}

// GetLicenseListPageDataServices groups all business service dependencies
type GetLicenseListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetLicenseListPageDataUseCase handles the business logic for getting license list page data
type GetLicenseListPageDataUseCase struct {
	repositories GetLicenseListPageDataRepositories
	services     GetLicenseListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetLicenseListPageDataUseCase creates a new GetLicenseListPageDataUseCase
func NewGetLicenseListPageDataUseCase(
	repositories GetLicenseListPageDataRepositories,
	services GetLicenseListPageDataServices,
) *GetLicenseListPageDataUseCase {
	return &GetLicenseListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get license list page data operation
func (uc *GetLicenseListPageDataUseCase) Execute(
	ctx context.Context,
	req *licensepb.GetLicenseListPageDataRequest,
) (*licensepb.GetLicenseListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLicense, ports.ActionList); err != nil {
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

// executeWithTransaction executes license list page data retrieval within a transaction
func (uc *GetLicenseListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *licensepb.GetLicenseListPageDataRequest,
) (*licensepb.GetLicenseListPageDataResponse, error) {
	var result *licensepb.GetLicenseListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"license.errors.list_page_data_failed",
				"license list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting license list page data
func (uc *GetLicenseListPageDataUseCase) executeCore(
	ctx context.Context,
	req *licensepb.GetLicenseListPageDataRequest,
) (*licensepb.GetLicenseListPageDataResponse, error) {
	// First, get all licenses from the repository
	listReq := &licensepb.ListLicensesRequest{}
	listResp, err := uc.repositories.License.ListLicenses(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.errors.list_failed",
			"failed to retrieve licenses: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &licensepb.GetLicenseListPageDataResponse{
			LicenseList:   []*licensepb.License{},
			Pagination:    emptyPagination,
			SearchResults: []*commonpb.SearchResult{},
			Success:       true,
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
			"license.errors.processing_failed",
			"failed to process license list data: %w",
		), err)
	}

	// Convert processed items back to license protobuf format
	licenses := make([]*licensepb.License, len(result.Items))
	for i, item := range result.Items {
		if license, ok := item.(*licensepb.License); ok {
			licenses[i] = license
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"license.errors.type_conversion_failed",
				"failed to convert item to license type",
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

	return &licensepb.GetLicenseListPageDataResponse{
		LicenseList:   licenses,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// validateInput validates the input request
func (uc *GetLicenseListPageDataUseCase) validateInput(
	ctx context.Context,
	req *licensepb.GetLicenseListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.validation.request_required",
			"request is required",
		))
	}

	return nil
}
