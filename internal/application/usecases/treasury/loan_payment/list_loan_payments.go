package loanpayment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// ListLoanPaymentsRepositories groups all repository dependencies.
type ListLoanPaymentsRepositories struct {
	LoanPayment loanpaymentpb.LoanPaymentDomainServiceServer
}

// ListLoanPaymentsServices groups all business service dependencies.
type ListLoanPaymentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListLoanPaymentsUseCase handles the business logic for listing loan payments.
type ListLoanPaymentsUseCase struct {
	repositories ListLoanPaymentsRepositories
	services     ListLoanPaymentsServices
}

// NewListLoanPaymentsUseCase creates the use case with grouped dependencies.
func NewListLoanPaymentsUseCase(
	repositories ListLoanPaymentsRepositories,
	services ListLoanPaymentsServices,
) *ListLoanPaymentsUseCase {
	return &ListLoanPaymentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list loan payments operation.
func (uc *ListLoanPaymentsUseCase) Execute(ctx context.Context, req *loanpaymentpb.ListLoanPaymentsRequest) (*loanpaymentpb.ListLoanPaymentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityLoanPayment, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.LoanPayment == nil {
		return nil, errors.New("loan_payment repository is not available")
	}

	resp, err := uc.repositories.LoanPayment.ListLoanPayments(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "loan_payment.errors.list_failed", "[ERR-DEFAULT] Failed to list loan payments")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
