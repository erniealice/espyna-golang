// Package dashboard implements the read-only Loan Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetLoanDashboardPageDataUseCase from the postgres treasury adapters
// and add it to TreasuryUseCases (see usecases/treasury/usecases.go).
//
// Phase 0i: Execute takes/returns proto types (GetLoanDashboardRequest /
// GetLoanDashboardResponse). The old Go-struct Request/Response/LoanStats/
// LoanSlice/TimeBucket are deleted — proto-generated types replace them.
package dashboard

import (
	"context"
	"time"

	dashboardpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/dashboard"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// LoanSlice mirrors treasury.LoanSlice in shape — duplicated here to avoid
// importing the postgres adapter from the application layer (clean-arch).
// Kept as a Go-only type because it is the output of
// LoanDashboardQueries.TopByOutstanding.
type LoanSlice struct {
	ID               string
	LoanNumber       string
	LenderName       string
	RemainingBalance int64
	PrincipalAmount  int64
	Status           string
}

// TimeBucket mirrors treasury.TimeBucket.
// Kept as a Go-only type because it is the output of
// LoanDashboardQueries.OutstandingPrincipalByMonth.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// LoanDashboardQueries is the slice of the postgres loan adapter the dashboard
// use case needs. Implemented by *PostgresLoanRepository.
type LoanDashboardQueries interface {
	SumOutstanding(ctx context.Context, workspaceID string) (int64, error)
	SumInterestAccruedYTD(ctx context.Context, workspaceID string, year int) (int64, error)
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
	TopByOutstanding(ctx context.Context, workspaceID string, limit int32) ([]LoanSlice, error)
	OutstandingPrincipalByMonth(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
}

// LoanPaymentDashboardQueries is the slice of the postgres loan_payment
// adapter the dashboard use case needs.
type LoanPaymentDashboardQueries interface {
	SumDueWithin(ctx context.Context, workspaceID string, days int) (int64, error)
	RecentByLoan(ctx context.Context, workspaceID string, limit int32) ([]*loanpaymentpb.LoanPayment, error)
}

// GetLoanDashboardPageDataUseCase orchestrates the loan dashboard projection.
type GetLoanDashboardPageDataUseCase struct {
	loans    LoanDashboardQueries
	payments LoanPaymentDashboardQueries
}

// NewGetLoanDashboardPageDataUseCase constructs the use case.
func NewGetLoanDashboardPageDataUseCase(
	loans LoanDashboardQueries,
	payments LoanPaymentDashboardQueries,
) *GetLoanDashboardPageDataUseCase {
	return &GetLoanDashboardPageDataUseCase{
		loans:    loans,
		payments: payments,
	}
}

// Execute assembles the loan dashboard proto response. Failures degrade gracefully.
func (uc *GetLoanDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *dashboardpb.GetLoanDashboardRequest,
) (*dashboardpb.GetLoanDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &dashboardpb.GetLoanDashboardResponse{
		Success: true,
		Stats:   &dashboardpb.LoanStats{},
	}

	if uc.loans != nil {
		if outstanding, err := uc.loans.SumOutstanding(ctx, workspaceID); err == nil {
			resp.Stats.TotalOutstanding = outstanding
		}
		if interestYTD, err := uc.loans.SumInterestAccruedYTD(ctx, workspaceID, now.Year()); err == nil {
			resp.Stats.InterestYtd = interestYTD
		}
		if byStatus, err := uc.loans.CountByStatus(ctx, workspaceID); err == nil {
			resp.Stats.DefaultedCount = byStatus["DEFAULTED"]
			resp.Stats.ActiveCount = byStatus["ACTIVE"]
			resp.Stats.CompletedCount = byStatus["COMPLETED"]
		}
		if top, err := uc.loans.TopByOutstanding(ctx, workspaceID, 5); err == nil {
			for _, l := range top {
				resp.TopLoans = append(resp.TopLoans, &dashboardpb.LoanSlice{
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
		if buckets, err := uc.loans.OutstandingPrincipalByMonth(ctx, workspaceID, from, to); err == nil {
			resp.TrendLabels = make([]string, 0, len(buckets))
			resp.TrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.TrendLabels = append(resp.TrendLabels, b.Period.Format("Jan"))
				resp.TrendValues = append(resp.TrendValues, float64(b.Value))
			}
		}
	}

	if uc.payments != nil {
		if due, err := uc.payments.SumDueWithin(ctx, workspaceID, 30); err == nil {
			resp.Stats.PaymentsDue30 = due
		}
		if recents, err := uc.payments.RecentByLoan(ctx, workspaceID, 5); err == nil {
			resp.RecentPayments = recents
		}
	}

	return resp, nil
}
