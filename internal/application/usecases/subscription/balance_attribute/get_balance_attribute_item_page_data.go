package balance_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// GetBalanceAttributeItemPageDataRepositories groups all repository dependencies
type GetBalanceAttributeItemPageDataRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
}

// GetBalanceAttributeItemPageDataServices groups all business service dependencies
type GetBalanceAttributeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetBalanceAttributeItemPageDataUseCase handles the business logic for getting balance attribute item page data
type GetBalanceAttributeItemPageDataUseCase struct {
	repositories GetBalanceAttributeItemPageDataRepositories
	services     GetBalanceAttributeItemPageDataServices
}

// NewGetBalanceAttributeItemPageDataUseCase creates a new GetBalanceAttributeItemPageDataUseCase
func NewGetBalanceAttributeItemPageDataUseCase(
	repositories GetBalanceAttributeItemPageDataRepositories,
	services GetBalanceAttributeItemPageDataServices,
) *GetBalanceAttributeItemPageDataUseCase {
	return &GetBalanceAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get balance attribute item page data operation
func (uc *GetBalanceAttributeItemPageDataUseCase) Execute(ctx context.Context, req *balanceattributepb.GetBalanceAttributeItemPageDataRequest) (*balanceattributepb.GetBalanceAttributeItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalanceAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.GetBalanceAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.item_page_data_failed", "Failed to retrieve balance attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetBalanceAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *balanceattributepb.GetBalanceAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required for balance attributes [DEFAULT]"))
	}

	// Validate balance attribute ID - uses direct field req.BalanceAttributeId
	if strings.TrimSpace(req.BalanceAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.id_required", "Balance attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.BalanceAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.id_too_short", "Balance attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
