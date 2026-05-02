// Package dashboard implements the read-only Loan Dashboard use case
// (Phase 2 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetLoanDashboardPageDataUseCase from the postgres treasury adapters
// and add it to TreasuryUseCases (see usecases/treasury/usecases.go).
package dashboard

import (
	"context"
	"time"

	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

// LoanSlice mirrors treasury.LoanSlice in shape — duplicated here to avoid
// importing the postgres adapter from the application layer (clean-arch).
type LoanSlice struct {
	ID               string
	LoanNumber       string
	LenderName       string
	RemainingBalance int64
	PrincipalAmount  int64
	Status           string
}

// TimeBucket mirrors treasury.TimeBucket.
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

// LoanStats holds the four stat-card values for the dashboard. Centavos.
type LoanStats struct {
	TotalOutstanding int64
	InterestYTD      int64
	PaymentsDue30    int64 // sum (in centavos) of remaining balance for loans maturing in 30d
	DefaultedCount   int64
	ActiveCount      int64
	CompletedCount   int64
}

// GetLoanDashboardPageDataRequest is the request shape.
type GetLoanDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetLoanDashboardPageDataResponse is the projection the view layer reads.
type GetLoanDashboardPageDataResponse struct {
	Stats          LoanStats
	TrendLabels    []string
	TrendValues    []float64
	TopLoans       []LoanSlice
	RecentPayments []*loanpaymentpb.LoanPayment
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

// Execute assembles the loan dashboard response. Failures degrade gracefully.
func (uc *GetLoanDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetLoanDashboardPageDataRequest,
) (*GetLoanDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetLoanDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetLoanDashboardPageDataResponse{}

	if uc.loans != nil {
		if outstanding, err := uc.loans.SumOutstanding(ctx, req.WorkspaceID); err == nil {
			resp.Stats.TotalOutstanding = outstanding
		}
		if interestYTD, err := uc.loans.SumInterestAccruedYTD(ctx, req.WorkspaceID, req.Now.Year()); err == nil {
			resp.Stats.InterestYTD = interestYTD
		}
		if byStatus, err := uc.loans.CountByStatus(ctx, req.WorkspaceID); err == nil {
			resp.Stats.DefaultedCount = byStatus["DEFAULTED"]
			resp.Stats.ActiveCount = byStatus["ACTIVE"]
			resp.Stats.CompletedCount = byStatus["COMPLETED"]
		}
		if top, err := uc.loans.TopByOutstanding(ctx, req.WorkspaceID, 5); err == nil {
			resp.TopLoans = top
		}

		// 6-month outstanding-principal trend ending now.
		from := req.Now.AddDate(0, -5, 0)
		from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if buckets, err := uc.loans.OutstandingPrincipalByMonth(ctx, req.WorkspaceID, from, to); err == nil {
			resp.TrendLabels = make([]string, 0, len(buckets))
			resp.TrendValues = make([]float64, 0, len(buckets))
			for _, b := range buckets {
				resp.TrendLabels = append(resp.TrendLabels, b.Period.Format("Jan"))
				resp.TrendValues = append(resp.TrendValues, float64(b.Value))
			}
		}
	}

	if uc.payments != nil {
		if due, err := uc.payments.SumDueWithin(ctx, req.WorkspaceID, 30); err == nil {
			resp.Stats.PaymentsDue30 = due
		}
		if recents, err := uc.payments.RecentByLoan(ctx, req.WorkspaceID, 5); err == nil {
			resp.RecentPayments = recents
		}
	}

	return resp, nil
}
