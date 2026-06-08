// Package dashboard hosts the Dashboard umbrella sub-aggregate for the
// service-driven domain.
//
// Wave B Round 1 LANDED 2026-05-20: P1.C.1 Admin + P1.C.7 Schedule +
// P1.C.10 Integration.
//
// Wave B Round 2a LANDED 2026-05-20: P1.C.2 Location, P1.C.4 Equity,
// P1.C.6 Payroll dashboards.
//
// Wave B Round 2b LANDED 2026-05-21: P1.C.3 Ledger, P1.C.5 Treasury
// (unified Loan + Cash).
//
// Wave C Round 2b LANDED 2026-05-21: P1.C.8 Expenditure, P1.C.9 Job
// (source aggregate operation), P1.C.11 Product, P1.C.12 Fulfillment.
// All 12 P1.C dashboards now live under this umbrella.
package dashboard

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/dashboard"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/admin"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/equity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/ledger"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/location"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/payroll"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/schedule"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
)

// Deps groups every dependency the umbrella factory needs to thread into
// per-candidate `NewUseCases` calls. Round 1 only references Admin +
// Schedule fields. Other candidates land their typed fields here when
// their proto + use case ship.
type Deps struct {
	DB         *sql.DB
	Authorizer ports.Authorizer
	Translator ports.Translator

	AdminPermission        admin.PermissionDashboardRepository
	AdminRole              admin.RoleDashboardRepository
	AdminWorkspaceUser     admin.WorkspaceUserDashboardRepository
	AdminWorkspaceUserRole admin.WorkspaceUserRoleDashboardRepository

	// Location — Wave B P1.C.2 LANDED 2026-05-20. Absorbs the previously flat
	// `entity.LocationDashboard *GetLocationDashboardPageDataUseCase` field
	// at `usecases/entity/usecases.go:73` (removed in the same commit). The
	// location dashboard reads across location + location_area
	// (EntityRepositories), with type assertions performed in the
	// composition root. May be nil under non-postgres builds; the use case
	// tolerates nil repos and returns zero-valued sections.
	Location     location.LocationDashboardRepository
	LocationArea location.LocationAreaDashboardRepository

	// Equity — Wave B P1.C.4 LANDED 2026-05-20. Absorbs the previously flat
	// `ledger.EquityDashboard *GetEquityDashboardPageDataUseCase` field at
	// `usecases/ledger/usecases.go:75` (removed in the same commit). The
	// equity dashboard reads across equity_account + equity_transaction
	// (LedgerRepositories), with type assertions performed in the
	// composition root. May be nil under non-postgres builds; the use case
	// tolerates nil repos and returns zero-valued sections.
	EquityAccount     equity.EquityAccountDashboardRepository
	EquityTransaction equity.EquityTransactionDashboardRepository

	// Payroll — Wave B P1.C.6 LANDED 2026-05-20. Absorbs the previously flat
	// `payroll.Dashboard *GetPayrollDashboardPageDataUseCase` field at
	// `usecases/payroll/usecases.go:62` (removed in the same commit). The
	// payroll dashboard reads across payroll_run + payroll_remittance
	// (PayrollRepositories), with type assertions performed in the
	// composition root. May be nil under non-postgres builds; the use case
	// tolerates nil repos and returns zero-valued sections.
	PayrollRun        payroll.PayrollRunDashboardRepository
	PayrollRemittance payroll.PayrollRemittanceDashboardRepository

	// Ledger — Wave B P1.C.3 LANDED 2026-05-21. Absorbs the previously flat
	// `ledger.Dashboard *ledgerdashboard.GetLedgerDashboardPageDataUseCase`
	// field at `usecases/ledger/usecases.go:74` (removed in the same commit).
	// The ledger dashboard reads across account + journal_entry
	// (LedgerRepositories), with type assertions performed in the composition
	// root. May be nil under non-postgres builds; the use case tolerates nil
	// repos and returns zero-valued sections.
	LedgerAccount      ledger.AccountDashboardRepository
	LedgerJournalEntry ledger.JournalEntryDashboardRepository

	// Treasury — Wave B P1.C.5 LANDED 2026-05-21. Unified Loan + Cash
	// candidate per Q-SDM-DASHBOARD-COUNT (LOCKED 2026-05-20). Absorbs the
	// previously flat `treasury.LoanDashboard` + `treasury.CashDashboard`
	// fields at `usecases/treasury/usecases.go:89, :90` (both removed in the
	// same commit). Loan slice reads across loan + loan_payment; Cash slice
	// reads across collection. All type assertions performed in the
	// composition root.
	TreasuryLoan        treasury.LoanDashboardRepository
	TreasuryLoanPayment treasury.LoanPaymentDashboardRepository
	TreasuryCollection  treasury.CollectionDashboardRepository

	// Expenditure — Wave B P1.C.8 LANDED 2026-05-21. Absorbs the previously
	// flat `expenditure.Dashboard` field at `usecases/expenditure/usecases.go:127`
	// (removed in the same commit). Reads from the expenditure aggregate with
	// a kind discriminator ("purchase" | "expense").
	Expenditure expenditure.ExpenditureDashboardRepository

	// Job — Wave B P1.C.9 LANDED 2026-05-21. Absorbs the previously flat
	// `operation.UseCases.Dashboard` field at `usecases/operation/usecases.go:115`
	// (removed in the same commit). Reads across job + job_activity. The
	// service-layer package is named `job` even though the source aggregate
	// is `operation` (per wave-b-surface-map §P1.C.9).
	Job               job.JobDashboardRepository
	JobActivity       job.JobActivityDashboardRepository
	JobActivityRecent job.JobActivityRecentRepository

	// Product — Wave B P1.C.11 LANDED 2026-05-21. Absorbs the previously
	// flat `product.UseCases.Dashboard *GetServiceDashboardPageDataUseCase`
	// field at `usecases/product/usecases.go:78` (removed in the same commit).
	// Reads from the product aggregate filtered by kind.
	Product product.ProductDashboardRepository

	// Fulfillment — Wave B P1.C.12 LANDED 2026-05-21. Absorbs the previously
	// flat `fulfillment.UseCases.Dashboard` field at
	// `usecases/fulfillment/usecases.go:35` (removed in the same commit).
	// Reads from the fulfillment aggregate with status-event joins.
	Fulfillment fulfillment.FulfillmentDashboardRepository

	ScheduleEntityDashboard *eventdashboard.GetScheduleDashboardPageDataUseCase
}

