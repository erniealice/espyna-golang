package payrollrun

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

const entityPayrollRun = "payroll_run"

// CreatePayrollRunRepositories groups all repository dependencies.
type CreatePayrollRunRepositories struct {
	PayrollRun payrollrunpb.PayrollRunDomainServiceServer
}

// newCreatePayrollRunRepositories casts the generic Repositories to this use case's repos.
func newCreatePayrollRunRepositories(r Repositories) CreatePayrollRunRepositories {
	return CreatePayrollRunRepositories{PayrollRun: r.PayrollRun}
}

// CreatePayrollRunServices groups all business service dependencies.
type CreatePayrollRunServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreatePayrollRunUseCase handles the business logic for creating a payroll run.
type CreatePayrollRunUseCase struct {
	repositories CreatePayrollRunRepositories
	services     CreatePayrollRunServices
}

// NewCreatePayrollRunUseCase creates the use case with grouped dependencies.
func NewCreatePayrollRunUseCase(
	repositories CreatePayrollRunRepositories,
	services CreatePayrollRunServices,
) *CreatePayrollRunUseCase {
	return &CreatePayrollRunUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payroll run operation.
func (uc *CreatePayrollRunUseCase) Execute(ctx context.Context, req *payrollrunpb.CreatePayrollRunRequest) (*payrollrunpb.CreatePayrollRunResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPayrollRun, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreatePayrollRunUseCase) executeWithTransaction(ctx context.Context, req *payrollrunpb.CreatePayrollRunRequest) (*payrollrunpb.CreatePayrollRunResponse, error) {
	var result *payrollrunpb.CreatePayrollRunResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "payroll_run.errors.creation_failed", "Payroll run creation failed [DEFAULT]")
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

func (uc *CreatePayrollRunUseCase) executeCore(ctx context.Context, req *payrollrunpb.CreatePayrollRunRequest) (*payrollrunpb.CreatePayrollRunResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichPayrollRunData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.PayrollRun == nil {
		return nil, errors.New("payroll run repository is not available")
	}
	return uc.repositories.PayrollRun.CreatePayrollRun(ctx, req)
}

func (uc *CreatePayrollRunUseCase) validateInput(ctx context.Context, req *payrollrunpb.CreatePayrollRunRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.data_required", "[ERR-DEFAULT] Payroll run data is required"))
	}
	if req.Data.PayPeriodStart == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.pay_period_start_required", "[ERR-DEFAULT] Pay period start date is required"))
	}
	if req.Data.PayPeriodEnd == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.pay_period_end_required", "[ERR-DEFAULT] Pay period end date is required"))
	}
	return nil
}

func (uc *CreatePayrollRunUseCase) enrichPayrollRunData(run *payrollrunpb.PayrollRun) error {
	now := time.Now()

	if run.Id == "" {
		run.Id = uc.services.IDGenerator.GenerateID()
	}

	// Default status: Draft
	if run.Status == payrollrunpb.PayrollRunStatus_PAYROLL_RUN_STATUS_UNSPECIFIED {
		run.Status = payrollrunpb.PayrollRunStatus_PAYROLL_RUN_STATUS_DRAFT
	}

	run.DateCreated = &[]int64{now.UnixMilli()}[0]
	run.DateModified = &[]int64{now.UnixMilli()}[0]

	return nil
}

func (uc *CreatePayrollRunUseCase) validateBusinessRules(ctx context.Context, run *payrollrunpb.PayrollRun) error {
	// Pay period end must be after pay period start
	if run.PayPeriodEnd != "" && run.PayPeriodStart != "" && run.PayPeriodEnd <= run.PayPeriodStart {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.pay_period_end_before_start", "[ERR-DEFAULT] Pay period end must be after pay period start"))
	}
	return nil
}
