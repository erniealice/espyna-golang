package loanpayment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// ListLoanPaymentsRepositories groups all repository dependencies.
type ListLoanPaymentsRepositories struct {
	LoanPayment loanpaymentpb.LoanPaymentDomainServiceServer
}

// ListLoanPaymentsServices groups all business service dependencies.
type ListLoanPaymentsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityLoanPayment,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if uc.repositories.LoanPayment == nil {
		return nil, errors.New("loan_payment repository is not available")
	}

	resp, err := uc.repositories.LoanPayment.ListLoanPayments(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.errors.list_failed", "[ERR-DEFAULT] Failed to list loan payments")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
