package balance

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// ReadBalanceRepositories groups all repository dependencies
type ReadBalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// ReadBalanceServices groups all business service dependencies
type ReadBalanceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadBalanceUseCase handles the business logic for reading balances
type ReadBalanceUseCase struct {
	repositories ReadBalanceRepositories
	services     ReadBalanceServices
}

// NewReadBalanceUseCase creates a new ReadBalanceUseCase
func NewReadBalanceUseCase(
	repositories ReadBalanceRepositories,
	services ReadBalanceServices,
) *ReadBalanceUseCase {
	return &ReadBalanceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read balance operation
func (uc *ReadBalanceUseCase) Execute(ctx context.Context, req *balancepb.ReadBalanceRequest) (*balancepb.ReadBalanceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalance, ports.ActionRead); err != nil {
		return nil, err
	}


	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Balance.ReadBalance(ctx, req)
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
func (uc *ReadBalanceUseCase) validateInput(ctx context.Context, req *balancepb.ReadBalanceRequest) error {
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

// validateBusinessRules enforces business constraints for reading balances
func (uc *ReadBalanceUseCase) validateBusinessRules(balance *balancepb.Balance) error {
	// Validate balance ID format
	if len(balance.Id) < 3 {
		return errors.New("balance ID must be at least 3 characters long")
	}

	return nil
}
