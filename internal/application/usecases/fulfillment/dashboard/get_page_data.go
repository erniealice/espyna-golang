// Package dashboard implements the read-only Fulfillment Dashboard use case
// (Phase 3 — Pyeza dashboard block + per-app live dashboards plan).
//
// The use case orchestrates aggregate queries on the Fulfillment repository
// to produce the four-stat / three-widget projection the fayna fulfillment
// dashboard view consumes. It is read-only and does not depend on auth /
// transaction / translation services.
//
// Wiring: the orchestrator must construct
// *GetFulfillmentDashboardPageDataUseCase from the postgres
// PostgresFulfillmentRepository, which exposes the dashboard methods
// (CountByStatus / AvgFulfillmentTimeDays / RecentExceptions /
// DailyDeliveredLast30) as concrete-type methods. Wiring is **not yet**
// in place — deferred to the orchestrator follow-up.
package dashboard

import (
	"context"
	"time"

	fulfillmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// TimeBucket mirrors fulfillment.TimeBucket.
type TimeBucket struct {
	Period time.Time
	Value  int64
}

// FulfillmentDashboardQueries is the slice of the postgres fulfillment adapter
// the dashboard use case needs. Implemented by *PostgresFulfillmentRepository
// in contrib/postgres/internal/adapter/fulfillment/fulfillment_dashboard.go.
type FulfillmentDashboardQueries interface {
	CountByStatus(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	AvgFulfillmentTimeDays(ctx context.Context, workspaceID string, since time.Time) (float64, error)
	RecentExceptions(ctx context.Context, workspaceID string, limit int32) ([]*fulfillmentpb.Fulfillment, error)
	DailyDeliveredLast30(ctx context.Context, workspaceID string, asOf time.Time) ([]TimeBucket, error)
}

// FulfillmentStats holds the four stat-card values for the Fulfillment dashboard.
type FulfillmentStats struct {
	Pending          int64
	InTransit        int64
	DeliveredToday   int64
	Exceptions       int64
	AvgFulfillDays   float64 // optional secondary metric (days)
}

// GetFulfillmentDashboardPageDataRequest is the request shape.
type GetFulfillmentDashboardPageDataRequest struct {
	WorkspaceID string
	Now         time.Time
}

// GetFulfillmentDashboardPageDataResponse is the projection the view layer reads.
type GetFulfillmentDashboardPageDataResponse struct {
	Stats             FulfillmentStats
	StatusMixLabels   []string  // for the donut widget
	StatusMixValues   []float64 // counts per status
	TrendLabels       []string  // 30-day daily-delivered chart
	TrendValues       []float64
	RecentExceptions  []*fulfillmentpb.Fulfillment
}

// GetFulfillmentDashboardPageDataUseCase orchestrates the Fulfillment dashboard
// projection.
type GetFulfillmentDashboardPageDataUseCase struct {
	q FulfillmentDashboardQueries
}

// NewGetFulfillmentDashboardPageDataUseCase constructs the use case.
func NewGetFulfillmentDashboardPageDataUseCase(
	q FulfillmentDashboardQueries,
) *GetFulfillmentDashboardPageDataUseCase {
	return &GetFulfillmentDashboardPageDataUseCase{q: q}
}

// Execute runs the aggregate queries and assembles the response. Failures of
// individual aggregates degrade gracefully.
func (uc *GetFulfillmentDashboardPageDataUseCase) Execute(
	ctx context.Context,
	req *GetFulfillmentDashboardPageDataRequest,
) (*GetFulfillmentDashboardPageDataResponse, error) {
	if req == nil {
		req = &GetFulfillmentDashboardPageDataRequest{}
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	resp := &GetFulfillmentDashboardPageDataResponse{}
	if uc.q == nil {
		return resp, nil
	}

	// Status counts (any time) drive the four stats and the donut.
	if byStatus, err := uc.q.CountByStatus(ctx, req.WorkspaceID, time.Time{}); err == nil && byStatus != nil {
		resp.Stats.Pending = byStatus["PENDING"]
		resp.Stats.InTransit = byStatus["IN_TRANSIT"]
		resp.Stats.Exceptions = byStatus["FAILED"] + byStatus["CANCELLED"] + byStatus["EXCEPTION"]

		// Donut order: Pending, Ready, In Transit, Delivered, Partially,
		// Failed, Cancelled, Exception (skip empties).
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
	if buckets, err := uc.q.DailyDeliveredLast30(ctx, req.WorkspaceID, req.Now); err == nil {
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
	since := req.Now.AddDate(0, 0, -30)
	if avg, err := uc.q.AvgFulfillmentTimeDays(ctx, req.WorkspaceID, since); err == nil {
		resp.Stats.AvgFulfillDays = avg
	}

	// Recent exceptions (top 5).
	if recents, err := uc.q.RecentExceptions(ctx, req.WorkspaceID, 5); err == nil {
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
