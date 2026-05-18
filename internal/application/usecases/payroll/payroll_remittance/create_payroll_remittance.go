package payrollremittance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
)

const entityPayrollRemittance = "payroll_remittance"

// CreatePayrollRemittanceRepositories groups all repository dependencies.
type CreatePayrollRemittanceRepositories struct {
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
}

// newCreatePayrollRemittanceRepositories casts the generic Repositories to this use case's repos.
func newCreatePayrollRemittanceRepositories(r Repositories) CreatePayrollRemittanceRepositories {
	return CreatePayrollRemittanceRepositories{PayrollRemittance: r.PayrollRemittance}
}

// CreatePayrollRemittanceServices groups all business service dependencies.
type CreatePayrollRemittanceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePayrollRemittanceUseCase handles the business logic for creating a payroll remittance.
type CreatePayrollRemittanceUseCase struct {
	repositories CreatePayrollRemittanceRepositories
	services     CreatePayrollRemittanceServices
}

// NewCreatePayrollRemittanceUseCase creates the use case with grouped dependencies.
func NewCreatePayrollRemittanceUseCase(
	repositories CreatePayrollRemittanceRepositories,
	services CreatePayrollRemittanceServices,
) *CreatePayrollRemittanceUseCase {
	return &CreatePayrollRemittanceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payroll remittance operation.
func (uc *CreatePayrollRemittanceUseCase) Execute(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) (*payrollremittancepb.CreatePayrollRemittanceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPayrollRemittance, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreatePayrollRemittanceUseCase) executeWithTransaction(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) (*payrollremittancepb.CreatePayrollRemittanceResponse, error) {
	var result *payrollremittancepb.CreatePayrollRemittanceResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payroll_remittance.errors.creation_failed", "Payroll remittance creation failed [DEFAULT]")
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

func (uc *CreatePayrollRemittanceUseCase) executeCore(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) (*payrollremittancepb.CreatePayrollRemittanceResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichRemittanceData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.PayrollRemittance == nil {
		return nil, errors.New("payroll remittance repository is not available")
	}
	return uc.repositories.PayrollRemittance.CreatePayrollRemittance(ctx, req)
}

func (uc *CreatePayrollRemittanceUseCase) validateInput(ctx context.Context, req *payrollremittancepb.CreatePayrollRemittanceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.data_required", "[ERR-DEFAULT] Payroll remittance data is required"))
	}
	if req.Data.PayrollRunId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.payroll_run_id_required", "[ERR-DEFAULT] Payroll run ID is required"))
	}
	if req.Data.RemittanceType == payrollremittancepb.RemittanceType_REMITTANCE_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.remittance_type_required", "[ERR-DEFAULT] Remittance type is required"))
	}
	if req.Data.Amount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.amount_required", "[ERR-DEFAULT] Remittance amount must be greater than zero"))
	}
	return nil
}

func (uc *CreatePayrollRemittanceUseCase) enrichRemittanceData(rem *payrollremittancepb.PayrollRemittance) error {
	now := time.Now()

	if rem.Id == "" {
		rem.Id = uc.services.IDService.GenerateID()
	}

	// Default status: Pending
	if rem.Status == payrollremittancepb.RemittanceStatus_REMITTANCE_STATUS_UNSPECIFIED {
		rem.Status = payrollremittancepb.RemittanceStatus_REMITTANCE_STATUS_PENDING
	}

	rem.DateCreated = &[]int64{now.UnixMilli()}[0]

	return nil
}

func (uc *CreatePayrollRemittanceUseCase) validateBusinessRules(ctx context.Context, rem *payrollremittancepb.PayrollRemittance) error {
	// Remittance type must be one of: SSS, PhilHealth, Pag-IBIG, BIR withholding
	switch rem.RemittanceType {
	case payrollremittancepb.RemittanceType_REMITTANCE_TYPE_SSS,
		payrollremittancepb.RemittanceType_REMITTANCE_TYPE_PHILHEALTH,
		payrollremittancepb.RemittanceType_REMITTANCE_TYPE_PAGIBIG,
		payrollremittancepb.RemittanceType_REMITTANCE_TYPE_BIR_WITHHOLDING:
		// valid
	default:
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_remittance.validation.invalid_remittance_type", "[ERR-DEFAULT] Invalid remittance type — must be SSS, PhilHealth, Pag-IBIG, or BIR Withholding"))
	}
	return nil
}
