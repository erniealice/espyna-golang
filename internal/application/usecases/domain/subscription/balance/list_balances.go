package balance

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// ListBalancesRepositories groups all repository dependencies
type ListBalancesRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// ListBalancesServices groups all business service dependencies
type ListBalancesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator // Current: Text translation and localization
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Balance,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.Balance, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "balance.validation.request_required", "request is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for balance listing
func (uc *ListBalancesUseCase) validateBusinessRules(req *balancepb.ListBalancesRequest) error {
	// No specific business rules for listing balances
	return nil
}
