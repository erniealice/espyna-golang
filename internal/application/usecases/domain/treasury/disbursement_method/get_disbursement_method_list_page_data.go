package disbursementmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// GetDisbursementMethodListPageDataRepositories groups all repository dependencies.
type GetDisbursementMethodListPageDataRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// GetDisbursementMethodListPageDataServices groups all business service dependencies.
type GetDisbursementMethodListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetDisbursementMethodListPageDataUseCase handles fetching paginated, searchable list data.
type GetDisbursementMethodListPageDataUseCase struct {
	repositories GetDisbursementMethodListPageDataRepositories
	services     GetDisbursementMethodListPageDataServices
}

// NewGetDisbursementMethodListPageDataUseCase creates use case with grouped dependencies.
func NewGetDisbursementMethodListPageDataUseCase(
	repositories GetDisbursementMethodListPageDataRepositories,
	services GetDisbursementMethodListPageDataServices,
) *GetDisbursementMethodListPageDataUseCase {
	return &GetDisbursementMethodListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get disbursement method list page data operation.
func (uc *GetDisbursementMethodListPageDataUseCase) Execute(ctx context.Context, req *disbursementmethodpb.GetDisbursementMethodListPageDataRequest) (*disbursementmethodpb.GetDisbursementMethodListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	resp, err := uc.repositories.DisbursementMethod.GetDisbursementMethodListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load disbursement method list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetDisbursementMethodListPageDataUseCase) validateInput(ctx context.Context, req *disbursementmethodpb.GetDisbursementMethodListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
