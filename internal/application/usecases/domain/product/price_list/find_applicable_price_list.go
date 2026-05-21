package price_list

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
)

// FindApplicablePriceListRepositories groups all repository dependencies
type FindApplicablePriceListRepositories struct {
	PriceList pricelistpb.PriceListDomainServiceServer
}

// FindApplicablePriceListServices groups all business service dependencies
type FindApplicablePriceListServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// FindApplicablePriceListUseCase handles the business logic for finding the
// applicable price list for a given location and date.
type FindApplicablePriceListUseCase struct {
	repositories FindApplicablePriceListRepositories
	services     FindApplicablePriceListServices
}

// NewFindApplicablePriceListUseCase creates a new FindApplicablePriceListUseCase
func NewFindApplicablePriceListUseCase(
	repositories FindApplicablePriceListRepositories,
	services FindApplicablePriceListServices,
) *FindApplicablePriceListUseCase {
	return &FindApplicablePriceListUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute finds the active price list applicable to the given location and date.
func (uc *FindApplicablePriceListUseCase) Execute(
	ctx context.Context,
	req *pricelistpb.FindApplicablePriceListRequest,
) (*pricelistpb.FindApplicablePriceListResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceList, ports.ActionList); err != nil {
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

	return uc.executeCore(ctx, req)
}

// executeWithTransaction wraps the core logic in a transaction.
func (uc *FindApplicablePriceListUseCase) executeWithTransaction(
	ctx context.Context,
	req *pricelistpb.FindApplicablePriceListRequest,
) (*pricelistpb.FindApplicablePriceListResponse, error) {
	var result *pricelistpb.FindApplicablePriceListResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"price_list.errors.find_applicable_failed",
				"find applicable price list failed: %w",
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

// executeCore delegates to the repository.
func (uc *FindApplicablePriceListUseCase) executeCore(
	ctx context.Context,
	req *pricelistpb.FindApplicablePriceListRequest,
) (*pricelistpb.FindApplicablePriceListResponse, error) {
	resp, err := uc.repositories.PriceList.FindApplicablePriceList(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.find_applicable_failed",
			"failed to find applicable price list: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request.
func (uc *FindApplicablePriceListUseCase) validateInput(
	ctx context.Context,
	req *pricelistpb.FindApplicablePriceListRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.request_required",
			"request is required",
		))
	}
	if req.LocationId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.location_id_required",
			"location_id is required",
		))
	}
	if req.Date == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.date_required",
			"date is required",
		))
	}
	return nil
}
