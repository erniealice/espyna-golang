package integration

import (
	"context"
	"time"

	integrationdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/domain/integration/dashboard"
	integrationdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/integration"
)

// GetIntegrationDashboardUseCase is the proto-shaped wrapper over the
// entity-layer `integrationdashboard.GetIntegrationDashboardPageDataUseCase`.
//
// It translates the proto Request defined in
// `proto/v1/service/dashboard/integration/dashboard.proto` into the Go-shaped
// Request the entity-layer use case carries, then projects the Go-shaped
// Response back into the proto Response. The wrapper does NOT re-implement
// the aggregation algorithm — it delegates to the entity-layer use case to
// preserve the noop-by-default contract documented at
// usecases/integration/dashboard/get_page_data.go.
//
// Note: no authcheck.Check call. Per hexagonal-rules.md §8 service-driven
// domains take a conditional subset of layers; dashboard reads are
// authenticated by the upstream HTTP view middleware (the dashboard URL
// resolves through hybra's RegisterRoutes only when the session is
// authenticated). Future Wave B candidates that surface sensitive data via
// the dashboard MAY revisit this omission.
type GetIntegrationDashboardUseCase struct {
	entity *integrationdashboard.GetIntegrationDashboardPageDataUseCase
}

// NewGetIntegrationDashboardUseCase wires the wrapper.
//
// The entity-layer use case is constructed with a nil
// `IntegrationStatsQueries` port — matching the current entity-layer
// wiring at usecases/integration/usecases.go:107. The nil-safe Execute on
// the entity layer returns an empty response under that condition, so the
// service-layer wrapper renders empty-state proto responses by design
// until provider stats hooks are wired (see the ORCHESTRATOR FOLLOW-UP
// section in usecases/integration/dashboard/get_page_data.go).
func NewGetIntegrationDashboardUseCase() *GetIntegrationDashboardUseCase {
	return &GetIntegrationDashboardUseCase{
		entity: integrationdashboard.NewGetIntegrationDashboardPageDataUseCase(nil),
	}
}

// Execute runs the entity-layer dashboard aggregation with proto-shaped IO.
//
// The proto NowMillis is unix milliseconds; zero or unset falls back to
// server time (matches the entity-layer zero-value time.Time semantics).
// Failure modes degrade gracefully — the entity-layer use case never
// returns an error in the current shape, but the wrapper is defensive in
// case future provider stats hooks introduce error paths: any error from
// the entity layer propagates unchanged so callers (hybra view) can
// decide whether to surface or swallow.
func (uc *GetIntegrationDashboardUseCase) Execute(
	ctx context.Context,
	req *integrationdashpb.GetIntegrationDashboardRequest,
) (*integrationdashpb.GetIntegrationDashboardResponse, error) {
	if uc == nil || uc.entity == nil {
		// Defensive — Wave B agents that consume the registry should
		// always receive a non-nil wrapper from the factory, but a
		// degraded empty response is preferable to a panic.
		return &integrationdashpb.GetIntegrationDashboardResponse{
			Success: true,
			Stats:   &integrationdashpb.IntegrationStats{},
		}, nil
	}

	if req == nil {
		req = &integrationdashpb.GetIntegrationDashboardRequest{}
	}

	now := time.Time{}
	if req.NowMillis != nil && *req.NowMillis != 0 {
		now = time.UnixMilli(*req.NowMillis)
	}

	entityResp, err := uc.entity.Execute(ctx, &integrationdashboard.GetIntegrationDashboardPageDataRequest{
		WorkspaceID: req.GetWorkspaceId(),
		Now:         now,
	})
	if err != nil {
		return nil, err
	}
	if entityResp == nil {
		return &integrationdashpb.GetIntegrationDashboardResponse{
			Success: true,
			Stats:   &integrationdashpb.IntegrationStats{},
		}, nil
	}

	// Project Go-shaped trend buckets into parallel label/value series.
	// Empty input → empty output; the view layer handles the empty-state
	// default (flat-zero 7-day trend).
	labels := make([]string, 0, len(entityResp.TrendBuckets))
	values := make([]float64, 0, len(entityResp.TrendBuckets))
	for _, b := range entityResp.TrendBuckets {
		labels = append(labels, b.Period.Format("Mon"))
		values = append(values, float64(b.Value))
	}

	providers := make([]*integrationdashpb.IntegrationProviderRow, 0, len(entityResp.Providers))
	for _, p := range entityResp.Providers {
		// Per codex review P2 2026-05-20: zero time.Time → negative Unix millis;
		// guard so unset LastSync emits 0 instead of leaking the zero-time epoch.
		var lastSync int64
		if !p.LastSync.IsZero() {
			lastSync = p.LastSync.UnixMilli()
		}
		providers = append(providers, &integrationdashpb.IntegrationProviderRow{
			Id:             p.ID,
			Name:           p.Name,
			Status:         p.Status,
			LastSyncMillis: lastSync,
			EventsLast_7D:  p.EventsLast7d,
		})
	}

	errors := make([]*integrationdashpb.IntegrationErrorEntry, 0, len(entityResp.RecentErrors))
	for _, e := range entityResp.RecentErrors {
		// Same zero-time guard as Providers; OccurredAt may be unset for
		// queued-but-not-yet-emitted errors.
		var occurredAt int64
		if !e.OccurredAt.IsZero() {
			occurredAt = e.OccurredAt.UnixMilli()
		}
		errors = append(errors, &integrationdashpb.IntegrationErrorEntry{
			Id:               e.ID,
			Provider:         e.Provider,
			Message:          e.Message,
			OccurredAtMillis: occurredAt,
		})
	}

	return &integrationdashpb.GetIntegrationDashboardResponse{
		Success: true,
		Stats: &integrationdashpb.IntegrationStats{
			TotalIntegrations:  entityResp.TotalIntegrations,
			ActiveIntegrations: entityResp.ActiveIntegrations,
			ErrorsLast_24H:     entityResp.ErrorsLast24h,
			Disconnected:       entityResp.Disconnected,
		},
		TrendLabels:  labels,
		TrendValues:  values,
		Providers:    providers,
		RecentErrors: errors,
	}, nil
}
