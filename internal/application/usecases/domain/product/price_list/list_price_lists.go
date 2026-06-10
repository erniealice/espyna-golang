package price_list

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

// ListPriceListsRepositories groups all repository dependencies
type ListPriceListsRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer // Primary entity repository
}

// ListPriceListsServices groups all business service dependencies
type ListPriceListsServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ListPriceListsUseCase handles the business logic for listing price lists
type ListPriceListsUseCase struct {
	repositories ListPriceListsRepositories
	services     ListPriceListsServices
}

// NewListPriceListsUseCase creates a new ListPriceListsUseCase
func NewListPriceListsUseCase(
	repositories ListPriceListsRepositories,
	services ListPriceListsServices,
) *ListPriceListsUseCase {
	return &ListPriceListsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list price lists operation
func (uc *ListPriceListsUseCase) Execute(ctx context.Context, req *pricelistpb.ListPriceListsRequest) (*pricelistpb.ListPriceListsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PriceList, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.PriceList, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceList.ListPriceLists(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.list_failed", "Failed to retrieve price lists [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListPriceListsUseCase) validateInput(ctx context.Context, req *pricelistpb.ListPriceListsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
