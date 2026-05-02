// Package dashboard implements the read-only Integration Dashboard use case
// (Phase 7 — Pyeza dashboard block + per-app live dashboards plan).
//
// The integration sidebar app is a *composition surface*. There is no first-
// class Integration entity in cyta or centymo, and there is no
// `integration` table the dashboard can aggregate over. The provider-specific
// adapters that exist today (PostgresIntegrationPaymentRepository, etc.)
// expose log-write methods (LogWebhook) but no aggregate read methods.
//
// This use case is therefore intentionally noop-by-default: it returns a
// zero-valued response unless / until provider stats hooks are wired in.
// The view degrades gracefully — empty stats, flat trend, empty tables —
// when the response is empty.
//
// # ORCHESTRATOR FOLLOW-UP
//
// To light this dashboard up with real data, three things need to happen:
//
//  1. Add aggregate methods to each provider's adapter (or a shared
//     IntegrationStatsRepository wrapping them all):
//     - CountByStatus(workspaceID) → map[string]int64
//     - RecentErrors(workspaceID, since, limit) → []ErrorEntry
//     - SyncEventsByDay(workspaceID, from, to) → []TimeBucket
//     - ProviderRows(workspaceID, limit) → []ProviderRow
//  2. Construct *GetIntegrationDashboardPageDataUseCase with those queries
//     in espyna's container.
//  3. Adapt the use-case response to the view-layer Request/Response shape
//     defined in hybra-golang/views/integration/dashboard.
//
// Until then, the dashboard renders dummy / zero values.
package dashboard

import (
	"context"
	"time"
)

// TimeBucket is a generic (period, value) tuple for time-series aggregates,
// shared with adapter-layer dashboard aggregates.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// ProviderRow is a per-provider summary for the dashboard "By provider" widget.
type ProviderRow struct {
	ID           string
	Name         string
	Status       string
	LastSync     time.Time
	EventsLast7d int64
}

// ErrorEntry is a single recent error row for the dashboard list widget.
type ErrorEntry struct {
	ID         string
	Provider   string
	Message    string
	OccurredAt time.Time
}

// IntegrationStatsQueries is the slice of aggregate queries the dashboard use
// case expects. Concrete implementations may hop across multiple per-provider
// repositories; the use case treats this as a single read-only port.
//
// Every method is nil-safe inside the use case — implementations may be added
// piecemeal as adapters land.
type IntegrationStatsQueries interface {
	CountByStatus(ctx context.Context, workspaceID string) (map[string]int64, error)
	RecentErrors(ctx context.Context, workspaceID string, since time.Time, limit int32) ([]ErrorEntry, error)
	SyncEventsByDay(ctx context.Context, workspaceID string, from, to time.Time) ([]TimeBucket, error)
	ProviderRows(ctx context.Context, workspaceID string, limit int32) ([]ProviderRow, error)
}

// GetIntegrationDashboardPageDataRequest is the request shape.
type GetIntegrationDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetIntegrationDashboardPageDataResponse is the projection the view layer reads.
// Numbers are absolute counts; the view formats for display.
type GetIntegrationDashboardPageDataResponse struct {
	TotalIntegrations  int64
	ActiveIntegrations int64
	ErrorsLast24h      int64
	Disconnected       int64

	TrendBuckets []TimeBucket // 7 entries, one per day
	Providers    []ProviderRow
	RecentErrors []ErrorEntry
}

// GetIntegrationDashboardPageDataUseCase orchestrates the read-only dashboard
// projection. The single-port wiring keeps the orchestrator thin.
type GetIntegrationDashboardPageDataUseCase struct {
	stats IntegrationStatsQueries
}

// NewGetIntegrationDashboardPageDataUseCase constructs the use case. Passing a
// nil queries slice is supported — the use case returns a zero-valued
// response.
func NewGetIntegrationDashboardPageDataUseCase(
	stats IntegrationStatsQueries,
) *GetIntegrationDashboardPageDataUseCase {
	return &GetIntegrationDashboardPageDataUseCase{stats: stats}
}

// Execute runs the aggregate queries and assembles the response. Failures of
// individual aggregates degrade gracefully: missing data renders empty stats /
// trend rather than blocking the dashboard.
func (uc *GetIntegrationDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetIntegrationDashboardPageDataRequest,
) (*GetIntegrationDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetIntegrationDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}

	resp := &GetIntegrationDashboardPageDataResponse{}

	// No queries wired — return empty response. The view renders empty state.
	if uc.stats == nil {
		return resp, nil
	}

	// Status counts → derived stats (nil-safe).
	if statusCounts, err := uc.stats.CountByStatus(ctx, req.WorkspaceID); err == nil && statusCounts != nil {
		for status, count := range statusCounts {
			resp.TotalIntegrations += count
			switch status {
			case "active":
				resp.ActiveIntegrations += count
			case "disconnected":
				resp.Disconnected += count
			}
		}
	}

	// Errors in the last 24h.
	since24h := req.Now.Add(-24 * time.Hour)
	if errs, err := uc.stats.RecentErrors(ctx, req.WorkspaceID, since24h, 50); err == nil {
		resp.ErrorsLast24h = int64(len(errs))
	}

	// Recent errors list (most recent 5, last 7 days).
	since7d := req.Now.AddDate(0, 0, -7)
	if errs, err := uc.stats.RecentErrors(ctx, req.WorkspaceID, since7d, 5); err == nil {
		resp.RecentErrors = errs
	}

	// 7-day sync events trend.
	if buckets, err := uc.stats.SyncEventsByDay(ctx, req.WorkspaceID, since7d, req.Now); err == nil {
		resp.TrendBuckets = buckets
	}

	// Top provider rows (5).
	if rows, err := uc.stats.ProviderRows(ctx, req.WorkspaceID, 5); err == nil {
		resp.Providers = rows
	}

	return resp, nil
}
