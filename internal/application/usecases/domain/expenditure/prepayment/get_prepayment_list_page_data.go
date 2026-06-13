package prepayment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

// GetPrepaymentListPageDataRepositories groups all repository dependencies
type GetPrepaymentListPageDataRepositories struct {
	Prepayment prepaymentpb.PrepaymentDomainServiceServer
}

// GetPrepaymentListPageDataServices groups all business service dependencies
type GetPrepaymentListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetPrepaymentListPageDataUseCase handles fetching paginated, searchable prepayment list data
type GetPrepaymentListPageDataUseCase struct {
	repositories GetPrepaymentListPageDataRepositories
	services     GetPrepaymentListPageDataServices
}

// NewGetPrepaymentListPageDataUseCase creates use case with grouped dependencies
func NewGetPrepaymentListPageDataUseCase(
	repositories GetPrepaymentListPageDataRepositories,
	services GetPrepaymentListPageDataServices,
) *GetPrepaymentListPageDataUseCase {
	return &GetPrepaymentListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get prepayment list page data operation
func (uc *GetPrepaymentListPageDataUseCase) Execute(ctx context.Context, req *prepaymentpb.GetPrepaymentListPageDataRequest) (*prepaymentpb.GetPrepaymentListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityPrepayment,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "prepayment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.Prepayment == nil {
		return nil, errors.New("prepayment repository is not available")
	}
	resp, err := uc.repositories.Prepayment.GetPrepaymentListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "prepayment.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load prepayment list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetPrepaymentListPageDataUseCase) validateInput(ctx context.Context, req *prepaymentpb.GetPrepaymentListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "prepayment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "prepayment.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "prepayment.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
