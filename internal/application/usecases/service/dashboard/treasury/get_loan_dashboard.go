package treasury

import (
	"context"
	"time"

	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
	treasurydashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/treasury"
)

// LoanSlice mirrors the treasury.LoanSlice row shape for the dashboard
// queries. Kept as a Go-only repository return type — the service-layer
// use case projects it onto the proto `LoanSlice` message.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `LoanDashboardRepository` adapter MUST return EXACTLY this
// named type (via `type LoanSlice = treasurydash.LoanSlice` alias on the
// adapter side). Returning the adapter package's own `treasury.LoanSlice`
// would silently fail the runtime type assertion in `initializers/service.go`
// (Go interface satisfaction requires exact named return type match). See
// `contrib/postgres/internal/adapter/treasury/treasury_dashboard_assertions.go`
// for the compile-time guard.
type LoanSlice struct {
	ID               string
	LoanNumber       string
	LenderName       string
	RemainingBalance int64
	PrincipalAmount  int64
	Status           string
}

// TimeBucket mirrors treasury.TimeBucket. Value semantics depend on the
// producing method (centavos for monetary buckets).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// LoanDashboardRepository is satisfied by PostgresLoanRepository.
//
// Extension interface — the aggregate methods live on the postgres loan
// adapter; this package surfaces them as a Go interface the composition root
// assembles via type assertion.
type LoanDashboardRepository interface {
	SumOutstanding(ctx context.Context, workspaceID string) (int64, error)
	SumInterestAccruedYTD(ctx context.Context, workspaceID string, year int) (int64, error)
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
	TopByOutstanding(ctx context.Context, workspaceID string, limit int32) ([]LoanSlice, error)
	OutstandingPrincipalByMonth(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
}

// LoanPaymentDashboardRepository is satisfied by PostgresLoanPaymentRepository.
type LoanPaymentDashboardRepository interface {
	SumDueWithin(ctx context.Context, workspaceID string, days int) (int64, error)
	RecentByLoan(ctx context.Context, workspaceID string, limit int32) ([]*loanpaymentpb.LoanPayment, error)
}

// GetLoanDashboardRepositories groups the per-repository dependencies the
// service-layer loan dashboard composes. Any sub-repository may be nil when
// the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories.
type GetLoanDashboardRepositories struct {
	Loan        LoanDashboardRepository
	LoanPayment LoanPaymentDashboardRepository
}

// GetLoanDashboardUseCase composes the loan + loan_payment aggregates into
// the service-layer loan dashboard projection.
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware. Matches the Admin/Location/Equity
// pilot pattern.
type GetLoanDashboardUseCase struct {
	repositories GetLoanDashboardRepositories
}

// NewGetLoanDashboardUseCase wires the use case from grouped dependencies.
func NewGetLoanDashboardUseCase(
	repositories GetLoanDashboardRepositories,
) *GetLoanDashboardUseCase {
	return &GetLoanDashboardUseCase{repositories: repositories}
}

// Execute assembles the loan dashboard proto response. Failures degrade
// gracefully.
func (uc *GetLoanDashboardUseCase) Execute(
	ctx context.Context,
	req *treasurydashpb.GetLoanDashboardRequest,
) (*treasurydashpb.GetLoanDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &treasurydashpb.GetLoanDashboardResponse{
		Success: true,
		Stats:   &treasurydashpb.LoanStats{},
	}

	if uc.repositories.Loan != nil {
		if outstanding, err := uc.repositories.Loan.SumOutstanding(ctx, workspaceID); err == nil {
			resp.Stats.TotalOutstanding = outstanding
		}
		if interestYTD, err := uc.repositories.Loan.SumInterestAccruedYTD(ctx, workspaceID, now.Year()); err == nil {
			resp.Stats.InterestYtd = interestYTD
		}
		if byStatus, err := uc.repositories.Loan.CountByStatus(ctx, workspaceID); err == nil {
			resp.Stats.DefaultedCount = byStatus["DEFAULTED"]
			resp.Stats.ActiveCount = byStatus["ACTIVE"]
			resp.Stats.CompletedCount = byStatus["COMPLETED"]
		}
		if top, err := uc.repositories.Loan.TopByOutstanding(ctx, workspaceID, 5); err == nil {
			for _, l := range top {
				resp.TopLoans = append(resp.TopLoans, &treasurydashpb.LoanSlice{
					Id:               l.ID,
					LoanNumber:       l.LoanNumber,
					LenderName:       l.LenderName,
					RemainingBalance: l.RemainingBalance,
					PrincipalAmount:  l.PrincipalAmount,
					Status:           l.Status,
				})
			}
		}

		// 6-month outstanding-principal trend ending now.
		from := now.AddDate(0, -5, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if buckets, err := uc.repositories.Loan.OutstandingPrincipalByMonth(ctx, workspaceID, from, to); err == nil {
			resp.TrendLabels = make([]string, 0, len(buckets))
			resp.TrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.TrendLabels = append(resp.TrendLabels, b.Period.Format("Jan"))
				resp.TrendValues = append(resp.TrendValues, float64(b.Value))
			}
		}
	}

	if uc.repositories.LoanPayment != nil {
		if due, err := uc.repositories.LoanPayment.SumDueWithin(ctx, workspaceID, 30); err == nil {
			resp.Stats.PaymentsDue30 = due
		}
		if recents, err := uc.repositories.LoanPayment.RecentByLoan(ctx, workspaceID, 5); err == nil {
			resp.RecentPayments = recents
		}
	}

	return resp, nil
}
