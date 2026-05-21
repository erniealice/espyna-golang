package payment_term

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// UpdatePaymentTermRepositories groups all repository dependencies
type UpdatePaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// UpdatePaymentTermServices groups all business service dependencies
type UpdatePaymentTermServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePaymentTermUseCase handles the business logic for updating a payment term
type UpdatePaymentTermUseCase struct {
	repositories UpdatePaymentTermRepositories
	services     UpdatePaymentTermServices
}

// NewUpdatePaymentTermUseCase creates use case with grouped dependencies
func NewUpdatePaymentTermUseCase(
	repositories UpdatePaymentTermRepositories,
	services UpdatePaymentTermServices,
) *UpdatePaymentTermUseCase {
	return &UpdatePaymentTermUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdatePaymentTermUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdatePaymentTermUseCase with grouped parameters instead
func NewUpdatePaymentTermUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *UpdatePaymentTermUseCase {
	repositories := UpdatePaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := UpdatePaymentTermServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdatePaymentTermUseCase(repositories, services)
}

// Execute performs the update payment term operation
func (uc *UpdatePaymentTermUseCase) Execute(ctx context.Context, req *paymenttermpb.UpdatePaymentTermRequest) (*paymenttermpb.UpdatePaymentTermResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.request_required", "Request is required for payment terms [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.id_required", "Payment term ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.name_required", "Payment term name is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.PaymentTerm.UpdatePaymentTerm(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.errors.update_failed", "Payment term update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