// DashboardUseCases is the per-candidate aggregator.
//
// Wave B/C status: 12/12 LANDED — Admin (P1.C.1), Location (P1.C.2),
// Ledger (P1.C.3), Equity (P1.C.4), Treasury (P1.C.5 unified Loan+Cash),
// Payroll (P1.C.6), Schedule (P1.C.7), Integration (P1.C.10),
// Expenditure (P1.C.8), Job (P1.C.9, source aggregate `operation`),
// Product (P1.C.11), Fulfillment (P1.C.12).
type DashboardUseCases struct {
	Admin       *admin.UseCases       // P1.C.1 LANDED
	Location    *location.UseCases    // P1.C.2 LANDED 2026-05-20
	Ledger      *ledger.UseCases      // P1.C.3 LANDED 2026-05-21
	Equity      *equity.UseCases      // P1.C.4 LANDED 2026-05-20
	Treasury    *treasury.UseCases    // P1.C.5 LANDED 2026-05-21 (unified Loan+Cash)
	Payroll     *payroll.UseCases     // P1.C.6 LANDED 2026-05-20
	Schedule    *schedule.UseCases    // P1.C.7 LANDED
	Integration *integration.UseCases // P1.C.10 LANDED 2026-06-08 (typed-field conversion)
	Expenditure *expenditure.UseCases // P1.C.8 LANDED 2026-05-21
	Job         *job.UseCases         // P1.C.9 LANDED 2026-05-21 (source aggregate `operation`)
	Product     *product.UseCases     // P1.C.11 LANDED 2026-05-21
	Fulfillment *fulfillment.UseCases // P1.C.12 LANDED 2026-05-21
}

// NewDashboardUseCases wires every landed candidate from grouped
// dependencies. Pending candidates remain nil placeholders.
func NewDashboardUseCases(deps *Deps) *DashboardUseCases {
	if deps == nil {
		deps = &Deps{}
	}
	return &DashboardUseCases{
		Admin: admin.NewUseCases(&admin.Deps{
			Permission:        deps.AdminPermission,
			Role:              deps.AdminRole,
			WorkspaceUser:     deps.AdminWorkspaceUser,
			WorkspaceUserRole: deps.AdminWorkspaceUserRole,
			Translator:        deps.Translator,
		}),
		Location: location.NewUseCases(&location.Deps{
			Location:     deps.Location,
			LocationArea: deps.LocationArea,
			Translator:   deps.Translator,
		}), // Wave B P1.C.2 LANDED
		Ledger: ledger.NewUseCases(&ledger.Deps{
			Account:      deps.LedgerAccount,
			JournalEntry: deps.LedgerJournalEntry,
		}), // Wave B P1.C.3 LANDED 2026-05-21
		Equity: equity.NewUseCases(&equity.Deps{
			EquityAccount:     deps.EquityAccount,
			EquityTransaction: deps.EquityTransaction,
		}), // Wave B P1.C.4 LANDED
		Treasury: treasury.NewUseCases(&treasury.Deps{
			Loan:        deps.TreasuryLoan,
			LoanPayment: deps.TreasuryLoanPayment,
			Collection:  deps.TreasuryCollection,
		}), // Wave B P1.C.5 LANDED 2026-05-21 (unified Loan+Cash)
		Payroll: payroll.NewUseCases(&payroll.Deps{
			PayrollRun:        deps.PayrollRun,
			PayrollRemittance: deps.PayrollRemittance,
			Translator:        deps.Translator,
		}), // Wave B P1.C.6 LANDED
		Schedule: schedule.NewUseCases(&schedule.Deps{
			EntityDashboard: deps.ScheduleEntityDashboard,
			Authorizer:      deps.Authorizer,
			Translator:      deps.Translator,
		}),
		Integration: integration.NewUseCases(), // Wave B P1.C.10 LANDED (typed-field conversion)
		Expenditure: expenditure.NewUseCases(&expenditure.Deps{
			Expenditure: deps.Expenditure,
		}), // Wave C P1.C.8 LANDED 2026-05-21
		Job: job.NewUseCases(&job.Deps{
			Job:               deps.Job,
			JobActivity:       deps.JobActivity,
			JobActivityRecent: deps.JobActivityRecent,
		}), // Wave C P1.C.9 LANDED 2026-05-21 (source aggregate `operation`)
		Product: product.NewUseCases(&product.Deps{
			Product: deps.Product,
		}), // Wave C P1.C.11 LANDED 2026-05-21
		Fulfillment: fulfillment.NewUseCases(&fulfillment.Deps{
			Fulfillment: deps.Fulfillment,
		}), // Wave C P1.C.12 LANDED 2026-05-21
	}
}
