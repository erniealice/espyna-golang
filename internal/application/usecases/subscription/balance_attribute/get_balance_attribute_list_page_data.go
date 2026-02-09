package balance_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// GetBalanceAttributeListPageDataRepositories groups all repository dependencies
type GetBalanceAttributeListPageDataRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
}

// GetBalanceAttributeListPageDataServices groups all business service dependencies
type GetBalanceAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetBalanceAttributeListPageDataUseCase handles the business logic for getting balance attribute list page data
type GetBalanceAttributeListPageDataUseCase struct {
	repositories GetBalanceAttributeListPageDataRepositories
	services     GetBalanceAttributeListPageDataServices
}

// NewGetBalanceAttributeListPageDataUseCase creates a new GetBalanceAttributeListPageDataUseCase
func NewGetBalanceAttributeListPageDataUseCase(
	repositories GetBalanceAttributeListPageDataRepositories,
	services GetBalanceAttributeListPageDataServices,
) *GetBalanceAttributeListPageDataUseCase {
	return &GetBalanceAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get balance attribute list page data operation
func (uc *GetBalanceAttributeListPageDataUseCase) Execute(ctx context.Context, req *balanceattributepb.GetBalanceAttributeListPageDataRequest) (*balanceattributepb.GetBalanceAttributeListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalanceAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.GetBalanceAttributeListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetBalanceAttributeListPageDataUseCase) validateInput(ctx context.Context, req *balanceattributepb.GetBalanceAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
