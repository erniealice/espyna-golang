package balance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// UpdateBalanceRepositories groups all repository dependencies
type UpdateBalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// UpdateBalanceServices groups all business service dependencies
type UpdateBalanceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateBalanceUseCase handles the business logic for updating balances
type UpdateBalanceUseCase struct {
	repositories UpdateBalanceRepositories
	services     UpdateBalanceServices
}

// NewUpdateBalanceUseCase creates a new UpdateBalanceUseCase
func NewUpdateBalanceUseCase(
	repositories UpdateBalanceRepositories,
	services UpdateBalanceServices,
) *UpdateBalanceUseCase {
	return &UpdateBalanceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update balance operation
func (uc *UpdateBalanceUseCase) Execute(ctx context.Context, req *balancepb.UpdateBalanceRequest) (*balancepb.UpdateBalanceResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "authorization.errors.access_denied", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityBalance, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "authorization.errors.access_denied", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "authorization.errors.access_denied", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichBalanceData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Balance.UpdateBalance(ctx, req)
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
func (uc *UpdateBalanceUseCase) validateInput(ctx context.Context, req *balancepb.UpdateBalanceRequest) error {
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

// enrichBalanceData adds audit information for updates
func (uc *UpdateBalanceUseCase) enrichBalanceData(balance *balancepb.Balance) error {
	now := time.Now()

	// Update modification timestamp
	balance.DateModified = &[]int64{now.UnixMilli()}[0]
	balance.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for balance updates
func (uc *UpdateBalanceUseCase) validateBusinessRules(balance *balancepb.Balance) error {
	// Validate amount constraints
	if balance.Amount < 0 {
		return errors.New("balance amount cannot be negative")
	}

	// Validate balance ID format
	if len(balance.Id) < 3 {
		return errors.New("balance ID must be at least 3 characters long")
	}

	// Additional financial constraints can be added here
	// For example, maximum balance limits, currency validation, etc.

	return nil
}
