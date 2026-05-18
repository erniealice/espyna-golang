package payment_term

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// ReadPaymentTermRepositories groups all repository dependencies
type ReadPaymentTermRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// ReadPaymentTermServices groups all business service dependencies
type ReadPaymentTermServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPaymentTermUseCase handles the business logic for reading a payment term
type ReadPaymentTermUseCase struct {
	repositories ReadPaymentTermRepositories
	services     ReadPaymentTermServices
}

// NewReadPaymentTermUseCase creates use case with grouped dependencies
func NewReadPaymentTermUseCase(
	repositories ReadPaymentTermRepositories,
	services ReadPaymentTermServices,
) *ReadPaymentTermUseCase {
	return &ReadPaymentTermUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadPaymentTermUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadPaymentTermUseCase with grouped parameters instead
func NewReadPaymentTermUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *ReadPaymentTermUseCase {
	repositories := ReadPaymentTermRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := ReadPaymentTermServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadPaymentTermUseCase(repositories, services)
}

// Execute performs the read payment term operation
func (uc *ReadPaymentTermUseCase) Execute(ctx context.Context, req *paymenttermpb.ReadPaymentTermRequest) (*paymenttermpb.ReadPaymentTermResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionRead); err != nil {
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
	resp, err := uc.repositories.PaymentTerm.ReadPaymentTerm(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.errors.not_found", "Payment term with ID \"{paymentTermId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{paymentTermId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
