package ledger

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	grossprofit "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/reporting/gross_profit"
)

// LedgerRepositories groups all repository dependencies for ledger use cases.
type LedgerRepositories struct {
	ReportingService ports.LedgerReportingService
}

// UseCases contains all ledger-related use cases.
type UseCases struct {
	GetGrossProfitReport *grossprofit.GetGrossProfitReportUseCase
}

// NewUseCases creates all ledger use cases with proper constructor injection.
func NewUseCases(repositories LedgerRepositories) *UseCases {
	return &UseCases{
		GetGrossProfitReport: grossprofit.NewGetGrossProfitReportUseCase(repositories.ReportingService),
	}
}
