package payrollrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// ListPayrollRunsRepositories groups all repository dependencies.
type ListPayrollRunsRepositories struct {
	PayrollRun payrollrunpb.PayrollRunDomainServiceServer
}

// newListPayrollRunsRepositories casts the generic Repositories to this use case's repos.
func newListPayrollRunsRepositories(r Repositories) ListPayrollRunsRepositories {
	return ListPayrollRunsRepositories{PayrollRun: r.PayrollRun}
}

// ListPayrollRunsServices groups all business service dependencies.
type ListPayrollRunsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPayrollRunsUseCase handles the business logic for listing payroll runs.
type ListPayrollRunsUseCase struct {
	repositories ListPayrollRunsRepositories
	services     ListPayrollRunsServices
}

// NewListPayrollRunsUseCase creates the use case with grouped dependencies.
func NewListPayrollRunsUseCase(
	repositories ListPayrollRunsRepositories,
	services ListPayrollRunsServices,
) *ListPayrollRunsUseCase {
	return &ListPayrollRunsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list payroll runs operation.
func (uc *ListPayrollRunsUseCase) Execute(ctx context.Context, req *payrollrunpb.ListPayrollRunsRequest) (*payrollrunpb.ListPayrollRunsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPayrollRun, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_run.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PayrollRun == nil {
		return nil, errors.New("payroll run repository is not available")
	}

	resp, err := uc.repositories.PayrollRun.ListPayrollRuns(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_run.errors.list_failed", "[ERR-DEFAULT] Failed to list payroll runs")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

func (uc *ListPayrollRunsUseCase) validateInput(ctx context.Context, req *payrollrunpb.ListPayrollRunsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payroll_run.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}
