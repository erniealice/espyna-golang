package delegate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// GetDelegateItemPageDataRepositories groups all repository dependencies
type GetDelegateItemPageDataRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// GetDelegateItemPageDataServices groups all business service dependencies
type GetDelegateItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetDelegateItemPageDataUseCase handles the business logic for getting delegate item page data
type GetDelegateItemPageDataUseCase struct {
	repositories GetDelegateItemPageDataRepositories
	services     GetDelegateItemPageDataServices
}

// NewGetDelegateItemPageDataUseCase creates use case with grouped dependencies
func NewGetDelegateItemPageDataUseCase(
	repositories GetDelegateItemPageDataRepositories,
	services GetDelegateItemPageDataServices,
) *GetDelegateItemPageDataUseCase {
	return &GetDelegateItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get delegate item page data operation
func (uc *GetDelegateItemPageDataUseCase) Execute(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) (*delegatepb.GetDelegateItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Delegate.GetDelegateItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load delegate details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetDelegateItemPageDataUseCase) validateInput(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate delegate ID
	if req.DelegateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.delegate_id_required", "[ERR-DEFAULT] Delegate ID is required"))
	}

	// Basic ID format validation
	if len(req.DelegateId) < 3 || len(req.DelegateId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.invalid_delegate_id_format", "[ERR-DEFAULT] Invalid delegate ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.DelegateId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.delegate_id_invalid_characters", "[ERR-DEFAULT] Delegate ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetDelegateItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *delegatepb.GetDelegateItemPageDataRequest) error {
	// Check authorization for viewing specific delegate
	// This would typically involve checking user permissions for the specific delegate
	// For now, we'll allow all authenticated users to view delegate details

	return nil
}
