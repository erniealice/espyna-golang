// Package dashboard implements the read-only Schedule (event) Dashboard
// use case (Phase 6 — Pyeza dashboard block + per-app live dashboards plan).
//
// Wiring deferred: the orchestrator must construct
// *GetScheduleDashboardPageDataUseCase from the postgres event adapter and
// add it to the cyta event module.
package dashboard

import (
	"context"
	"time"

	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
)

// TimeBucket mirrors event.TimeBucket (count, not centavos).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// EventDashboardRepository is the slice of the postgres event adapter the
// dashboard use case needs. Implemented by *PostgresEventRepository.
type EventDashboardRepository interface {
	CountToday(ctx context.Context, workspaceID string, today time.Time) (int64, error)
	CountThisWeek(ctx context.Context, workspaceID string, weekStart time.Time) (int64, error)
	UpcomingByStartDate(ctx context.Context, workspaceID string, limit int32) ([]*eventpb.Event, error)
	CountByDay(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
	CountByTag(ctx context.Context, workspaceID string) (map[string]int64, error)
}

// ScheduleStats holds the four stat-card values for the schedule dashboard.
type ScheduleStats struct {
	Today          int64
	ThisWeek       int64
	ByTag          int64 // distinct tags in use this period
	UtilizationPct int64 // proxy: events-this-week (placeholder until a real capacity metric exists)
}

// GetScheduleDashboardPageDataRequest is the request shape.
type GetScheduleDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetScheduleDashboardPageDataResponse is the projection the view layer reads.
type GetScheduleDashboardPageDataResponse struct {
	Stats       ScheduleStats
	ByDayLabels []string
	ByDayValues []float64
	ByTag       map[string]int64
	Upcoming    []*eventpb.Event
}

// GetScheduleDashboardPageDataUseCase orchestrates the schedule projection.
type GetScheduleDashboardPageDataUseCase struct {
	repo EventDashboardRepository
}

// NewGetScheduleDashboardPageDataUseCase constructs the use case.
func NewGetScheduleDashboardPageDataUseCase(repo EventDashboardRepository) *GetScheduleDashboardPageDataUseCase {
	return &GetScheduleDashboardPageDataUseCase{repo: repo}
}

// Execute assembles the schedule dashboard response. Failures degrade gracefully.
func (uc *GetScheduleDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetScheduleDashboardPageDataRequest,
) (*GetScheduleDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetScheduleDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetScheduleDashboardPageDataResponse{ByTag: map[string]int64{}}
	if uc.repo == nil {
		return resp, nil
	}

	// Today (start of day) + week-start (Monday).
	loc := req.Now.Location()
	if loc == nil {
		loc = time.UTC
	}
	today := time.Date(req.Now.Year(), req.Now.Month(), req.Now.Day(), 0, 0, 0, 0, loc)
	// time.Sunday == 0; shift so Monday is the start of the week.
	wd := int(today.Weekday())
	if wd == 0 {
		wd = 7
	}
	weekStart := today.AddDate(0, 0, -(wd - 1))

	if n, err := uc.repo.CountToday(ctx, req.WorkspaceID, today); err == nil {
		resp.Stats.Today = n
	}
	if n, err := uc.repo.CountThisWeek(ctx, req.WorkspaceID, weekStart); err == nil {
		resp.Stats.ThisWeek = n
		// UtilizationPct is a placeholder — see ScheduleStats doc.
		resp.Stats.UtilizationPct = n
	}
	if up, err := uc.repo.UpcomingByStartDate(ctx, req.WorkspaceID, 5); err == nil {
		resp.Upcoming = up
	}
	if byTag, err := uc.repo.CountByTag(ctx, req.WorkspaceID); err == nil && byTag != nil {
		resp.ByTag = byTag
		resp.Stats.ByTag = int64(len(byTag))
	}

	// 14-day events-by-day trend ending today.
	from := today.AddDate(0, 0, -13)
	if buckets, err := uc.repo.CountByDay(ctx, req.WorkspaceID, from, today); err == nil {
		resp.ByDayLabels = make([]string, 0, len(buckets))
		resp.ByDayValues = make([]float64, 0, len(buckets))
		for _, b := range buckets {
			resp.ByDayLabels = append(resp.ByDayLabels, b.Period.Format("Jan 2"))
			resp.ByDayValues = append(resp.ByDayValues, float64(b.Value))
		}
	}

	return resp, nil
}
