// Package dashboard implements the read-only Job Dashboard use case
// (Phase 3 — Pyeza dashboard block + per-app live dashboards plan).
//
// The use case orchestrates aggregate queries across the Job and JobActivity
// repositories to produce the four-stat / three-widget projection the fayna
// job dashboard view consumes. It is intentionally read-only and does not
// depend on the auth / transaction / translation services.
//
// Wiring: the orchestrator must construct *GetJobDashboardPageDataUseCase
// from the postgres operation adapters (PostgresJobRepository +
// PostgresJobActivityRepository), which expose the dashboard methods
// (CountByStatus / UpcomingDeadlines / TopByCompletionRisk / SumHoursByWeek)
// as concrete-type methods. See packages/espyna-golang/internal/application/
// usecases/operation/usecases.go for the existing container pattern; this
// dashboard use case is **not yet** wired — wiring is deferred to the
// orchestrator follow-up.
package dashboard

import (
	"context"
	"time"

	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

// JobRisk mirrors operation.JobRisk in shape — duplicated here to avoid
// importing the postgres adapter package from the application layer
// (clean-arch boundary).
type JobRisk struct {
	JobID         string
	Code          string
	Name          string
	CompletionPct float64
	DateEnd       time.Time
}

// TimeBucket mirrors operation.TimeBucket. Value semantics depend on the
// producing method (e.g. SumHoursByWeek returns centi-hours, ÷100 to display).
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// JobDashboardQueries is the slice of the postgres job adapter the dashboard
// use case needs. Implemented by *PostgresJobRepository in
// contrib/postgres/internal/adapter/operation/job_dashboard.go.
type JobDashboardQueries interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	UpcomingDeadlines(ctx context.Context, workspaceID string, days int, limit int32) ([]*jobpb.Job, error)
	TopByCompletionRisk(ctx context.Context, workspaceID string, limit int32) ([]JobRisk, error)
}

// JobActivityDashboardQueries is the slice of the postgres job_activity
// adapter the dashboard use case needs. Implemented by
// *PostgresJobActivityRepository in contrib/postgres/internal/adapter/
// operation/job_activity_dashboard.go.
type JobActivityDashboardQueries interface {
	SumHoursByWeek(ctx context.Context, workspaceID string, weeks int) ([]TimeBucket, error)
}

// JobActivityRecentQueries is an optional slice — when non-nil the use case
// includes recent activity in the Recent Activity widget. The default
// JobActivity proto repo exposes ListJobActivities, but to keep the dashboard
// use case independent of paging concerns we only declare the narrow
// recent-activity helper here. Concrete adapter wiring may bind a small
// shim that calls ListJobActivities under the hood.
type JobActivityRecentQueries interface {
	RecentActivity(ctx context.Context, workspaceID string, limit int32) ([]*jobactivitypb.JobActivity, error)
}

// JobStats holds the four stat-card values for the Job dashboard.
type JobStats struct {
	ActiveJobs       int64
	DoneThisMonth    int64
	OverdueJobs      int64
	HoursThisWeek    float64 // hours (already divided ÷100 from centi-hours)
}

// GetJobDashboardPageDataRequest is the request shape.
type GetJobDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetJobDashboardPageDataResponse is the projection the view layer reads.
type GetJobDashboardPageDataResponse struct {
	Stats             JobStats
	TrendLabels       []string
	TrendValues       []float64 // hours per week (already ÷100)
	UpcomingDeadlines []*jobpb.Job
	RiskTopRows       []JobRisk
	RecentActivity    []*jobactivitypb.JobActivity
}

// GetJobDashboardPageDataUseCase orchestrates the Job dashboard projection.
type GetJobDashboardPageDataUseCase struct {
	jobs       JobDashboardQueries
	activities JobActivityDashboardQueries
	recents    JobActivityRecentQueries // optional
}

