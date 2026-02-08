package balance

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

type GetBalanceItemPageDataRepositories struct {
	Balance balancepb.BalanceDomainServiceServer
}

type GetBalanceItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetBalanceItemPageDataUseCase handles the business logic for getting balance item page data
type GetBalanceItemPageDataUseCase struct {
	repositories GetBalanceItemPageDataRepositories
	services     GetBalanceItemPageDataServices
}

// NewGetBalanceItemPageDataUseCase creates a new GetBalanceItemPageDataUseCase
func NewGetBalanceItemPageDataUseCase(
	repositories GetBalanceItemPageDataRepositories,
	services GetBalanceItemPageDataServices,
) *GetBalanceItemPageDataUseCase {
	return &GetBalanceItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get balance item page data operation
func (uc *GetBalanceItemPageDataUseCase) Execute(
	ctx context.Context,
	req *balancepb.GetBalanceItemPageDataRequest,
) (*balancepb.GetBalanceItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.BalanceId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes balance item page data retrieval within a transaction
func (uc *GetBalanceItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *balancepb.GetBalanceItemPageDataRequest,
) (*balancepb.GetBalanceItemPageDataResponse, error) {
	var result *balancepb.GetBalanceItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"balance.errors.item_page_data_failed",
				"balance item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting balance item page data
func (uc *GetBalanceItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *balancepb.GetBalanceItemPageDataRequest,
) (*balancepb.GetBalanceItemPageDataResponse, error) {
	// Create read request for the balance
	readReq := &balancepb.ReadBalanceRequest{
		Data: &balancepb.Balance{
			Id: req.BalanceId,
		},
	}

	// Retrieve the balance
	readResp, err := uc.repositories.Balance.ReadBalance(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.errors.read_failed",
			"failed to retrieve balance: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.errors.not_found",
			"balance not found",
		))
	}

	// Get the balance (should be only one)
	balance := readResp.Data[0]

	// Validate that we got the expected balance
	if balance.Id != req.BalanceId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.errors.id_mismatch",
			"retrieved balance ID does not match requested ID",
		))
	}

	// Apply financial data validation and processing
	if err := uc.validateFinancialData(ctx, balance); err != nil {
		return nil, err
	}

	// Apply data transformation for optimal frontend consumption
	processedBalance := uc.applyDataTransformation(ctx, balance)

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (client details, subscription details, etc.) if not already populated
	// 2. Apply business rules for financial data visibility/access control
	// 3. Format monetary values for optimal frontend consumption
	// 4. Add audit logging for financial data access
	// 5. Check for balance reconciliation status
	// 6. Apply currency conversion if needed

	return &balancepb.GetBalanceItemPageDataResponse{
		Balance: processedBalance,
		Success: true,
	}, nil
}

// validateInput validates the input request
func (uc *GetBalanceItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *balancepb.GetBalanceItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.validation.request_required",
			"request is required",
		))
	}

	if req.BalanceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.validation.id_required",
			"balance ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading balance item page data
func (uc *GetBalanceItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	balanceId string,
) error {
	// Validate balance ID format
	if len(balanceId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.validation.id_too_short",
			"balance ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this balance
	// - Validate balance belongs to the current user's organization
	// - Check if balance is in a state that allows viewing
	// - Rate limiting for balance access
	// - Audit logging requirements for financial data access
	// - Compliance checks for financial data access

	return nil
}

// validateFinancialData validates the financial integrity of the balance data
func (uc *GetBalanceItemPageDataUseCase) validateFinancialData(
	ctx context.Context,
	balance *balancepb.Balance,
) error {
	// Validate amount precision (important for financial data)
	if balance.Amount != 0 {
		// Check for invalid floating point values
		if balance.Amount != balance.Amount { // NaN check
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"balance.validation.invalid_amount",
				"balance amount is not a valid number",
			))
		}

		// Check for infinite values
		if balance.Amount > 1e15 || balance.Amount < -1e15 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.TranslationService,
				"balance.validation.amount_out_of_range",
				"balance amount is out of valid range",
			))
		}
	}

	// Validate currency if present
	if balance.Currency != "" && len(balance.Currency) != 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.validation.invalid_currency",
			"currency code must be 3 characters",
		))
	}

	// Validate balance type
	validBalanceTypes := map[string]bool{
		"credit":  true,
		"debit":   true,
		"pending": true,
		"hold":    true,
		"refund":  true,
	}
	if balance.BalanceType != "" && !validBalanceTypes[balance.BalanceType] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"balance.validation.invalid_balance_type",
			"invalid balance type",
		))
	}

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetBalanceItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	balance *balancepb.Balance,
) *balancepb.Balance {
	// Create a copy to avoid modifying the original
	processedBalance := &balancepb.Balance{
		Id:                 balance.Id,
		Amount:             balance.Amount,
		DateCreated:        balance.DateCreated,
		DateCreatedString:  balance.DateCreatedString,
		DateModified:       balance.DateModified,
		DateModifiedString: balance.DateModifiedString,
		Active:             balance.Active,
		ClientId:           balance.ClientId,
		SubscriptionId:     balance.SubscriptionId,
		Currency:           balance.Currency,
		BalanceType:        balance.BalanceType,
	}

	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting monetary amounts with proper decimal precision
	// - Converting dates to appropriate timezone
	// - Computing derived fields (e.g., absolute amount, formatted currency)
	// - Applying localization for currency display
	// - Sanitizing sensitive financial data based on user permissions
	// - Adding computed status fields (e.g., "overdue", "pending_reconciliation")

	// Example: Ensure currency defaults to USD if not set
	if processedBalance.Currency == "" {
		processedBalance.Currency = "USD"
	}

	// Example: Ensure balance type is set
	if processedBalance.BalanceType == "" {
		if processedBalance.Amount >= 0 {
			processedBalance.BalanceType = "credit"
		} else {
			processedBalance.BalanceType = "debit"
		}
	}

	return processedBalance
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like client and subscription details
// This would be called from executeCore if needed
func (uc *GetBalanceItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	balance *balancepb.Balance,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to client and subscription repositories
	// to populate additional context for the balance

	// Example implementation would be:
	// if balance.ClientId != "" {
	//     // Load client data for additional context
	// }
	// if balance.SubscriptionId != "" {
	//     // Load subscription data for additional context
	// }

	return nil
}

// checkAccessPermissions validates user has permission to access this balance
func (uc *GetBalanceItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	balanceId string,
) error {
	// TODO: Implement proper access control for financial data
	// This could involve:
	// - Checking user role/permissions for financial data access
	// - Validating balance belongs to user's organization
	// - Applying multi-tenant access controls
	// - Checking compliance requirements for financial data access
	// - Audit logging for financial data access

	return nil
}

// validateBalanceReconciliation checks if the balance is properly reconciled
func (uc *GetBalanceItemPageDataUseCase) validateBalanceReconciliation(
	ctx context.Context,
	balance *balancepb.Balance,
) error {
	// TODO: Implement balance reconciliation validation
	// This could involve:
	// - Checking if balance matches expected values
	// - Validating against transaction history
	// - Checking for pending reconciliation items
	// - Alerting for discrepancies

	return nil
}
