package payroll

import (
	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	payrollservice "github.com/erniealice/espyna-golang/internal/application/services/payroll"
	payrolldashboard "github.com/erniealice/espyna-golang/internal/application/usecases/payroll/dashboard"
	payrollremittanceuc "github.com/erniealice/espyna-golang/internal/application/usecases/payroll/payroll_remittance"
	payrollrunuc "github.com/erniealice/espyna-golang/internal/application/usecases/payroll/payroll_run"

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
type PayrollUseCases struct {
	PayrollRun          *payrollrunuc.UseCases
	PayrollRemittance   *payrollremittanceuc.UseCases
	Orchestrator        *payrollservice.Orchestrator
	CalculatePayrollRun *payrollrunuc.CalculatePayrollRunUseCase
	GeneratePayCycles   *payrollrunuc.GeneratePayCyclesUseCase

	// Dashboard use case (nil when postgres build tag is inactive).
	Dashboard *payrolldashboard.GetPayrollDashboardPageDataUseCase
}

// NewUseCases creates all payroll use cases with proper constructor injection.
// CrossDomainRepositories may be empty (then orchestrator is nil and the
// CalculatePayrollRun/GeneratePayCycles use cases panic on Execute — caller
// should pass a real cross-domain bundle for full functionality).
func NewUseCases(
	repos PayrollRepositories,
	cross CrossDomainRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *PayrollUseCases {
	runServices := payrollrunuc.Services{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
	}
	runRepos := payrollrunuc.Repositories{
		PayrollRun: repos.PayrollRun,
	}

	remittanceServices := payrollremittanceuc.Services{
		AuthorizationService: authSvc,
		TransactionService:   txSvc,
		TranslationService:   i18nSvc,
		IDService:            idService,
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

	// Wire payroll dashboard via type assertions on payroll repos.
	var payrollDash *payrolldashboard.GetPayrollDashboardPageDataUseCase
	if repos.PayrollRun != nil && repos.PayrollRemittance != nil {
		runQ, rOK := repos.PayrollRun.(payrolldashboard.PayrollRunDashboardQueries)
		remQ, mOK := repos.PayrollRemittance.(payrolldashboard.PayrollRemittanceDashboardQueries)
		if rOK && mOK {
			payrollDash = payrolldashboard.NewGetPayrollDashboardPageDataUseCase(runQ, remQ)
		}
	}

	return &PayrollUseCases{
		PayrollRun:          payrollrunuc.NewUseCases(runRepos, runServices),
		PayrollRemittance:   payrollremittanceuc.NewUseCases(remittanceRepos, remittanceServices),
		Orchestrator:        orch,
		CalculatePayrollRun: payrollrunuc.NewCalculatePayrollRunUseCase(orch, authSvc, i18nSvc, txSvc),
		GeneratePayCycles:   payrollrunuc.NewGeneratePayCyclesUseCase(orch, authSvc, i18nSvc),
		Dashboard:           payrollDash,
	}
}
