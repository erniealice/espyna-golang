package payroll

import (
	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	payrollservice "github.com/erniealice/espyna-golang/internal/application/usecases/service/payroll"
	payrollremittanceuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/payroll/payroll_remittance"
	payrollrunuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/payroll/payroll_run"

	// Protobuf domain services for payroll repositories
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
	leavebalancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/leave_balance"
	paycyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/pay_cycle"
	payrollremittancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_remittance"
	payrollrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/payroll_run"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"
)

// PayrollRepositories contains all payroll domain repositories.
// Cross-domain repos (supplier, expenditure, etc.) are passed alongside via
// CrossDomainRepositories so the orchestrator can read employees / contracts
// and write Expenditure rows.
type PayrollRepositories struct {
	PayrollRun        payrollrunpb.PayrollRunDomainServiceServer
	PayrollRemittance payrollremittancepb.PayrollRemittanceDomainServiceServer
	PayCycle          paycyclepb.PayCycleDomainServiceServer
	RateTable         ratetablepb.RateTableDomainServiceServer
	RateBand          ratebandpb.RateBandDomainServiceServer
	LeaveBalance      leavebalancepb.LeaveBalanceDomainServiceServer
}

// CrossDomainRepositories holds repos owned by other domains that the payroll
// orchestrator needs to read and write.
type CrossDomainRepositories struct {
	Workspace            workspacepb.WorkspaceDomainServiceServer
	Supplier             supplierpb.SupplierDomainServiceServer
	SupplierContract     suppliercontractpb.SupplierContractDomainServiceServer
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
	Expenditure          expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem  expenditurelinepb.ExpenditureLineItemDomainServiceServer
}

// PayrollUseCases contains all payroll-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 — the two flat advance fields
// (CalculatePayrollRun + GeneratePayCycles) have been folded into the
// PayrollRun sub-aggregate as .PayrollRun.Calculate / .PayrollRun.GeneratePayCycles.
// F6 closure.
//
// 20260520-service-domain-migration Wave B P1.C.6 — the `Dashboard` flat
// field has been folded into the service-driven Dashboard umbrella at
// `Service.Dashboard.Payroll.GetPayrollDashboard`. The proto contract moved
// from `proto/v1/domain/payroll/dashboard/` to
// `proto/v1/service/dashboard/payroll/`; the repository composition lives
// at `usecases/service/dashboard/payroll/`. See hexagonal-rules.md §8 Wave B
// P1.C.6 worked example.
//
// 20260530-audit-remediation-program Wave 1.4 (structure audit 20260527144541
// F5 / Q9→A) — the `Orchestrator *payrollservice.Orchestrator` flat field has
// been REMOVED. It exposed the runtime payroll lifecycle service (a
// services/payroll object, not a payroll entity sub-aggregate) directly on the
// Layer-7 aggregator — the F5 "flat aggregator field when sub-aggregates exist"
// drift. The orchestrator's two consumable flows are already nested under the
// PayrollRun sub-aggregate as `.PayrollRun.Calculate` /
// `.PayrollRun.GeneratePayCycles` (Phase 3 F6 closure), and the flat field had
// ZERO readers anywhere in the monorepo (no direct access, no reflection
// navigator, no tests, not referenced by the routing config or composition
// root). Removal is therefore behavior-preserving: the orchestrator is still
// constructed below and threaded into the two wrappers exactly as before — it
// is simply no longer re-exposed as dead aggregator state.
type PayrollUseCases struct {
	PayrollRun        *payrollrunuc.UseCases
	PayrollRemittance *payrollremittanceuc.UseCases
}

// NewUseCases creates all payroll use cases with proper constructor injection.
// CrossDomainRepositories may be empty (then orchestrator is nil and the
// CalculatePayrollRun/GeneratePayCycles use cases panic on Execute — caller
// should pass a real cross-domain bundle for full functionality).
func NewUseCases(
	repos PayrollRepositories,
	cross CrossDomainRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) *PayrollUseCases {
	runServices := payrollrunuc.Services{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idService,
		ActionGatekeeper: actionGate,
	}
	runRepos := payrollrunuc.Repositories{
		PayrollRun: repos.PayrollRun,
	}

	remittanceServices := payrollremittanceuc.Services{
		Authorizer:       authSvc,
		Transactor:       txSvc,
		Translator:       i18nSvc,
		IDGenerator:      idService,
		ActionGatekeeper: actionGate,
	}
	remittanceRepos := payrollremittanceuc.Repositories{
		PayrollRemittance: repos.PayrollRemittance,
	}

	var orch *payrollservice.Orchestrator
	if cross.Workspace != nil &&
		cross.Supplier != nil &&
		cross.SupplierContract != nil &&
		cross.SupplierContractLine != nil &&
		cross.Expenditure != nil &&
		cross.ExpenditureLineItem != nil &&
		repos.PayrollRun != nil &&
		repos.PayCycle != nil &&
		repos.RateTable != nil &&
		repos.RateBand != nil {
		orch = payrollservice.NewOrchestrator(payrollservice.OrchestratorRepositories{
			Workspace:            cross.Workspace,
			Supplier:             cross.Supplier,
			SupplierContract:     cross.SupplierContract,
			SupplierContractLine: cross.SupplierContractLine,
			PayrollRun:           repos.PayrollRun,
			PayCycle:             repos.PayCycle,
			RateTable:            repos.RateTable,
			RateBand:             repos.RateBand,
			LeaveBalance:         repos.LeaveBalance,
			Expenditure:          cross.Expenditure,
			ExpenditureLineItem:  cross.ExpenditureLineItem,
		}, idService)
	}

	// Build sub-aggregates. The orchestrator-backed wrappers nest under
	// PayrollRun (Phase 3 F6 closure).
	//
	// Dashboard wiring relocated to the service-driven umbrella per Wave B
	// P1.C.6 — see `internal/composition/core/initializers/service.go`
	// where the payroll dashboard repos are threaded into
	// `Service.Dashboard.Payroll`.
	payrollRunUC := payrollrunuc.NewUseCases(runRepos, runServices)
	if payrollRunUC != nil {
		payrollRunUC.Calculate = payrollrunuc.NewCalculatePayrollRunUseCase(orch, authSvc, i18nSvc, txSvc)
		payrollRunUC.GeneratePayCycles = payrollrunuc.NewGeneratePayCyclesUseCase(orch, authSvc, i18nSvc)
	}

	return &PayrollUseCases{
		PayrollRun:        payrollRunUC,
		PayrollRemittance: payrollremittanceuc.NewUseCases(remittanceRepos, remittanceServices),
	}
}
