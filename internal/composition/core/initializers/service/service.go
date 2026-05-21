// Package service hosts the initializers/service/ sub-package — the strict
// 1:1 mirror of proto/v1/service/. Each file in this package wires one
// service-layer use-case sub-aggregate (audit, security, auth, dashboard,
// reporting). The umbrella factory InitializeAll composes them into a
// *service.ServiceUseCases.
//
// Per docs/plan/20260521-composition-reshape/ Q-CR7 + Q-CR8, this package
// replaces the flat composition/core/initializers/service.go (300 LOC) and
// the composition/{audit,auth,security}/ helper dirs.
package service

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/dashboard"
	svcusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeAll wires every service-driven use case sub-aggregate.
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
//
// Note: the entityAuth *entityauth.UseCases parameter from the OLD
// InitializeService signature is REMOVED — Option B builds it internally
// inside initServiceAuth (auth.go). txSvc and idSvc are added because
// initServiceAuth needs them.
func InitializeAll(
	db *sql.DB,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	txSvc ports.Transactor,
	idSvc ports.IDGenerator,
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
) (*svcusecases.ServiceUseCases, error) {
	deps := &svcusecases.Deps{
		DB:         db,
		Authorizer: authSvc,
		Translator: i18nSvc,
	}

	auditUC := initServiceAudit(db, authSvc, i18nSvc)
	securityUC := initServiceSecurity(db, i18nSvc)
	authUC := initServiceAuth(entityRepos, deps, txSvc, i18nSvc, idSvc)
	dashboardUC := initServiceDashboard(db, authSvc, i18nSvc, entityRepos, ledgerRepos, payrollRepos, treasuryRepos, expenditureRepos, operationRepos, productRepos, fulfillmentRepos, scheduleEntityDash)
	reportingUC := initServiceReporting(db, authSvc, i18nSvc, ledgerReportingSvc)

	return svcusecases.NewServiceUseCases(auditUC, securityUC, authUC, dashboardUC, reportingUC, deps), nil
}
