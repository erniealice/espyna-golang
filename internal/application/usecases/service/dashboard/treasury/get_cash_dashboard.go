package treasury

import (
	"context"
	"sort"
	"time"

	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	treasurydashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/treasury"
)

// CollectionDashboardRepository is satisfied by PostgresCollectionRepository.
//
// Extension interface — the aggregate methods live on the postgres collection
// adapter; this package surfaces them as a Go interface the composition root
// assembles via type assertion.
type CollectionDashboardRepository interface {
	SumPending(ctx context.Context, workspaceID string) (int64, error)
	SumOverdue(ctx context.Context, workspaceID string, asOf time.Time) (int64, error)
	SumCollectedToday(ctx context.Context, workspaceID string, today time.Time) (int64, error)
	SumByModeWeek(ctx context.Context, workspaceID string, weekStart time.Time) (map[string]int64, error)
	RecentByDate(ctx context.Context, workspaceID string, limit int32) ([]*collectionpb.Collection, error)
	SumByDayLast30(ctx context.Context, workspaceID string, asOf time.Time) ([]TimeBucket, error)
}

// GetCashDashboardRepositories groups the per-repository dependencies the
// service-layer cash dashboard composes. Any sub-repository may be nil when
// the postgres build tag is inactive — Execute tolerates the nil case.
type GetCashDashboardRepositories struct {
	Collection CollectionDashboardRepository
}

// GetCashDashboardUseCase composes the collection aggregate into the
// service-layer cash dashboard projection.
type GetCashDashboardUseCase struct {
	repositories GetCashDashboardRepositories
}

// NewGetCashDashboardUseCase wires the use case from grouped dependencies.
func NewGetCashDashboardUseCase(
	repositories GetCashDashboardRepositories,
) *GetCashDashboardUseCase {
	return &GetCashDashboardUseCase{repositories: repositories}
}

// Execute assembles the cash dashboard proto response. Failures degrade
// gracefully — a nil collections adapter returns a zero-valued response.
//
// Per the codex review pattern (Schedule pilot P1 2026-05-20), the
// ModeLabels/ModeValues parallel slices come from a Go map; the keys are
// sorted before emission to keep proto output deterministic for cross-
// language clients.
func (uc *GetCashDashboardUseCase) Execute(
	ctx context.Context,
	req *treasurydashpb.GetCashDashboardRequest,
) (*treasurydashpb.GetCashDashboardResponse, error) {
	now := time.Now()
	if req != nil && req.GetNowMillis() != 0 {
		now = time.UnixMilli(req.GetNowMillis())
	}

	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
	}

	resp := &treasurydashpb.GetCashDashboardResponse{
		Success: true,
		Stats:   &treasurydashpb.CashStats{},
	}
	if uc.repositories.Collection == nil {
		return resp, nil
	}

	if pending, err := uc.repositories.Collection.SumPending(ctx, workspaceID); err == nil {
		resp.Stats.Pending = pending
	}
	if overdue, err := uc.repositories.Collection.SumOverdue(ctx, workspaceID, now); err == nil {
		resp.Stats.Overdue = overdue
	}
	if today, err := uc.repositories.Collection.SumCollectedToday(ctx, workspaceID, now); err == nil {
		resp.Stats.CollectedToday = today
	}

	// Week start = Monday of the current week.
	weekStart := startOfWeek(now)
	if byMode, err := uc.repositories.Collection.SumByModeWeek(ctx, workspaceID, weekStart); err == nil {
		modeKeys := make([]string, 0, len(byMode))
		for k := range byMode {
			modeKeys = append(modeKeys, k)
		}
		sort.Strings(modeKeys)

		var weekTotal int64
		labels := make([]string, 0, len(modeKeys))
		values := make([]float64, 0, len(modeKeys))
		for _, k := range modeKeys {
			v := byMode[k]
			labels = append(labels, k)
			values = append(values, float64(v))
			weekTotal += v
		}
		resp.ModeLabels = labels
		resp.ModeValues = values
		resp.Stats.CollectedThisWeek = weekTotal
	}

	if buckets, err := uc.repositories.Collection.SumByDayLast30(ctx, workspaceID, now); err == nil {
		resp.DailyLabels = make([]string, 0, len(buckets))
		resp.DailyValues = make([]float64, 0, len(buckets))
		for _, b := range buckets {
			resp.DailyLabels = append(resp.DailyLabels, b.Period.Format("Jan 02"))
			resp.DailyValues = append(resp.DailyValues, float64(b.Value))
		}
	}

	if recents, err := uc.repositories.Collection.RecentByDate(ctx, workspaceID, 5); err == nil {
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
