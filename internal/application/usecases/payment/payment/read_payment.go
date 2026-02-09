package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
)

// ReadPaymentRepositories groups all repository dependencies
type ReadPaymentRepositories struct {
	Payment paymentpb.PaymentDomainServiceServer // Primary entity repository
}

// ReadPaymentServices groups all business service dependencies
type ReadPaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPaymentUseCase handles the business logic for reading payments
type ReadPaymentUseCase struct {
	repositories ReadPaymentRepositories
	services     ReadPaymentServices
}

// NewReadPaymentUseCase creates use case with grouped dependencies
func NewReadPaymentUseCase(
	repositories ReadPaymentRepositories,
	services ReadPaymentServices,
) *ReadPaymentUseCase {
	return &ReadPaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read payment operation
func (uc *ReadPaymentUseCase) Execute(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPayment, ports.ActionRead); err != nil {
		return nil, err
	}


	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic validation
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

// executeWithTransaction executes payment reading within a transaction
func (uc *ReadPaymentUseCase) executeWithTransaction(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	var result *paymentpb.ReadPaymentResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment.errors.read_failed", "")
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

// executeCore contains the core business logic for reading a payment
func (uc *ReadPaymentUseCase) executeCore(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	// Call repository
	resp, err := uc.repositories.Payment.ReadPayment(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.errors.not_found", "")
		translatedError = strings.ReplaceAll(translatedError, "{paymentId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPaymentUseCase) validateInput(ctx context.Context, req *paymentpb.ReadPaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for reading payments
func (uc *ReadPaymentUseCase) validateBusinessRules(ctx context.Context, payment *paymentpb.Payment) error {

	// Validate payment ID format
	if len(payment.Id) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment.validation.id_too_short", ""))
	}

	// Financial security: Ensure proper access control for payment data
	// Additional authorization checks would be implemented here in a real system

	return nil
}
