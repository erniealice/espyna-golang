package service

import (
	"database/sql"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	reportingusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/reporting"
)

// initServiceReporting wires the service-layer Reporting umbrella sub-aggregate.
//
// Wave B P1.E.1-P1.E.5 — pass the same raw ledger reporting service
// through as `any` to EVERY reporting sub-candidate. Each sub-candidate's
// narrow `reporter` interface is unexported; the assertion happens
// inside the leaf package via NewUseCases() / SetReporterFromAny. Apps
// additionally call SetReporterFromAny after the espyna container
// builds, since the table config lives app-side (see
// apps/service-admin/internal/composition/container.go, struct ~line 187, factory call ~line 759) and the
// concrete adapter satisfying the narrow port may not be ready at
// InitializeAll time. Nil = graceful degradation on mock builds.
//
// One shared `ledgerReportingSvc any` value satisfies every group's
// reporter port because the postgres `LedgerReportingAdapter` exposes
// the union of all 13 (non-AR-aging) + 2 (AR aging) methods
// structurally — splitting the assertion across 5 leaves (rather than
// asserting a fat union here) keeps each leaf's port narrow.
func initServiceReporting(
	db *sql.DB,
	authSvc ports.AuthorizationService,
	i18nSvc ports.TranslationService,
	ledgerReportingSvc any,
) *reportingusecases.ReportingUseCases {
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
	return reportingusecases.NewReportingUseCases(reportingDeps)
}
