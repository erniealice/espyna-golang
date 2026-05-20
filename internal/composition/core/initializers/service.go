package initializers

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	entityauth "github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/event/dashboard"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
	auditusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
	serviceauthusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	dashboardusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard"
	admindash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/admin"
	equitydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/equity"
	expendituredash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/expenditure"
	fulfillmentdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/fulfillment"
	jobdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/job"
	ledgerdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/ledger"
	locationdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/location"
	payrolldash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/payroll"
	productdash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/product"
	treasurydash "github.com/erniealice/espyna-golang/internal/application/usecases/service/dashboard/treasury"
	reportingusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting"
	securityusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/security"
	audithelpers "github.com/erniealice/espyna-golang/internal/composition/audit"
	authhelpers "github.com/erniealice/espyna-golang/internal/composition/auth"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
	securityhelpers "github.com/erniealice/espyna-golang/internal/composition/security"
)

// InitializeService wires every service-driven use case sub-aggregate.
//
// Wave B Round 1 LANDED: Audit, Security, Auth, Admin (P1.C.1),
// Schedule (P1.C.7), Integration (registry path P1.C.10), AR Aging
// (P1.E.1). Round 2a LANDED 2026-05-20: Location P1.C.2, Equity P1.C.4,
// Payroll P1.C.6. Round 2b LANDED 2026-05-21: Ledger P1.C.3, Treasury
// P1.C.5 (unified Loan+Cash).
//
// Wave C Round 2b LANDED 2026-05-21: Expenditure P1.C.8, Job P1.C.9
// (source aggregate `operation`), Product P1.C.11, Fulfillment P1.C.12.
//
// db may be nil when no SQL provider is in play; in that case the use
// cases degrade gracefully (return empty responses).
func InitializeService(
	db *sql.DB,
	authSvc ports.AuthorizationService,
	i18nSvc ports.TranslationService,
	entityAuth *entityauth.UseCases,
	entityRepos *domain.EntityRepositories,
	ledgerRepos *domain.LedgerRepositories,
	payrollRepos *domain.PayrollRepositories,
	treasuryRepos *domain.TreasuryRepositories,
	expenditureRepos *domain.ExpenditureRepositories,
	operationRepos *domain.OperationRepositories,
	productRepos *domain.ProductRepositories,
	fulfillmentRepos *domain.FulfillmentRepositories,
	scheduleEntityDash *eventdashboard.GetScheduleDashboardPageDataUseCase,
	ledgerReportingSvc any,
) (*service.ServiceUseCases, error) {
	auditSvc := audithelpers.NewAuditServiceFromDB(db)
	permQuery := securityhelpers.NewPermissionQueryFromDB(db)

	auditUC := auditusecases.NewUseCases(
		auditusecases.Repositories{AuditService: auditSvc},
		auditusecases.Services{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
		},
	)

	securityUC := securityusecases.NewUseCases(
		securityusecases.Repositories{PermissionQuery: permQuery},
		securityusecases.Services{TranslationService: i18nSvc},
	)

	deps := &service.Deps{
		DB:                   db,
		AuthorizationService: authSvc,
		TranslationService:   i18nSvc,
	}

	var serviceAuthUC *serviceauthusecases.UseCases = authhelpers.BuildServiceAuthUseCases(entityAuth, deps)

	dashboardDeps := &dashboardusecases.Deps{
		DB:                      db,
		AuthorizationService:    authSvc,
		TranslationService:      i18nSvc,
		ScheduleEntityDashboard: scheduleEntityDash,
	}

	// Wave B P1.C.1 Admin — type-assert entity repos into admin's dashboard
	// repository interfaces. Each may be nil (non-postgres builds tolerate).
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/entity/admin_dashboard_assertions.go.
	if entityRepos != nil {
		if entityRepos.Permission != nil {
			if q, ok := entityRepos.Permission.(admindash.PermissionDashboardRepository); ok {
				dashboardDeps.AdminPermission = q
			}
		}
		if entityRepos.Role != nil {
			if q, ok := entityRepos.Role.(admindash.RoleDashboardRepository); ok {
				dashboardDeps.AdminRole = q
			}
		}
		if entityRepos.WorkspaceUser != nil {
			if q, ok := entityRepos.WorkspaceUser.(admindash.WorkspaceUserDashboardRepository); ok {
				dashboardDeps.AdminWorkspaceUser = q
			}
		}
		if entityRepos.WorkspaceUserRole != nil {
			if q, ok := entityRepos.WorkspaceUserRole.(admindash.WorkspaceUserRoleDashboardRepository); ok {
				dashboardDeps.AdminWorkspaceUserRole = q
			}
		}

		// Wave B P1.C.2 Location — type-assert entity repos into location's
		// dashboard repository interfaces. Each may be nil (non-postgres
		// builds tolerate). Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see
		// contrib/postgres/internal/adapter/entity/location_dashboard_assertions.go.
		if entityRepos.Location != nil {
			if q, ok := entityRepos.Location.(locationdash.LocationDashboardRepository); ok {
				dashboardDeps.Location = q
			}
		}
		if entityRepos.LocationArea != nil {
			if q, ok := entityRepos.LocationArea.(locationdash.LocationAreaDashboardRepository); ok {
				dashboardDeps.LocationArea = q
			}
		}
	}

	// Wave B P1.C.4 Equity — type-assert ledger repos into equity's dashboard
	// repository interfaces. Each may be nil (non-postgres builds tolerate).
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/ledger/equity_dashboard_assertions.go.
	if ledgerRepos != nil {
		if ledgerRepos.EquityAccount != nil {
			if q, ok := ledgerRepos.EquityAccount.(equitydash.EquityAccountDashboardRepository); ok {
				dashboardDeps.EquityAccount = q
			}
		}
		if ledgerRepos.EquityTransaction != nil {
			if q, ok := ledgerRepos.EquityTransaction.(equitydash.EquityTransactionDashboardRepository); ok {
				dashboardDeps.EquityTransaction = q
			}
		}

		// Wave B P1.C.3 Ledger — type-assert ledger repos into ledger's
		// dashboard repository interfaces. Each may be nil (non-postgres
		// builds tolerate). Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see
		// contrib/postgres/internal/adapter/ledger/ledger_dashboard_assertions.go.
		if ledgerRepos.Account != nil {
			if q, ok := ledgerRepos.Account.(ledgerdash.AccountDashboardRepository); ok {
				dashboardDeps.LedgerAccount = q
			}
		}
		if ledgerRepos.JournalEntry != nil {
			if q, ok := ledgerRepos.JournalEntry.(ledgerdash.JournalEntryDashboardRepository); ok {
				dashboardDeps.LedgerJournalEntry = q
			}
		}
	}

	// Wave B P1.C.5 Treasury (unified Loan+Cash) — type-assert treasury repos
	// into treasury's dashboard repository interfaces. Each may be nil (non-
	// postgres builds tolerate). Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED —
	// see contrib/postgres/internal/adapter/treasury/treasury_dashboard_assertions.go.
	// Q-SDM-DASHBOARD-COUNT LOCKED 2026-05-20: treasury.LoanDashboard +
	// treasury.CashDashboard fold into ONE unified candidate.
	if treasuryRepos != nil {
		if treasuryRepos.Loan != nil {
			if q, ok := treasuryRepos.Loan.(treasurydash.LoanDashboardRepository); ok {
				dashboardDeps.TreasuryLoan = q
			}
		}
		if treasuryRepos.LoanPayment != nil {
			if q, ok := treasuryRepos.LoanPayment.(treasurydash.LoanPaymentDashboardRepository); ok {
				dashboardDeps.TreasuryLoanPayment = q
			}
		}
		if treasuryRepos.Collection != nil {
			if q, ok := treasuryRepos.Collection.(treasurydash.CollectionDashboardRepository); ok {
				dashboardDeps.TreasuryCollection = q
			}
		}
	}

	// Wave B P1.C.6 Payroll — type-assert payroll repos into payroll's
	// dashboard repository interfaces. Each may be nil (non-postgres builds
	// tolerate). Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/
	// postgres/internal/adapter/entity/payroll_dashboard_assertions.go.
	if payrollRepos != nil {
		if payrollRepos.PayrollRun != nil {
			if q, ok := payrollRepos.PayrollRun.(payrolldash.PayrollRunDashboardRepository); ok {
				dashboardDeps.PayrollRun = q
			}
		}
		if payrollRepos.PayrollRemittance != nil {
			if q, ok := payrollRepos.PayrollRemittance.(payrolldash.PayrollRemittanceDashboardRepository); ok {
				dashboardDeps.PayrollRemittance = q
			}
		}
	}

	// Wave C P1.C.8 Expenditure — type-assert expenditure repos into expenditure's
	// dashboard repository interface. Nil under non-postgres builds —
	// expenditure dashboard use case tolerates nil repositories.
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/expenditure/expenditure_dashboard_assertions.go.
	if expenditureRepos != nil {
		if expenditureRepos.Expenditure != nil {
			if q, ok := expenditureRepos.Expenditure.(expendituredash.ExpenditureDashboardRepository); ok {
				dashboardDeps.Expenditure = q
			}
		}
	}

	// Wave C P1.C.9 Job — type-assert operation repos into job's dashboard
	// repository interfaces. The source aggregate is `operation` but the
	// service-layer package + umbrella field name are `Job` (per
	// wave-b-surface-map §P1.C.9). The JobActivity adapter may satisfy BOTH
	// JobActivityDashboardRepository AND the OPTIONAL JobActivityRecentRepository
	// (recent-activity widget). If the recent assertion fails, dashboardDeps.
	// JobActivityRecent stays nil and Execute degrades to an empty
	// recent-activity widget. Nil under non-postgres builds.
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/operation/job_dashboard_assertions.go.
	if operationRepos != nil {
		if operationRepos.Job != nil {
			if q, ok := operationRepos.Job.(jobdash.JobDashboardRepository); ok {
				dashboardDeps.Job = q
			}
		}
		if operationRepos.JobActivity != nil {
			if q, ok := operationRepos.JobActivity.(jobdash.JobActivityDashboardRepository); ok {
				dashboardDeps.JobActivity = q
			}
			if q, ok := operationRepos.JobActivity.(jobdash.JobActivityRecentRepository); ok {
				dashboardDeps.JobActivityRecent = q
			}
		}
	}

	// Wave C P1.C.11 Product — type-assert product repos into product's
	// dashboard repository interface. Nil under non-postgres builds — product
	// dashboard use case tolerates nil repositories.
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/product/product_dashboard_assertions.go.
	if productRepos != nil {
		if productRepos.Product != nil {
			if q, ok := productRepos.Product.(productdash.ProductDashboardRepository); ok {
				dashboardDeps.Product = q
			}
		}
	}

	// Wave C P1.C.12 Fulfillment — type-assert fulfillment repos into
	// fulfillment's dashboard repository interface. Nil under non-postgres
	// builds — fulfillment dashboard use case tolerates nil repositories.
	// Q-SDM-DASHBOARD-COMPILE-ASSERTIONS LOCKED — see contrib/postgres/
	// internal/adapter/fulfillment/fulfillment_dashboard_assertions.go.
	if fulfillmentRepos != nil {
		if fulfillmentRepos.Fulfillment != nil {
			if q, ok := fulfillmentRepos.Fulfillment.(fulfillmentdash.FulfillmentDashboardRepository); ok {
				dashboardDeps.Fulfillment = q
			}
		}
	}

	// Wave B P1.E.1-P1.E.5 — pass the same raw ledger reporting service
	// through as `any` to EVERY reporting sub-candidate. Each sub-candidate's
	// narrow `reporter` interface is unexported; the assertion happens
	// inside the leaf package via NewUseCases() / SetReporterFromAny. Apps
	// additionally call SetReporterFromAny after the espyna container
	// builds, since the table config lives app-side (see
	// apps/service-admin/internal/composition/ledger_reporting.go) and the
	// concrete adapter satisfying the narrow port may not be ready at
	// InitializeService time. Nil = graceful degradation on mock builds.
	//
	// One shared `ledgerReportingSvc any` value satisfies every group's
	// reporter port because the postgres `LedgerReportingAdapter` exposes
	// the union of all 13 (non-AR-aging) + 2 (AR aging) methods
	// structurally — splitting the assertion across 5 leaves (rather than
	// asserting a fat union here) keeps each leaf's port narrow.
	reportingDeps := &reportingusecases.Deps{
		DB:                     db,
		AuthorizationService:   authSvc,
		TranslationService:     i18nSvc,
		ARAgingReporter:        ledgerReportingSvc,
		APAgingReporter:        ledgerReportingSvc,
		GrossCashFlowReporter:  ledgerReportingSvc,
		StatementsReporter:     ledgerReportingSvc,
		DomainSpecificReporter: ledgerReportingSvc,
	}
	dashboardUC := dashboardusecases.NewDashboardUseCases(dashboardDeps)
	reportingUC := reportingusecases.NewReportingUseCases(reportingDeps)

	return service.NewServiceUseCases(auditUC, securityUC, serviceAuthUC, dashboardUC, reportingUC, deps), nil
}
