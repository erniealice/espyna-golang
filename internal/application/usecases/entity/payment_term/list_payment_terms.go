package payment_term

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// ListPaymentTermsRepositories groups all repository dependencies
type ListPaymentTermsRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// ListPaymentTermsServices groups all business service dependencies
type ListPaymentTermsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPaymentTermsUseCase handles the business logic for listing payment terms
type ListPaymentTermsUseCase struct {
	repositories ListPaymentTermsRepositories
	services     ListPaymentTermsServices
}

// NewListPaymentTermsUseCase creates use case with grouped dependencies
func NewListPaymentTermsUseCase(
	repositories ListPaymentTermsRepositories,
	services ListPaymentTermsServices,
) *ListPaymentTermsUseCase {
	return &ListPaymentTermsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListPaymentTermsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListPaymentTermsUseCase with grouped parameters instead
func NewListPaymentTermsUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *ListPaymentTermsUseCase {
	repositories := ListPaymentTermsRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := ListPaymentTermsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListPaymentTermsUseCase(repositories, services)
}

// Execute performs the list payment terms operation
func (uc *ListPaymentTermsUseCase) Execute(ctx context.Context, req *paymenttermpb.ListPaymentTermsRequest) (*paymenttermpb.ListPaymentTermsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &paymenttermpb.ListPaymentTermsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.PaymentTerm.ListPaymentTerms(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_term.errors.list_failed", "Failed to retrieve payment terms [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
