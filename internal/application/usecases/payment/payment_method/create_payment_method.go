package payment_method

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

// CreatePaymentMethodRepositories groups all repository dependencies
type CreatePaymentMethodRepositories struct {
	PaymentMethod paymentmethodpb.PaymentMethodDomainServiceServer // Primary entity repository
}

// CreatePaymentMethodServices groups all business service dependencies
type CreatePaymentMethodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePaymentMethodUseCase handles the business logic for creating payment methods
type CreatePaymentMethodUseCase struct {
	repositories CreatePaymentMethodRepositories
	services     CreatePaymentMethodServices
}

// NewCreatePaymentMethodUseCase creates use case with grouped dependencies
func NewCreatePaymentMethodUseCase(
	repositories CreatePaymentMethodRepositories,
	services CreatePaymentMethodServices,
) *CreatePaymentMethodUseCase {
	return &CreatePaymentMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payment method operation
func (uc *CreatePaymentMethodUseCase) Execute(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPaymentMethod, ports.ActionCreate)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentMethodData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment method creation within a transaction
func (uc *CreatePaymentMethodUseCase) executeWithTransaction(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	var result *paymentmethodpb.CreatePaymentMethodResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_method.errors.creation_failed", "")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for creating a payment method
func (uc *CreatePaymentMethodUseCase) executeCore(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	// Delegate to repository
	return uc.repositories.PaymentMethod.CreatePaymentMethod(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePaymentMethodUseCase) validateInput(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_method.validation.data_required", ""))
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

// enrichPaymentMethodData adds generated fields and audit information
func (uc *CreatePaymentMethodUseCase) enrichPaymentMethodData(paymentMethod *paymentmethodpb.PaymentMethod) error {
	now := time.Now()

	// Generate Payment Method ID if not provided
	if paymentMethod.Id == "" {
		paymentMethod.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	paymentMethod.DateCreated = &[]int64{now.UnixMilli()}[0]
	paymentMethod.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentMethod.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentMethod.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentMethod.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for payment methods
func (uc *CreatePaymentMethodUseCase) validateBusinessRules(ctx context.Context, paymentMethod *paymentmethodpb.PaymentMethod) error {
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

	return nil
}

// validateCardDetails validates credit/debit card information
func (uc *CreatePaymentMethodUseCase) validateCardDetails(ctx context.Context, card *paymentmethodpb.CardDetails) error {
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
func (uc *CreatePaymentMethodUseCase) validateBankAccountDetails(ctx context.Context, bank *paymentmethodpb.BankAccountDetails) error {
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
