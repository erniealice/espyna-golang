package balance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// CreateBalanceRepositories groups all repository dependencies
type CreateBalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// CreateBalanceServices groups all business service dependencies
type CreateBalanceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateBalanceUseCase handles the business logic for creating balances
type CreateBalanceUseCase struct {
	repositories CreateBalanceRepositories
	services     CreateBalanceServices
}

// NewCreateBalanceUseCase creates use case with grouped dependencies
func NewCreateBalanceUseCase(
	repositories CreateBalanceRepositories,
	services CreateBalanceServices,
) *CreateBalanceUseCase {
	return &CreateBalanceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create balance operation
func (uc *CreateBalanceUseCase) Execute(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalance, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the create balance operation within a transaction
func (uc *CreateBalanceUseCase) executeWithTransaction(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	var result *balancepb.CreateBalanceResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.creation_failed", "balance creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core create balance operation
func (uc *CreateBalanceUseCase) executeCore(ctx context.Context, req *balancepb.CreateBalanceRequest) (*balancepb.CreateBalanceResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.errors.authorization_failed", "Authorization failed for student account balances [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityBalance, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichBalanceData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Balance.CreateBalance(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateBalanceUseCase) validateInput(ctx context.Context, req *balancepb.CreateBalanceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance.validation.data_required", "balance data is required [DEFAULT]"))
	}
	return nil
}

// enrichBalanceData adds generated fields and audit information
func (uc *CreateBalanceUseCase) enrichBalanceData(balance *balancepb.Balance) error {
	now := time.Now()

	// Generate Balance ID if not provided
	if balance.Id == "" {
		balance.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	balance.DateCreated = &[]int64{now.UnixMilli()}[0]
	balance.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	balance.DateModified = &[]int64{now.UnixMilli()}[0]
	balance.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	balance.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for balances
func (uc *CreateBalanceUseCase) validateBusinessRules(balance *balancepb.Balance) error {
	// Validate amount constraints
	if balance.Amount < 0 {
		return errors.New("balance amount cannot be negative")
	}

	// Additional financial constraints can be added here
	// For example, maximum balance limits, currency validation, etc.

	return nil
}

// Additional validation methods can be added here as needed
