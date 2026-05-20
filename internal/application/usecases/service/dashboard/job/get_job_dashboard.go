package job

import (
	"context"
	"time"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	jobdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/job"
)

// JobRisk mirrors the postgres-adapter JobRisk — kept as a Go-only repository
// return type. CompletionPct is 0..100. DateEnd is the planned_end (falling
// back to due_date when planned_end is NULL).
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `JobDashboardRepository` adapter MUST return EXACTLY this
// named type (via `type JobRisk = jobdash.JobRisk` alias on the adapter
// side). Returning the adapter package's own `operation.JobRisk` would
// silently fail the runtime type assertion in `initializers/service.go`
// (Go interface satisfaction requires exact named return type match). See
// `contrib/postgres/internal/adapter/operation/job_dashboard_assertions.go`
// for the compile-time guard.
type JobRisk struct {
	JobID         string
	Code          string
	Name          string
	CompletionPct float64
	DateEnd       time.Time
}

// TimeBucket mirrors the postgres-adapter TimeBucket. Value semantics depend
// on the producing method (e.g. SumHoursByWeek returns centi-hours; ÷100 to
// display fractional hours).
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `JobActivityDashboardRepository` adapter MUST return EXACTLY
// this named type via the `type TimeBucket = jobdash.TimeBucket` alias.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// JobDashboardRepository is the slice of the postgres job adapter the
// dashboard use case consumes. The postgres adapter `PostgresJobRepository`
// satisfies it — see
// `contrib/postgres/internal/adapter/operation/job_dashboard_assertions.go`
// for the compile-time guarantee.
type JobDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	UpcomingDeadlines(ctx context.Context, workspaceID string, days int, limit int32) ([]*jobpb.Job, error)
	TopByCompletionRisk(ctx context.Context, workspaceID string, limit int32) ([]JobRisk, error)
}

// JobActivityDashboardRepository is the slice of the postgres job_activity
// adapter the dashboard use case consumes. The postgres adapter
// `PostgresJobActivityRepository` satisfies it.
type JobActivityDashboardRepository interface {
	SumHoursByWeek(ctx context.Context, workspaceID string, weeks int) ([]TimeBucket, error)
}

// JobActivityRecentRepository is an optional slice — when non-nil the use
// case includes recent activity in the Recent Activity widget. The default
// JobActivity proto repo exposes ListJobActivities, but to keep the dashboard
// use case independent of paging concerns we only declare the narrow
// recent-activity helper here.
type JobActivityRecentRepository interface {
	RecentActivity(ctx context.Context, workspaceID string, limit int32) ([]*jobactivitypb.JobActivity, error)
}

// GetJobDashboardRepositories groups the per-repository dependencies the
// service-layer job dashboard composes. Any sub-repository may be nil when
// the postgres build tag is inactive (or the type assertion in the
// initializer fails) — the Execute method tolerates nil repositories.
type GetJobDashboardRepositories struct {
	Job               JobDashboardRepository
	JobActivity       JobActivityDashboardRepository
	JobActivityRecent JobActivityRecentRepository
}

// GetJobDashboardUseCase composes the two operation aggregates (job +
// job_activity) into the service-layer job dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the job-dashboard repository
// composition that previously lived at `usecases/operation/dashboard/`. The
// relocation moves the proto contract out of the Go-only Request/Response
// shape and into the service-driven category.
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware. Matches the Admin/Equity pilot pattern.
type GetJobDashboardUseCase struct {
	repositories GetJobDashboardRepositories
}

// NewGetJobDashboardUseCase wires the use case from grouped dependencies.
func NewGetJobDashboardUseCase(
	repositories GetJobDashboardRepositories,
) *GetJobDashboardUseCase {
	return &GetJobDashboardUseCase{repositories: repositories}
}

// Execute runs the aggregate queries and assembles the proto response.
// Failures of individual aggregates degrade gracefully: missing data renders
// empty stats rather than blocking the dashboard.
func (uc *GetJobDashboardUseCase) Execute(
	ctx context.Context,
	req *jobdashpb.GetJobDashboardRequest,
) (*jobdashpb.GetJobDashboardResponse, error) {
	now := time.Now()
	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
		if req.GetNowMillis() != 0 {
			now = time.UnixMilli(req.GetNowMillis())
		}
	}

	resp := &jobdashpb.GetJobDashboardResponse{
		Success: true,
		Stats:   &jobdashpb.JobStats{},
	}

	if uc.repositories.Job != nil {
		// Active jobs (any creation date) — uses zero `since`.
		if byStatus, err := uc.repositories.Job.CountByStatus(ctx, workspaceID, time.Time{}); err == nil {
			resp.Stats.ActiveJobs = byStatus["JOB_STATUS_ACTIVE"] + byStatus["JOB_STATUS_RELEASED"]
		}
		// "Done this month" — COMPLETED count among jobs created since month-start.
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if byStatus, err := uc.repositories.Job.CountByStatus(ctx, workspaceID, monthStart); err == nil {
			resp.Stats.DoneThisMonth = byStatus["JOB_STATUS_COMPLETED"]
		}
		// Risk widget rows.
		if rows, err := uc.repositories.Job.TopByCompletionRisk(ctx, workspaceID, 5); err == nil {
			resp.RiskTopRows = make([]*jobdashpb.JobRiskRow, 0, len(rows))
			for _, r := range rows {
				row := &jobdashpb.JobRiskRow{
					JobId:         r.JobID,
					Code:          r.Code,
					Name:          r.Name,
					CompletionPct: r.CompletionPct,
				}
				if !r.DateEnd.IsZero() {
					row.DateEndMillis = r.DateEnd.UnixMilli()
				}
				resp.RiskTopRows = append(resp.RiskTopRows, row)
				// Overdue ≈ rows whose DateEnd is in the past.
				if !r.DateEnd.IsZero() && r.DateEnd.Before(now) {
					resp.Stats.OverdueJobs++
				}
			}
		}
		// Upcoming deadlines (next 14 days, top 5).
		if upc, err := uc.repositories.Job.UpcomingDeadlines(ctx, workspaceID, 14, 5); err == nil {
			resp.UpcomingDeadlines = upc
		}
	}

	if uc.repositories.JobActivity != nil {
		// 8-week hours-per-week trend.
		if buckets, err := uc.repositories.JobActivity.SumHoursByWeek(ctx, workspaceID, 8); err == nil {
			resp.TrendLabels = make([]string, 0, len(buckets))
			resp.TrendValues = make([]float64, 0, len(buckets))
			var thisWeekHours float64
			weekStart := startOfWeek(now)
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

	if uc.repositories.JobActivityRecent != nil {
		if recents, err := uc.repositories.JobActivityRecent.RecentActivity(ctx, workspaceID, 5); err == nil {
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
