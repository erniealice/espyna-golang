package balance

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// DeleteBalanceRepositories groups all repository dependencies
type DeleteBalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// DeleteBalanceServices groups all business service dependencies
type DeleteBalanceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteBalanceUseCase handles the business logic for deleting balances
type DeleteBalanceUseCase struct {
	repositories DeleteBalanceRepositories
	services     DeleteBalanceServices
}

// NewDeleteBalanceUseCase creates a new DeleteBalanceUseCase
func NewDeleteBalanceUseCase(
	repositories DeleteBalanceRepositories,
	services DeleteBalanceServices,
) *DeleteBalanceUseCase {
	return &DeleteBalanceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete balance operation
func (uc *DeleteBalanceUseCase) Execute(ctx context.Context, req *balancepb.DeleteBalanceRequest) (*balancepb.DeleteBalanceResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityBalance, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Balance.DeleteBalance(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("balance with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"balance.errors.not_found",
				map[string]interface{}{"balanceId": req.Data.Id},
				"Student account balance not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteBalanceUseCase) validateInput(ctx context.Context, req *balancepb.DeleteBalanceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.data_required", "balance data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.id_required", "balance ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for balance deletion
func (uc *DeleteBalanceUseCase) validateBusinessRules(req *balancepb.DeleteBalanceRequest) error {
	// Validate balance ID format
	if len(req.Data.Id) < 3 {
		return errors.New("balance ID must be at least 3 characters long")
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of balances with outstanding amounts

	return nil
}