// NewGetJobDashboardPageDataUseCase constructs the use case.
//
// The recents argument may be nil — when nil, the response's RecentActivity
// is left empty and the view renders an empty-state in the Recent Activity
// widget.
func NewGetJobDashboardPageDataUseCase(
	jobs JobDashboardQueries,
	activities JobActivityDashboardQueries,
	recents JobActivityRecentQueries,
) *GetJobDashboardPageDataUseCase {
	return &GetJobDashboardPageDataUseCase{
		jobs:       jobs,
		activities: activities,
		recents:    recents,
	}
}

// Execute runs the aggregate queries and assembles the response. Failures of
// individual aggregates degrade gracefully: missing data renders empty stats
// rather than blocking the dashboard.
func (uc *GetJobDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetJobDashboardPageDataRequest,
) (*GetJobDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetJobDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetJobDashboardPageDataResponse{}

	if uc.jobs != nil {
		// Active jobs (any creation date) — uses zero `since`.
		if byStatus, err := uc.jobs.CountByStatus(ctx, req.WorkspaceID, time.Time{}); err == nil {
			resp.Stats.ActiveJobs = byStatus["JOB_STATUS_ACTIVE"] + byStatus["JOB_STATUS_RELEASED"]
		}
		// "Done this month" = COMPLETED count among jobs created since
		// month-start. (Phase 3+ enhancement: switch to date_completed when
		// the schema gains a settled completion timestamp; today date_modified
		// is the closest proxy and the existing CountByStatus uses
		// date_created as the cutoff — accepted approximation.)
		monthStart := time.Date(req.Now.Year(), req.Now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if byStatus, err := uc.jobs.CountByStatus(ctx, req.WorkspaceID, monthStart); err == nil {
			resp.Stats.DoneThisMonth = byStatus["JOB_STATUS_COMPLETED"]
		}
		// Risk widget rows.
		if rows, err := uc.jobs.TopByCompletionRisk(ctx, req.WorkspaceID, 5); err == nil {
			resp.RiskTopRows = rows
			// Overdue ≈ rows whose DateEnd is in the past.
			now := req.Now
			for _, r := range rows {
				if !r.DateEnd.IsZero() && r.DateEnd.Before(now) {
					resp.Stats.OverdueJobs++
				}
			}
		}
		// Upcoming deadlines (next 14 days, top 5).
		if upc, err := uc.jobs.UpcomingDeadlines(ctx, req.WorkspaceID, 14, 5); err == nil {
			resp.UpcomingDeadlines = upc
		}
	}

	if uc.activities != nil {
		// 8-week hours-per-week trend.
		if buckets, err := uc.activities.SumHoursByWeek(ctx, req.WorkspaceID, 8); err == nil {
			resp.TrendLabels = make([]string, 0, len(buckets))
			resp.TrendValues = make([]float64, 0, len(buckets))
			var thisWeekHours float64
			weekStart := startOfWeek(req.Now)
			for _, b := range buckets {
				resp.TrendLabels = append(resp.TrendLabels, b.Period.Format("Jan 2"))
				hours := float64(b.Value) / 100.0 // centi-hours → hours
				resp.TrendValues = append(resp.TrendValues, hours)
				if !b.Period.Before(weekStart) {
					thisWeekHours += hours
				}
			}
			resp.Stats.HoursThisWeek = thisWeekHours
		}
	}

	if uc.recents != nil {
		if recents, err := uc.recents.RecentActivity(ctx, req.WorkspaceID, 5); err == nil {
			resp.RecentActivity = recents
		}
	}

	return resp, nil
}

// startOfWeek returns the ISO-week-start (Monday 00:00 UTC) for t.
func startOfWeek(t time.Time) time.Time {
	t = t.UTC()
	wd := int(t.Weekday())
	// Monday=1..Sunday=0 → shift to ISO (Mon=0..Sun=6).
	if wd == 0 {
		wd = 6
	} else {
		wd--
	}
	monday := time.Date(t.Year(), t.Month(), t.Day()-wd, 0, 0, 0, 0, time.UTC)
	return monday
}
