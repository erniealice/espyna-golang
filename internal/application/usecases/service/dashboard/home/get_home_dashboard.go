package home

import (
	"context"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// ---------- Repository interfaces ----------

// HomeDashboardStatsRepository counts workspace members and roles.
// Satisfied by PostgresWorkspaceUserRepository.
type HomeDashboardStatsRepository interface {
	// HomeDashboardStats returns total, active, inactive user counts and
	// total active roles for the workspace.
	HomeDashboardStats(ctx context.Context, workspaceID string) (HomeDashboardStats, error)
}

// HomeDashboardActivityRepository queries recent workspace activity.
// Satisfied by PostgresWorkspaceUserRepository.
type HomeDashboardActivityRepository interface {
	// HomeRecentActivity returns the most recent user/role activity rows.
	HomeRecentActivity(ctx context.Context, workspaceID string, limit int32) ([]ActivityRow, error)
}

// HomeDashboardChartRepository queries user-creation time-series.
// Satisfied by PostgresWorkspaceUserRepository.
type HomeDashboardChartRepository interface {
	// HomeUserCreationsPerMonth returns per-month user-creation counts for
	// the last N months.
	HomeUserCreationsPerMonth(ctx context.Context, workspaceID string, months int32) ([]MonthCount, error)
}

// ---------- Return types (Go-only) ----------

// HomeDashboardStats holds the four stat-card values.
type HomeDashboardStats struct {
	TotalUsers    int
	ActiveUsers   int
	InactiveUsers int
	TotalRoles    int
}

// ActivityRow is one raw row from the recent-activity query. The view layer
// maps event_type to icons and labels.
type ActivityRow struct {
	EventType string
	Name      string
	EventDate time.Time
}

// MonthCount is one bar/point in the user-creation chart.
type MonthCount struct {
	Label string
	Count int
}

// HomeDashboardResult is the assembled output of the home dashboard use case.
type HomeDashboardResult struct {
	Stats    HomeDashboardStats
	Activity []ActivityRow
	Chart    []MonthCount
}

// ---------- Use case ----------

// GetHomeDashboardRepositories groups per-repository dependencies.
type GetHomeDashboardRepositories struct {
	Stats    HomeDashboardStatsRepository
	Activity HomeDashboardActivityRepository
	Chart    HomeDashboardChartRepository
}

// GetHomeDashboardServices groups application services.
type GetHomeDashboardServices struct {
	Translator ports.Translator
}

// GetHomeDashboardUseCase composes workspace_user + role queries into the
// home dashboard projection.
type GetHomeDashboardUseCase struct {
	repositories GetHomeDashboardRepositories
	services     GetHomeDashboardServices
}

// NewGetHomeDashboardUseCase wires the use case from grouped dependencies.
func NewGetHomeDashboardUseCase(
	repositories GetHomeDashboardRepositories,
	services GetHomeDashboardServices,
) *GetHomeDashboardUseCase {
	return &GetHomeDashboardUseCase{repositories: repositories, services: services}
}

// Execute fans out the three aggregate queries and assembles the result.
// Each branch is nil-safe so the dashboard degrades gracefully on non-postgres
// builds.
func (uc *GetHomeDashboardUseCase) Execute(
	ctx context.Context,
	workspaceID string,
) (*HomeDashboardResult, error) {
	result := &HomeDashboardResult{}

	// Stats: total/active/inactive users + total roles.
	if uc.repositories.Stats != nil {
		stats, err := uc.repositories.Stats.HomeDashboardStats(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		result.Stats = stats
	}

	// Recent activity: last 5 user/role events.
	if uc.repositories.Activity != nil {
		rows, err := uc.repositories.Activity.HomeRecentActivity(ctx, workspaceID, 5)
		if err != nil {
			return nil, err
		}
		result.Activity = rows
	}

	// Chart: user creations per month for last 12 months.
	if uc.repositories.Chart != nil {
		months, err := uc.repositories.Chart.HomeUserCreationsPerMonth(ctx, workspaceID, 12)
		if err != nil {
			return nil, err
		}
		result.Chart = months
	}

	return result, nil
}
