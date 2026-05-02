// Package dashboard implements the read-only Expenditure Dashboard use case
// (Phase 5 — Pyeza dashboard block + per-app live dashboards plan). One use
// case serves both Purchase (kind="purchase") and Expense (kind="expense")
// surfaces — the request carries the Kind discriminator.
//
// Wiring deferred: the orchestrator must construct
// *GetExpenditureDashboardPageDataUseCase from the postgres expenditure
// adapter and add it to the centymo expenditure module.
package dashboard

import (
	"context"
	"time"

	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// TimeBucket mirrors expenditure.TimeBucket (centavos).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// TopSupplierRow mirrors expenditure.TopSupplierRow (centavos).
type TopSupplierRow struct {
	SupplierID   string
	SupplierName string
	Total        int64
}

// ExpenditureDashboardQueries is the slice of the postgres expenditure
// adapter the dashboard use case needs. Implemented by
// *PostgresExpenditureRepository.
type ExpenditureDashboardQueries interface {
	CountByStatus(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	SumOpenByStatus(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	TopBySupplier(ctx context.Context, workspaceID string, kind string, limit int32) ([]TopSupplierRow, error)
	RecentByDate(ctx context.Context, workspaceID string, kind string, limit int32) ([]*expenditurepb.Expenditure, error)
	SumByMonth(ctx context.Context, workspaceID string, kind string, from, to time.Time) ([]TimeBucket, error)
	SumByCategory(ctx context.Context, workspaceID string, kind string, from, to time.Time) (map[string]int64, error)
}

// Stats holds tile values for either purchase or expense surface (centavos).
type Stats struct {
	OpenCount       int64 // Open POs / Pending Approval (count, not centavos)
	AwaitingCount   int64 // Awaiting Receipt / Approved (count)
	TotalMTD        int64 // Spent MTD / Approved MTD (centavos)
	ReimbursableMTD int64 // For expense surface only (centavos)
	TopSupplierName string
	TopSupplierTotal int64 // centavos
	CategoryCount   int64 // distinct categories used (expense surface)
}

// GetExpenditureDashboardPageDataRequest is the request shape.
type GetExpenditureDashboardPageDataRequest struct {
	WorkspaceID string
	Kind        string // "purchase" | "expense"
	Now         time.Time
}

// GetExpenditureDashboardPageDataResponse is the projection the view layer reads.
type GetExpenditureDashboardPageDataResponse struct {
	Stats Stats

	// Spend per month (12 months) for chart-bar.
	MonthLabels []string
	MonthValues []float64 // centavos

	TopSuppliers []TopSupplierRow

	Recent []*expenditurepb.Expenditure

	// Categories — only populated for expense surface.
	CategoryLabels []string
	CategoryValues []float64 // centavos
}

// GetExpenditureDashboardPageDataUseCase orchestrates the projection.
type GetExpenditureDashboardPageDataUseCase struct {
	expenditures ExpenditureDashboardQueries
}

// NewGetExpenditureDashboardPageDataUseCase constructs the use case.
func NewGetExpenditureDashboardPageDataUseCase(
	expenditures ExpenditureDashboardQueries,
) *GetExpenditureDashboardPageDataUseCase {
	return &GetExpenditureDashboardPageDataUseCase{expenditures: expenditures}
}

// Execute assembles the dashboard response. Failures degrade gracefully.
func (uc *GetExpenditureDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetExpenditureDashboardPageDataRequest,
) (*GetExpenditureDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetExpenditureDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	if req.Kind == "" {
		req.Kind = "purchase"
	}
	resp := &GetExpenditureDashboardPageDataResponse{}

	if uc.expenditures == nil {
		return resp, nil
	}

	if byStatus, err := uc.expenditures.CountByStatus(ctx, req.WorkspaceID, req.Kind); err == nil {
		// Open: pending_approval, draft, approved (not yet paid). Awaiting receipt: approved.
		// Reuse the same map for both surfaces — view layer picks which fields to surface.
		resp.Stats.OpenCount = byStatus["pending_approval"] + byStatus["draft"] + byStatus["approved"]
		resp.Stats.AwaitingCount = byStatus["approved"]
	}

	// Month bucket setup: 12 months ending current month.
	monthEnd := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthStart := monthEnd.AddDate(0, -11, 0)
	if buckets, err := uc.expenditures.SumByMonth(ctx, req.WorkspaceID, req.Kind, monthStart, monthEnd); err == nil {
		resp.MonthLabels = make([]string, 0, len(buckets))
		resp.MonthValues = make([]float64, 0, len(buckets))
		for _, b := range buckets {
			resp.MonthLabels = append(resp.MonthLabels, b.Period.Format("Jan"))
			resp.MonthValues = append(resp.MonthValues, float64(b.Value))
		}
	}

	// Spent MTD: current month bucket.
	mtdStart := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
	mtdEnd := mtdStart.AddDate(0, 1, 0)
	if mtdBuckets, err := uc.expenditures.SumByMonth(ctx, req.WorkspaceID, req.Kind, mtdStart, mtdEnd); err == nil {
		var total int64
		for _, b := range mtdBuckets {
			total += b.Value
		}
		resp.Stats.TotalMTD = total
	}

	if top, err := uc.expenditures.TopBySupplier(ctx, req.WorkspaceID, req.Kind, 5); err == nil {
		resp.TopSuppliers = top
		if len(top) > 0 {
			resp.Stats.TopSupplierName = top[0].SupplierName
			resp.Stats.TopSupplierTotal = top[0].Total
		}
	}

	if recents, err := uc.expenditures.RecentByDate(ctx, req.WorkspaceID, req.Kind, 5); err == nil {
		resp.Recent = recents
	}

	// Category breakdown — primarily for expense surface; populated for both
	// (purchase view simply ignores the field).
	if cats, err := uc.expenditures.SumByCategory(ctx, req.WorkspaceID, req.Kind, mtdStart, mtdEnd); err == nil {
		resp.Stats.CategoryCount = int64(len(cats))
		resp.CategoryLabels = make([]string, 0, len(cats))
		resp.CategoryValues = make([]float64, 0, len(cats))
		for k, v := range cats {
			resp.CategoryLabels = append(resp.CategoryLabels, k)
			resp.CategoryValues = append(resp.CategoryValues, float64(v))
			if req.Kind == "expense" {
				resp.Stats.ReimbursableMTD += v
			}
		}
	}

	return resp, nil
}
