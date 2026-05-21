package pettycash

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

// GetPettyCashFundListPageDataRepositories groups all repository dependencies
type GetPettyCashFundListPageDataRepositories struct {
	PettyCashFund pettycashfundpb.PettyCashFundDomainServiceServer
}

// GetPettyCashFundListPageDataServices groups all business service dependencies
type GetPettyCashFundListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPettyCashFundListPageDataUseCase handles fetching paginated, searchable petty cash fund list data
type GetPettyCashFundListPageDataUseCase struct {
	repositories GetPettyCashFundListPageDataRepositories
	services     GetPettyCashFundListPageDataServices
}

// NewGetPettyCashFundListPageDataUseCase creates use case with grouped dependencies
func NewGetPettyCashFundListPageDataUseCase(
	repositories GetPettyCashFundListPageDataRepositories,
	services GetPettyCashFundListPageDataServices,
) *GetPettyCashFundListPageDataUseCase {
	return &GetPettyCashFundListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get petty cash fund list page data operation
func (uc *GetPettyCashFundListPageDataUseCase) Execute(ctx context.Context, req *pettycashfundpb.GetPettyCashFundListPageDataRequest) (*pettycashfundpb.GetPettyCashFundListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPettyCashFund, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "petty_cash_fund.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PettyCashFund == nil {
		return nil, errors.New("petty cash fund repository is not available")
	}
	resp, err := uc.repositories.PettyCashFund.GetPettyCashFundListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "petty_cash_fund.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load petty cash fund list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetPettyCashFundListPageDataUseCase) validateInput(ctx context.Context, req *pettycashfundpb.GetPettyCashFundListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "petty_cash_fund.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "petty_cash_fund.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "petty_cash_fund.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
