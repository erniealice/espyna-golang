package revenuepayment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// GetRevenuePaymentListPageDataRepositories groups all repository dependencies
type GetRevenuePaymentListPageDataRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer // Primary entity repository
}

// GetRevenuePaymentListPageDataServices groups all business service dependencies
type GetRevenuePaymentListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetRevenuePaymentListPageDataUseCase handles the business logic for getting revenue payment list page data with pagination, filtering, sorting, and search
type GetRevenuePaymentListPageDataUseCase struct {
	repositories GetRevenuePaymentListPageDataRepositories
	services     GetRevenuePaymentListPageDataServices
}

// NewGetRevenuePaymentListPageDataUseCase creates use case with grouped dependencies
func NewGetRevenuePaymentListPageDataUseCase(
	repositories GetRevenuePaymentListPageDataRepositories,
	services GetRevenuePaymentListPageDataServices,
) *GetRevenuePaymentListPageDataUseCase {
	return &GetRevenuePaymentListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get revenue payment list page data operation.
//
// GetRevenuePaymentListPageDataRequest carries an optional FilterRequest (filters=2)
// which the W4 adapter honors as a server-side revenue_id filter (design doc §4 / §5.4).
func (uc *GetRevenuePaymentListPageDataUseCase) Execute(ctx context.Context, req *pb.GetRevenuePaymentListPageDataRequest) (*pb.GetRevenuePaymentListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenuePayment, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.GetRevenuePaymentListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load revenue payment list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetRevenuePaymentListPageDataUseCase) validateInput(ctx context.Context, req *pb.GetRevenuePaymentListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 && (req.Pagination.Limit < 1 || req.Pagination.Limit > 100) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
		}
	}

	// Validate filter parameters
	if req.Filters != nil && len(req.Filters.Filters) > 10 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.too_many_filters", "[ERR-DEFAULT] Too many filters"))
	}

	// Validate sort parameters
	if req.Sort != nil && len(req.Sort.Fields) > 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.too_many_sort_fields", "[ERR-DEFAULT] Too many sort fields"))
	}

	// Validate search parameters
	if req.Search != nil && req.Search.Query != "" {
		if len(req.Search.Query) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
		}
	}

	return nil
}
