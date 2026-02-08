package payment_method

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

// UpdatePaymentMethodRepositories groups all repository dependencies
type UpdatePaymentMethodRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// UpdatePaymentMethodServices groups all business service dependencies
type UpdatePaymentMethodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePaymentMethodUseCase handles the business logic for updating payment methods
type UpdatePaymentMethodUseCase struct {
	repositories UpdatePaymentMethodRepositories
	services     UpdatePaymentMethodServices
}

// NewUpdatePaymentMethodUseCase creates use case with grouped dependencies
func NewUpdatePaymentMethodUseCase(
	repositories UpdatePaymentMethodRepositories,
	services UpdatePaymentMethodServices,
) *UpdatePaymentMethodUseCase {
	return &UpdatePaymentMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update payment method operation
func (uc *UpdatePaymentMethodUseCase) Execute(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentMethod, ports.ActionUpdate)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentMethodData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Transaction handling
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Core execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment method update within a transaction
func (uc *UpdatePaymentMethodUseCase) executeWithTransaction(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	var result *paymentmethodpb.UpdatePaymentMethodResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("payment method update failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *UpdatePaymentMethodUseCase) executeCore(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Delegate to repository
	return uc.repositories.PaymentMethod.UpdatePaymentMethod(ctx, req)
}

// validateInput validates the input request
func (uc *UpdatePaymentMethodUseCase) validateInput(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.id_required", ""))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.name_required", ""))
	}

	// Validate method details
	if req.Data.GetCard() == nil && req.Data.GetBankAccount() == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.details_required", ""))
	}

	return nil
}

// enrichPaymentMethodData adds updated audit information
func (uc *UpdatePaymentMethodUseCase) enrichPaymentMethodData(paymentMethod *paymentmethodpb.PaymentMethod) error {
	now := time.Now()

	// Update modification timestamp
	paymentMethod.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentMethod.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for payment method updates
func (uc *UpdatePaymentMethodUseCase) validateBusinessRules(ctx context.Context, paymentMethod *paymentmethodpb.PaymentMethod) error {
	// Validate payment method ID format
	if len(paymentMethod.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.id_too_short", ""))
	}

	// Validate payment method name length
	if len(paymentMethod.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.name_too_short", ""))
	}

	if len(paymentMethod.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.name_too_long", ""))
	}

	// Validate card details if provided
	if cardDetails := paymentMethod.GetCard(); cardDetails != nil {
		if err := uc.validateCardDetails(ctx, cardDetails); err != nil {
			return err
		}
	}

	// Validate bank account details if provided
	if bankDetails := paymentMethod.GetBankAccount(); bankDetails != nil {
		if err := uc.validateBankAccountDetails(ctx, bankDetails); err != nil {
			return err
		}
	}

	// Financial constraint: Cannot update payment methods that are currently being used in active transactions
	// This would typically check for active payments using this method

	return nil
}

// validateCardDetails validates credit/debit card information
func (uc *UpdatePaymentMethodUseCase) validateCardDetails(ctx context.Context, card *paymentmethodpb.CardDetails) error {
	// Validate card type
	validCardTypes := map[string]bool{
		"Visa": true, "MasterCard": true, "American Express": true,
		"Discover": true, "JCB": true, "Diners Club": true,
	}
	if !validCardTypes[card.CardType] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.invalid_card_type", ""))
	}

	// Validate last four digits
	if len(card.LastFourDigits) != 4 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.last_four_digits_length", ""))
	}

	// Ensure last four digits are numeric
	digitRegex := regexp.MustCompile(`^\d{4}$`)
	if !digitRegex.MatchString(card.LastFourDigits) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.last_four_digits_numeric", ""))
	}

	// Validate expiry date
	currentYear := time.Now().Year()
	if card.ExpiryYear < int32(currentYear) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.expiry_year_past", ""))
	}

	if card.ExpiryMonth < 1 || card.ExpiryMonth > 12 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.expiry_month_invalid", ""))
	}

	// Check if card is expired (current year, past month)
	if card.ExpiryYear == int32(currentYear) && card.ExpiryMonth < int32(time.Now().Month()) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.card_expired", ""))
	}

	return nil
}

// validateBankAccountDetails validates bank account information
func (uc *UpdatePaymentMethodUseCase) validateBankAccountDetails(ctx context.Context, bank *paymentmethodpb.BankAccountDetails) error {
	// Validate bank name
	if len(bank.BankName) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.bank_name_too_short", ""))
	}

	if len(bank.BankName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.bank_name_too_long", ""))
	}

	// Validate last four digits
	if len(bank.LastFourDigits) != 4 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.last_four_digits_length", ""))
	}

	// Ensure last four digits are numeric
	digitRegex := regexp.MustCompile(`^\d{4}$`)
	if !digitRegex.MatchString(bank.LastFourDigits) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.last_four_digits_numeric", ""))
	}

	return nil
}

// Additional validation methods can be added here as needed
