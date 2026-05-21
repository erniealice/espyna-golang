package payrollrun

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/services/payroll"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
)

// CalculatePayrollRunUseCase wraps the payroll.Orchestrator behind the standard
// proto-message use-case interface so it can be mounted as a route via the same
// generic handler machinery as other use cases.
type CalculatePayrollRunUseCase struct {
	orchestrator         *payroll.Orchestrator
	authorizationService ports.Authorizer
	translationService   ports.Translator
	transactionService   ports.Transactor
}

// NewCalculatePayrollRunUseCase wires the use case.
func NewCalculatePayrollRunUseCase(
	orchestrator *payroll.Orchestrator,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	txSvc ports.Transactor,
) *CalculatePayrollRunUseCase {
	return &CalculatePayrollRunUseCase{
		orchestrator:         orchestrator,
		authorizationService: authSvc,
		translationService:   i18nSvc,
		transactionService:   txSvc,
	}
}

// Execute performs the calculation. The PayrollRun must already exist and have
// PayCycles generated for it (use GeneratePayCycles first).
func (uc *CalculatePayrollRunUseCase) Execute(ctx context.Context, req *payrollrunpb.CalculatePayrollRunRequest) (*payrollrunpb.CalculatePayrollRunResponse, error) {
	if err := authcheck.Check(ctx, uc.authorizationService, uc.translationService,
		entityPayrollRun, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.PayrollRunId == "" {
		return errResponse("payroll_run_id is required"), nil
	}
	if uc.orchestrator == nil {
		return errResponse("payroll orchestrator not configured"), nil
	}

	result, err := uc.orchestrator.CalculatePayrollRun(ctx, req.PayrollRunId)
	if err != nil {
		return errResponse(fmt.Sprintf("calculate payroll run: %v", err)), nil
	}

	return &payrollrunpb.CalculatePayrollRunResponse{
		PayrollRunId:    result.PayrollRunID,
		CyclesProcessed: int32(result.CyclesProcessed),
		EmployeesPaid:   int32(result.EmployeesPaid),
		TotalGross:      result.TotalGross,
		TotalDeductions: result.TotalDeductions,
		TotalNet:        result.TotalNet,
		Success:         true,
	}, nil
}

func errResponse(msg string) *payrollrunpb.CalculatePayrollRunResponse {
	return &payrollrunpb.CalculatePayrollRunResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:    "calculate_failed",
			Message: msg,
		},
	}
}

// GeneratePayCyclesUseCase wraps the orchestrator's cycle generation.
type GeneratePayCyclesUseCase struct {
	orchestrator         *payroll.Orchestrator
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGeneratePayCyclesUseCase wires the use case.
func NewGeneratePayCyclesUseCase(
	orchestrator *payroll.Orchestrator,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GeneratePayCyclesUseCase {
	return &GeneratePayCyclesUseCase{
		orchestrator:         orchestrator,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute generates pay cycles for the run.
func (uc *GeneratePayCyclesUseCase) Execute(ctx context.Context, req *payrollrunpb.GeneratePayCyclesRequest) (*payrollrunpb.GeneratePayCyclesResponse, error) {
	if err := authcheck.Check(ctx, uc.authorizationService, uc.translationService,
		entityPayrollRun, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.PayrollRunId == "" {
		return generateErrResponse("payroll_run_id is required"), nil
	}
	if uc.orchestrator == nil {
		return generateErrResponse("payroll orchestrator not configured"), nil
	}
	if err := uc.orchestrator.GeneratePayCycles(ctx, req.PayrollRunId); err != nil {
		return generateErrResponse(fmt.Sprintf("generate pay cycles: %v", err)), nil
	}
	return &payrollrunpb.GeneratePayCyclesResponse{
		PayrollRunId: req.PayrollRunId,
		Success:      true,
	}, nil
}

func generateErrResponse(msg string) *payrollrunpb.GeneratePayCyclesResponse {
	return &payrollrunpb.GeneratePayCyclesResponse{
		Success: false,
		Error: &commonpb.Error{
			Code:    "generate_cycles_failed",
			Message: msg,
		},
	}
}
