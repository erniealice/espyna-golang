package price_list

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

// DeletePriceListRepositories groups all repository dependencies
type DeletePriceListRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer // Primary entity repository
}

// DeletePriceListServices groups all business service dependencies
type DeletePriceListServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// DeletePriceListUseCase handles the business logic for deleting price lists
type DeletePriceListUseCase struct {
	repositories DeletePriceListRepositories
	services     DeletePriceListServices
}

// NewDeletePriceListUseCase creates a new DeletePriceListUseCase
func NewDeletePriceListUseCase(
	repositories DeletePriceListRepositories,
	services DeletePriceListServices,
) *DeletePriceListUseCase {
	return &DeletePriceListUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete price list operation
func (uc *DeletePriceListUseCase) Execute(ctx context.Context, req *pricelistpb.DeletePriceListRequest) (*pricelistpb.DeletePriceListResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PriceList, entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price list deletion within a transaction
func (uc *DeletePriceListUseCase) executeWithTransaction(ctx context.Context, req *pricelistpb.DeletePriceListRequest) (*pricelistpb.DeletePriceListResponse, error) {
	var result *pricelistpb.DeletePriceListResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a price list
func (uc *DeletePriceListUseCase) executeCore(ctx context.Context, req *pricelistpb.DeletePriceListRequest) (*pricelistpb.DeletePriceListResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.PriceList, entityid.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceList.DeletePriceList(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.errors.not_found", "Price list with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeletePriceListUseCase) validateInput(ctx context.Context, req *pricelistpb.DeletePriceListRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.validation.data_required", "Price List data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_list.validation.id_required", "Price List ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for price list deletion
func (uc *DeletePriceListUseCase) validateBusinessRules(ctx context.Context, req *pricelistpb.DeletePriceListRequest) error {
	return nil
}
