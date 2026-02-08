package price_list

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

// ReadPriceListRepositories groups all repository dependencies
type ReadPriceListRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer // Primary entity repository
}

// ReadPriceListServices groups all business service dependencies
type ReadPriceListServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPriceListUseCase handles the business logic for reading a price list
type ReadPriceListUseCase struct {
	repositories ReadPriceListRepositories
	services     ReadPriceListServices
}

// NewReadPriceListUseCase creates use case with grouped dependencies
func NewReadPriceListUseCase(
	repositories ReadPriceListRepositories,
	services ReadPriceListServices,
) *ReadPriceListUseCase {
	return &ReadPriceListUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read price list operation
func (uc *ReadPriceListUseCase) Execute(ctx context.Context, req *pricelistpb.ReadPriceListRequest) (*pricelistpb.ReadPriceListResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceList, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.authorization_failed", "Authorization failed for price lists [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceList.ReadPriceList(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.errors.not_found", "Price list with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPriceListUseCase) validateInput(ctx context.Context, req *pricelistpb.ReadPriceListRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.data_required", "Price List data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_list.validation.id_required", "Price List ID is required [DEFAULT]"))
	}
	return nil
}
