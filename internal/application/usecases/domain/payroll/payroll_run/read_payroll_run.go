package payrollrun

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// ReadPayrollRunRepositories groups all repository dependencies.
type ReadPayrollRunRepositories struct {
	PayrollRun payrollrunpb.PayrollRunDomainServiceServer
}

// newReadPayrollRunRepositories casts the generic Repositories to this use case's repos.
func newReadPayrollRunRepositories(r Repositories) ReadPayrollRunRepositories {
	return ReadPayrollRunRepositories{PayrollRun: r.PayrollRun}
}

// ReadPayrollRunServices groups all business service dependencies.
type ReadPayrollRunServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadPayrollRunUseCase handles the business logic for reading a payroll run.
type ReadPayrollRunUseCase struct {
	repositories ReadPayrollRunRepositories
	services     ReadPayrollRunServices
}

// NewReadPayrollRunUseCase creates the use case with grouped dependencies.
func NewReadPayrollRunUseCase(
	repositories ReadPayrollRunRepositories,
	services ReadPayrollRunServices,
) *ReadPayrollRunUseCase {
	return &ReadPayrollRunUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read payroll run operation.
func (uc *ReadPayrollRunUseCase) Execute(ctx context.Context, req *payrollrunpb.ReadPayrollRunRequest) (*payrollrunpb.ReadPayrollRunResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPayrollRun, ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.repositories.PayrollRun == nil {
		return nil, errors.New("payroll run repository is not available")
	}

	resp, err := uc.repositories.PayrollRun.ReadPayrollRun(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadPayrollRunUseCase) validateInput(ctx context.Context, req *payrollrunpb.ReadPayrollRunRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payroll_run.validation.id_required", "[ERR-DEFAULT] Payroll run ID is required"))
	}
	return nil
}
