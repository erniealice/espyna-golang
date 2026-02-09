package payment_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

type GetPaymentAttributeListPageDataRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer
}

type GetPaymentAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPaymentAttributeListPageDataUseCase handles the business logic for getting payment attribute list page data
type GetPaymentAttributeListPageDataUseCase struct {
	repositories GetPaymentAttributeListPageDataRepositories
	services     GetPaymentAttributeListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetPaymentAttributeListPageDataUseCase creates a new GetPaymentAttributeListPageDataUseCase
func NewGetPaymentAttributeListPageDataUseCase(
	repositories GetPaymentAttributeListPageDataRepositories,
	services GetPaymentAttributeListPageDataServices,
) *GetPaymentAttributeListPageDataUseCase {
	return &GetPaymentAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get payment attribute list page data operation
func (uc *GetPaymentAttributeListPageDataUseCase) Execute(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeListPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentAttribute, ports.ActionList); err != nil {
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

// executeWithTransaction executes payment attribute list page data retrieval within a transaction
func (uc *GetPaymentAttributeListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeListPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeListPageDataResponse, error) {
	var result *paymentattributepb.GetPaymentAttributeListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"payment_attribute.errors.list_page_data_failed",
				"payment attribute list page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting payment attribute list page data
func (uc *GetPaymentAttributeListPageDataUseCase) executeCore(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeListPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeListPageDataResponse, error) {
	// First, get all payment attributes from the repository
	listReq := &paymentattributepb.ListPaymentAttributesRequest{}
	listResp, err := uc.repositories.PaymentAttribute.ListPaymentAttributes(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.errors.list_failed",
			"failed to retrieve payment attributes: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		// Return empty response with proper pagination metadata
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &paymentattributepb.GetPaymentAttributeListPageDataResponse{
			PaymentAttributeList: []*paymentattributepb.PaymentAttribute{},
			Pagination:           emptyPagination,
			SearchResults:        []*commonpb.SearchResult{},
			Success:              true,
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
			"payment_attribute.errors.processing_failed",
			"failed to process payment attribute list data: %w",
		), err)
	}

	// Convert processed items back to payment attribute protobuf format
	paymentAttributes := make([]*paymentattributepb.PaymentAttribute, len(result.Items))
	for i, item := range result.Items {
		if paymentAttribute, ok := item.(*paymentattributepb.PaymentAttribute); ok {
			paymentAttributes[i] = paymentAttribute
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"payment_attribute.errors.type_conversion_failed",
				"failed to convert item to payment attribute type",
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

	return &paymentattributepb.GetPaymentAttributeListPageDataResponse{
		PaymentAttributeList: paymentAttributes,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPaymentAttributeListPageDataUseCase) validateInput(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.validation.request_required",
			"request is required",
		))
	}
	return nil
}

// isValidPaymentAttributeField checks if a field name is valid for payment attribute filtering/sorting/searching
func (uc *GetPaymentAttributeListPageDataUseCase) isValidPaymentAttributeField(field string) bool {
	validFields := map[string]bool{
		"id":                   true,
		"name":                 true,
		"payment_id":           true,
		"attribute_id":         true,
		"value":                true,
		"active":               true,
		"date_created":         true,
		"date_created_string":  true,
		"date_modified":        true,
		"date_modified_string": true,
		// Nested fields
		"payment.name":   true,
		"payment.id":     true,
		"attribute.name": true,
		"attribute.id":   true,
	}

	return validFields[field]
}
