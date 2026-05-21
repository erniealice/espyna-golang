package loan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

const entityLoan = "loan"

// CreateLoanRepositories groups all repository dependencies.
type CreateLoanRepositories struct {
	Loan loanpb.LoanDomainServiceServer
}

// CreateLoanServices groups all business service dependencies.
type CreateLoanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateLoanUseCase handles the business logic for creating loans.
type CreateLoanUseCase struct {
	repositories CreateLoanRepositories
	services     CreateLoanServices
}

// NewCreateLoanUseCase creates the use case with grouped dependencies.
func NewCreateLoanUseCase(
	repositories CreateLoanRepositories,
	services CreateLoanServices,
) *CreateLoanUseCase {
	return &CreateLoanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create loan operation.
func (uc *CreateLoanUseCase) Execute(ctx context.Context, req *loanpb.CreateLoanRequest) (*loanpb.CreateLoanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityLoan, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateLoanUseCase) executeWithTransaction(ctx context.Context, req *loanpb.CreateLoanRequest) (*loanpb.CreateLoanResponse, error) {
	var result *loanpb.CreateLoanResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "loan.errors.creation_failed", "Loan creation failed [DEFAULT]")
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

func (uc *CreateLoanUseCase) executeCore(ctx context.Context, req *loanpb.CreateLoanRequest) (*loanpb.CreateLoanResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.Loan == nil {
		return nil, errors.New("loan repository is not available")
	}
	return uc.repositories.Loan.CreateLoan(ctx, req)
}

func (uc *CreateLoanUseCase) validateInput(ctx context.Context, req *loanpb.CreateLoanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.data_required", "[ERR-DEFAULT] Loan data is required"))
	}

	req.Data.LenderName = strings.TrimSpace(req.Data.LenderName)

	if req.Data.LenderName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.lender_name_required", "[ERR-DEFAULT] Lender name is required"))
	}
	if req.Data.PrincipalAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.principal_positive", "[ERR-DEFAULT] Principal amount must be greater than zero"))
	}
	if req.Data.TermMonths <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.term_positive", "[ERR-DEFAULT] Term months must be greater than zero"))
	}
	if req.Data.InterestRate < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.interest_rate_non_negative", "[ERR-DEFAULT] Interest rate must not be negative"))
	}
	return nil
}

func (uc *CreateLoanUseCase) enrichData(loan *loanpb.Loan) error {
	now := time.Now()

	if loan.Id == "" {
		loan.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set remaining balance to principal on creation
	if loan.RemainingBalance == 0 {
		loan.RemainingBalance = loan.PrincipalAmount
	}

	// Default status to ACTIVE if not set
	if loan.Status == loanpb.LoanStatus_LOAN_STATUS_UNSPECIFIED {
		loan.Status = loanpb.LoanStatus_LOAN_STATUS_ACTIVE
	}

	loan.DateCreated = &[]int64{now.UnixMilli()}[0]
	loan.DateModified = &[]int64{now.UnixMilli()}[0]
	loan.Active = true

	return nil
}

func (uc *CreateLoanUseCase) validateBusinessRules(ctx context.Context, loan *loanpb.Loan) error {
	if len(loan.LenderName) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.lender_name_too_long", "[ERR-DEFAULT] Lender name must not exceed 200 characters"))
	}
	if loan.TermMonths > 600 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "loan.validation.term_too_long", "[ERR-DEFAULT] Term must not exceed 600 months (50 years)"))
	}
	return nil
}
