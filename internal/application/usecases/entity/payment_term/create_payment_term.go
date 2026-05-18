package payment_term

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// CreatePaymentTermRepositories groups all repository dependencies
type CreatePaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// CreatePaymentTermServices groups all business service dependencies
type CreatePaymentTermServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePaymentTermUseCase handles the business logic for creating payment terms
type CreatePaymentTermUseCase struct {
	repositories CreatePaymentTermRepositories
	services     CreatePaymentTermServices
}

// NewCreatePaymentTermUseCase creates use case with grouped dependencies
func NewCreatePaymentTermUseCase(
	repositories CreatePaymentTermRepositories,
	services CreatePaymentTermServices,
) *CreatePaymentTermUseCase {
	return &CreatePaymentTermUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreatePaymentTermUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreatePaymentTermUseCase with grouped parameters instead
func NewCreatePaymentTermUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *CreatePaymentTermUseCase {
	repositories := CreatePaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := CreatePaymentTermServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreatePaymentTermUseCase(repositories, services)
}

// Execute performs the create payment term operation
func (uc *CreatePaymentTermUseCase) Execute(ctx context.Context, req *paymenttermpb.CreatePaymentTermRequest) (*paymenttermpb.CreatePaymentTermResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.request_required", "Request is required for payment terms [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedPaymentTerm := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedPaymentTerm)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedPaymentTerm)
}

// executeWithTransaction executes payment term creation within a transaction
func (uc *CreatePaymentTermUseCase) executeWithTransaction(ctx context.Context, enrichedPaymentTerm *paymenttermpb.PaymentTerm) (*paymenttermpb.CreatePaymentTermResponse, error) {
	var result *paymenttermpb.CreatePaymentTermResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedPaymentTerm)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_term.errors.creation_failed", "Payment term creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a payment term
func (uc *CreatePaymentTermUseCase) executeCore(ctx context.Context, enrichedPaymentTerm *paymenttermpb.PaymentTerm) (*paymenttermpb.CreatePaymentTermResponse, error) {
	return uc.repositories.PaymentTerm.CreatePaymentTerm(ctx, &paymenttermpb.CreatePaymentTermRequest{
		Data: enrichedPaymentTerm,
	})
}

// applyBusinessLogic applies business rules and returns enriched payment term
func (uc *CreatePaymentTermUseCase) applyBusinessLogic(paymentTerm *paymenttermpb.PaymentTerm) *paymenttermpb.PaymentTerm {
	now := time.Now()

	// Business logic: Generate PaymentTerm ID if not provided
	if paymentTerm.Id == "" {
		if uc.services.IDService != nil {
			paymentTerm.Id = uc.services.IDService.GenerateID()
		} else {
			paymentTerm.Id = fmt.Sprintf("payment_term-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new payment terms
	paymentTerm.Active = true

	// Business logic: Set creation audit fields
	paymentTerm.DateCreated = &[]int64{now.UnixMilli()}[0]
	paymentTerm.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentTerm.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentTerm.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return paymentTerm
}

// validateBusinessRules enforces business constraints
func (uc *CreatePaymentTermUseCase) validateBusinessRules(ctx context.Context, paymentTerm *paymenttermpb.PaymentTerm) error {
	// Business rule: Required data validation
	if paymentTerm == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.data_required", "Payment term data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if paymentTerm.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.name_required", "Payment term name is required [DEFAULT]"))
	}

	// Business rule: Code is required
	if paymentTerm.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.code_required", "Payment term code is required [DEFAULT]"))
	}

	return nil
}
