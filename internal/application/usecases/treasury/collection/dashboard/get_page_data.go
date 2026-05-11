// Package dashboard implements the read-only Cash (collection) Dashboard
// use case (Phase 5 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetCashDashboardPageDataUseCase from the postgres treasury adapters
// and add it to the centymo container init for the collection module.
package dashboard

import (
	"context"
	"time"

	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// TimeBucket mirrors treasury.TimeBucket (centavos).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// CollectionDashboardQueries is the slice of the postgres collection adapter
// the dashboard use case needs. Implemented by *PostgresCollectionRepository.
type CollectionDashboardQueries interface {
	SumPending(ctx context.Context, workspaceID string) (int64, error)
	SumOverdue(ctx context.Context, workspaceID string, asOf time.Time) (int64, error)
	SumCollectedToday(ctx context.Context, workspaceID string, today time.Time) (int64, error)
	SumByModeWeek(ctx context.Context, workspaceID string, weekStart time.Time) (map[string]int64, error)
	RecentByDate(ctx context.Context, workspaceID string, limit int32) ([]*collectionpb.Collection, error)
	SumByDayLast30(ctx context.Context, workspaceID string, asOf time.Time) ([]TimeBucket, error)
}

// CashStats holds the four stat-card values for the dashboard. Centavos.
type CashStats struct {
	Pending           int64
	Overdue           int64
	CollectedToday    int64
	CollectedThisWeek int64
}

// GetCashDashboardPageDataRequest is the request shape.
type GetCashDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetCashDashboardPageDataResponse is the projection the view layer reads.
type GetCashDashboardPageDataResponse struct {
	Stats GetCashStats

	// Daily series (last 30 days) for chart-line.
	DailyLabels []string  // "May 02"
	DailyValues []float64 // centavos

	// Payment-mode mix (this week) for chart-pie.
	ModeLabels []string
	ModeValues []float64 // centavos

	// Recent collections.
	Recent []*collectionpb.Collection
}

// GetCashStats wraps the stat values to keep the response struct shape
// parallel to the loan dashboard (Stats field of named type).
type GetCashStats = CashStats

// GetCashDashboardPageDataUseCase orchestrates the cash dashboard projection.
type GetCashDashboardPageDataUseCase struct {
	collections CollectionDashboardQueries
}

// NewGetCashDashboardPageDataUseCase constructs the use case.
func NewGetCashDashboardPageDataUseCase(
	collections CollectionDashboardQueries,
) *GetCashDashboardPageDataUseCase {
	return &GetCashDashboardPageDataUseCase{collections: collections}
}

// Execute assembles the cash dashboard response. Failures degrade gracefully.
func (uc *GetCashDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetCashDashboardPageDataRequest,
) (*GetCashDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetCashDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetCashDashboardPageDataResponse{}

	if uc.collections == nil {
		return resp, nil
	}

	if pending, err := uc.collections.SumPending(ctx, req.WorkspaceID); err == nil {
		resp.Stats.Pending = pending
	}
	if overdue, err := uc.collections.SumOverdue(ctx, req.WorkspaceID, req.Now); err == nil {
		resp.Stats.Overdue = overdue
	}
	if today, err := uc.collections.SumCollectedToday(ctx, req.WorkspaceID, req.Now); err == nil {
		resp.Stats.CollectedToday = today
	}

	// Week start = Monday of the current week.
	weekStart := startOfWeek(req.Now)
	if byMode, err := uc.collections.SumByModeWeek(ctx, req.WorkspaceID, weekStart); err == nil {
		var weekTotal int64
		labels := make([]string, 0, len(byMode))
		values := make([]float64, 0, len(byMode))
		for k, v := range byMode {
			labels = append(labels, k)
			values = append(values, float64(v))
			weekTotal += v
		}
		resp.ModeLabels = labels
		resp.ModeValues = values
		resp.Stats.CollectedThisWeek = weekTotal
	}

	if buckets, err := uc.collections.SumByDayLast30(ctx, req.WorkspaceID, req.Now); err == nil {
		resp.DailyLabels = make([]string, 0, len(buckets))
		resp.DailyValues = make([]float64, 0, len(buckets))
		for _, b := range buckets {
			resp.DailyLabels = append(resp.DailyLabels, b.Period.Format("Jan 02"))
			resp.DailyValues = append(resp.DailyValues, float64(b.Value))
		}
	}

	if recents, err := uc.collections.RecentByDate(ctx, req.WorkspaceID, 5); err == nil {
		resp.Recent = recents
	}

	return resp, nil
}

// startOfWeek returns the Monday 00:00 in the same location as t.
func startOfWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // treat Sunday as last day of week
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -(wd - 1))
}
