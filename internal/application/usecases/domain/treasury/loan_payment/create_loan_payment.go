package loanpayment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

const entityLoanPayment = "loan_payment"

// CreateLoanPaymentRepositories groups all repository dependencies.
type CreateLoanPaymentRepositories struct {
	LoanPayment loanpaymentpb.LoanPaymentDomainServiceServer
}

// CreateLoanPaymentServices groups all business service dependencies.
type CreateLoanPaymentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateLoanPaymentUseCase handles the business logic for recording loan payments.
//
// A loan payment records an actual cash disbursement/receipt against a loan.
// The handler should also update the parent loan's remaining_balance after creating the payment.
// Payment posting auto-generates two journal entries:
//  1. LOAN_PAYMENT: DR Loans Payable + DR Interest Expense / CR Cash
//  2. LOAN_FEE_AMORTIZATION: DR Debt Issuance Cost / CR Prepaid Financing Fees (if fee > 0)
type CreateLoanPaymentUseCase struct {
	repositories CreateLoanPaymentRepositories
	services     CreateLoanPaymentServices
}

// NewCreateLoanPaymentUseCase creates the use case with grouped dependencies.
func NewCreateLoanPaymentUseCase(
	repositories CreateLoanPaymentRepositories,
	services CreateLoanPaymentServices,
) *CreateLoanPaymentUseCase {
	return &CreateLoanPaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create loan payment operation.
func (uc *CreateLoanPaymentUseCase) Execute(ctx context.Context, req *loanpaymentpb.CreateLoanPaymentRequest) (*loanpaymentpb.CreateLoanPaymentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityLoanPayment, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateLoanPaymentUseCase) executeWithTransaction(ctx context.Context, req *loanpaymentpb.CreateLoanPaymentRequest) (*loanpaymentpb.CreateLoanPaymentResponse, error) {
	var result *loanpaymentpb.CreateLoanPaymentResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "loan_payment.errors.creation_failed", "Loan payment creation failed [DEFAULT]")
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

func (uc *CreateLoanPaymentUseCase) executeCore(ctx context.Context, req *loanpaymentpb.CreateLoanPaymentRequest) (*loanpaymentpb.CreateLoanPaymentResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.LoanPayment == nil {
		return nil, errors.New("loan_payment repository is not available")
	}
	return uc.repositories.LoanPayment.CreateLoanPayment(ctx, req)
}

func (uc *CreateLoanPaymentUseCase) validateInput(ctx context.Context, req *loanpaymentpb.CreateLoanPaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.data_required", "[ERR-DEFAULT] Loan payment data is required"))
	}
	if req.Data.LoanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.loan_id_required", "[ERR-DEFAULT] Loan ID is required"))
	}
	if req.Data.TotalAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.amount_positive", "[ERR-DEFAULT] Total amount must be greater than zero"))
	}
	if req.Data.PrincipalAmount < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.principal_non_negative", "[ERR-DEFAULT] Principal amount must not be negative"))
	}
	if req.Data.InterestAmount < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.interest_non_negative", "[ERR-DEFAULT] Interest amount must not be negative"))
	}
	return nil
}

func (uc *CreateLoanPaymentUseCase) enrichData(payment *loanpaymentpb.LoanPayment) error {
	now := time.Now()

	if payment.Id == "" {
		payment.Id = uc.services.IDGenerator.GenerateID()
	}

	if payment.PaymentDate == "" {
		payment.PaymentDate = now.Format("2006-01-02")
	}

	// Auto-compute total if not provided
	if payment.TotalAmount == 0 {
		payment.TotalAmount = payment.PrincipalAmount + payment.InterestAmount + payment.FeeAmount
	}

	payment.DateCreated = &[]int64{now.UnixMilli()}[0]
	payment.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

func (uc *CreateLoanPaymentUseCase) validateBusinessRules(ctx context.Context, payment *loanpaymentpb.LoanPayment) error {
	// Verify total = principal + interest + fee (allow small rounding tolerance)
	computed := payment.PrincipalAmount + payment.InterestAmount + payment.FeeAmount
	diff := payment.TotalAmount - computed
	if diff != 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan_payment.validation.total_mismatch", "[ERR-DEFAULT] Total amount must equal principal + interest + fees"))
	}
	return nil
}
