package balance

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	balancepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/balance"
)

// ListBalancesRepositories groups all repository dependencies
type ListBalancesRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// ListBalancesServices groups all business service dependencies
type ListBalancesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService   // Current: Text translation and localization
}

// ListBalancesUseCase handles the business logic for listing balances
type ListBalancesUseCase struct {
	repositories ListBalancesRepositories
	services     ListBalancesServices
}

// NewListBalancesUseCase creates a new ListBalancesUseCase
func NewListBalancesUseCase(
	repositories ListBalancesRepositories,
	services ListBalancesServices,
) *ListBalancesUseCase {
	return &ListBalancesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list balances operation
func (uc *ListBalancesUseCase) Execute(ctx context.Context, req *balancepb.ListBalancesRequest) (*balancepb.ListBalancesResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityBalance, ports.ActionRead)
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
	resp, err := uc.repositories.Balance.ListBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListBalancesUseCase) validateInput(ctx context.Context, req *balancepb.ListBalancesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.request_required", "request is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for balance listing
func (uc *ListBalancesUseCase) validateBusinessRules(req *balancepb.ListBalancesRequest) error {
	// No specific business rules for listing balances
	return nil
}
