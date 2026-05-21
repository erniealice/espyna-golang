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

// DeletePaymentTermRepositories groups all repository dependencies
type DeletePaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// DeletePaymentTermServices groups all business service dependencies
type DeletePaymentTermServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePaymentTermUseCase handles the business logic for deleting a payment term
type DeletePaymentTermUseCase struct {
	repositories DeletePaymentTermRepositories
	services     DeletePaymentTermServices
}

// NewDeletePaymentTermUseCase creates use case with grouped dependencies
func NewDeletePaymentTermUseCase(
	repositories DeletePaymentTermRepositories,
	services DeletePaymentTermServices,
) *DeletePaymentTermUseCase {
	return &DeletePaymentTermUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeletePaymentTermUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeletePaymentTermUseCase with grouped parameters instead
func NewDeletePaymentTermUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *DeletePaymentTermUseCase {
	repositories := DeletePaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := DeletePaymentTermServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeletePaymentTermUseCase(repositories, services)
}

// Execute performs the delete payment term operation
func (uc *DeletePaymentTermUseCase) Execute(ctx context.Context, req *paymenttermpb.DeletePaymentTermRequest) (*paymenttermpb.DeletePaymentTermResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.request_required", "Request is required for payment terms [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.validation.id_required", "Payment term ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.PaymentTerm.DeletePaymentTerm(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.errors.deletion_failed", "Payment term deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
