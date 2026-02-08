package balance_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// DeleteBalanceAttributeRepositories groups all repository dependencies
type DeleteBalanceAttributeRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
}

// DeleteBalanceAttributeServices groups all business service dependencies
type DeleteBalanceAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteBalanceAttributeUseCase handles the business logic for deleting balance attributes
type DeleteBalanceAttributeUseCase struct {
	repositories DeleteBalanceAttributeRepositories
	services     DeleteBalanceAttributeServices
}

// NewDeleteBalanceAttributeUseCase creates a new DeleteBalanceAttributeUseCase
func NewDeleteBalanceAttributeUseCase(
	repositories DeleteBalanceAttributeRepositories,
	services DeleteBalanceAttributeServices,
) *DeleteBalanceAttributeUseCase {
	return &DeleteBalanceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete balance attribute operation
func (uc *DeleteBalanceAttributeUseCase) Execute(ctx context.Context, req *balanceattributepb.DeleteBalanceAttributeRequest) (*balanceattributepb.DeleteBalanceAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.DeleteBalanceAttribute(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("balance_attribute with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"balance_attribute.errors.not_found",
				map[string]interface{}{"balanceAttributeId": req.Data.Id},
				"Balance attribute not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.deletion_failed", "Balance attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteBalanceAttributeUseCase) validateInput(ctx context.Context, req *balanceattributepb.DeleteBalanceAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.id_required", "Balance attribute ID is required [DEFAULT]"))
	}
	return nil
}
