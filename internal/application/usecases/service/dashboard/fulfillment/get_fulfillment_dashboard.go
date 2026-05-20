package fulfillment

import (
	"context"
	"time"

	fulfillmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
	fulfillmentdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/fulfillment"
)

// TimeBucket is a (period, value) tuple for time-series aggregates.
//
// For DailyDeliveredLast30, Value = count of deliveries on that day.
//
// **Named-type contract (Q-SDM-DASHBOARD-COMPILE-ASSERTIONS, LOCKED 2026-05-20):**
// the postgres `FulfillmentDashboardRepository` adapter MUST return EXACTLY
// this named type (via `type TimeBucket = fulfillmentdash.TimeBucket` alias
// on the adapter side). Returning the adapter package's own
// `fulfillment.TimeBucket` would silently fail the runtime type assertion in
// `initializers/service.go` (Go interface satisfaction requires exact named
// return type match). See
// `contrib/postgres/internal/adapter/fulfillment/fulfillment_dashboard_assertions.go`
// for the compile-time guard.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// FulfillmentDashboardRepository is the slice of the postgres fulfillment
// adapter the dashboard use case consumes. The postgres adapter
// `PostgresFulfillmentRepository` satisfies it — see
// `contrib/postgres/internal/adapter/fulfillment/fulfillment_dashboard_assertions.go`
// for the compile-time guarantee.
//
// Extension interface — the aggregate methods live on the postgres
// fulfillment adapter; this package surfaces them as a Go interface the
// composition root assembles via type assertion.
type FulfillmentDashboardRepository interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	AvgFulfillmentTimeDays(ctx context.Context, workspaceID string, since time.Time) (float64, error)
	RecentExceptions(ctx context.Context, workspaceID string, limit int32) ([]*fulfillmentpb.Fulfillment, error)
	DailyDeliveredLast30(ctx context.Context, workspaceID string, asOf time.Time) ([]TimeBucket, error)
}

// GetFulfillmentDashboardRepositories groups the per-repository dependencies
// the service-layer fulfillment dashboard composes. The single repository
// may be nil when the postgres build tag is inactive — the Execute method
// tolerates nil and returns a zero-valued response.
type GetFulfillmentDashboardRepositories struct {
	Fulfillment FulfillmentDashboardRepository
}

// GetFulfillmentDashboardUseCase composes the fulfillment aggregate into the
// service-layer dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the fulfillment-dashboard
// repository composition that previously lived at
// `usecases/fulfillment/dashboard/`. The relocation moves the proto contract
// out of the Go-only Request/Response shape and into the service-driven
// category, where it sits alongside the other dashboard candidates.
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware. Matches the Admin/Equity pilot pattern.
type GetFulfillmentDashboardUseCase struct {
	repositories GetFulfillmentDashboardRepositories
}

// NewGetFulfillmentDashboardUseCase wires the use case from grouped dependencies.
func NewGetFulfillmentDashboardUseCase(
	repositories GetFulfillmentDashboardRepositories,
) *GetFulfillmentDashboardUseCase {
	return &GetFulfillmentDashboardUseCase{repositories: repositories}
}

// Execute runs the aggregate queries and assembles the proto response.
// Failures of individual aggregates degrade gracefully — a nil fulfillment
// adapter returns a zero-valued response.
func (uc *GetFulfillmentDashboardUseCase) Execute(
	ctx context.Context,
	req *fulfillmentdashpb.GetFulfillmentDashboardRequest,
) (*fulfillmentdashpb.GetFulfillmentDashboardResponse, error) {
	now := time.Now()
	workspaceID := ""
	if req != nil {
		workspaceID = req.GetWorkspaceId()
		if req.GetNowMillis() != 0 {
			now = time.UnixMilli(req.GetNowMillis())
		}
	}

	resp := &fulfillmentdashpb.GetFulfillmentDashboardResponse{
		Success: true,
		Stats:   &fulfillmentdashpb.FulfillmentStats{},
	}
	if uc.repositories.Fulfillment == nil {
		return resp, nil
	}

	// Status counts (any time) drive the four stats and the donut.
	if byStatus, err := uc.repositories.Fulfillment.CountByStatus(ctx, workspaceID, time.Time{}); err == nil && byStatus != nil {
		resp.Stats.Pending = byStatus["PENDING"]
		resp.Stats.InTransit = byStatus["IN_TRANSIT"]
		resp.Stats.Exceptions = byStatus["FAILED"] + byStatus["CANCELLED"] + byStatus["EXCEPTION"]

		// Donut order: PENDING, READY, IN_TRANSIT, DELIVERED, PARTIALLY_DELIVERED,
		// FAILED, CANCELLED, EXCEPTION (skip empties). The fixed canonical order
		// already keeps proto output deterministic without a sort step.
		ordered := []string{"PENDING", "READY", "IN_TRANSIT", "DELIVERED", "PARTIALLY_DELIVERED", "FAILED", "CANCELLED", "EXCEPTION"}
		for _, k := range ordered {
			if v, ok := byStatus[k]; ok && v > 0 {
				resp.StatusMixLabels = append(resp.StatusMixLabels, displayStatus(k))
				resp.StatusMixValues = append(resp.StatusMixValues, float64(v))
			}
		}
	}

	// "Delivered today" — recompute via daily-delivered series so we share the
	// same query path the chart already needs. Sum the last bucket (today).
	if buckets, err := uc.repositories.Fulfillment.DailyDeliveredLast30(ctx, workspaceID, now); err == nil {
		resp.TrendLabels = make([]string, 0, len(buckets))
		resp.TrendValues = make([]float64, 0, len(buckets))
		for i, b := range buckets {
			resp.TrendLabels = append(resp.TrendLabels, b.Period.Format("Jan 2"))
			resp.TrendValues = append(resp.TrendValues, float64(b.Value))
			if i == len(buckets)-1 {
				resp.Stats.DeliveredToday = b.Value
			}
		}
	}

	// Average fulfillment time over the trailing 30 days.
	since := now.AddDate(0, 0, -30)
	if avg, err := uc.repositories.Fulfillment.AvgFulfillmentTimeDays(ctx, workspaceID, since); err == nil {
		resp.Stats.AvgFulfillDays = avg
	}

	// Recent exceptions (top 5).
	if recents, err := uc.repositories.Fulfillment.RecentExceptions(ctx, workspaceID, 5); err == nil {
		resp.RecentExceptions = recents
	}

	return resp, nil
}

// displayStatus is a small helper that converts the canonical SCREAMING_SNAKE
// to a Title Case display string. Real localization happens in the view via
// lyngua-driven labels.
func displayStatus(k string) string {
	switch k {
	case "PENDING":
		return "Pending"
	case "READY":
		return "Ready"
	case "IN_TRANSIT":
		return "In Transit"
	case "DELIVERED":
		return "Delivered"
	case "PARTIALLY_DELIVERED":
		return "Partial"
	case "FAILED":
		return "Failed"
	case "CANCELLED":
		return "Cancelled"
	case "EXCEPTION":
		return "Exception"
	}
	return k
}
