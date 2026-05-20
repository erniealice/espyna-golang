package schedule

import (
	"context"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/event/dashboard"
	scheduledashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/schedule"
)

// GetScheduleDashboardRepositories groups the infrastructure dependencies
// of the wrapper. EntityDashboard is the entity-layer use case that owns
// the actual query algorithm (CountToday / CountThisWeek / UpcomingByStartDate
// / CountByDay / CountByTag), which in turn delegates to a postgres-backed
// repo via the EventDashboardRepository port. It may be nil under
// non-postgres builds (mock_db, mock_auth) — the wrapper degrades
// gracefully to an empty Response in that case.
type GetScheduleDashboardRepositories struct {
	EntityDashboard *eventdashboard.GetScheduleDashboardPageDataUseCase
}

// GetScheduleDashboardServices groups application services. No
// TransactionService — the use case is read-only.
//
// AuthorizationService is wired through for parity with Audit's pattern,
// but the schedule dashboard does NOT call authcheck.Check today (see
// Execute's doc comment). TranslationService is reserved for future
// error-message translations.
type GetScheduleDashboardServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetScheduleDashboardUseCase is the proto-shaped wrapper over the entity-
// layer `eventdashboard.GetScheduleDashboardPageDataUseCase`.
//
// It translates the proto Request defined in
// `proto/v1/service/dashboard/schedule/dashboard.proto` into the Go-shaped
// Request the entity-layer use case carries, then projects the Go-shaped
// Response back into the proto Response. The wrapper does NOT re-implement
// the aggregation algorithm — it delegates to the entity-layer use case to
// preserve the nil-safe degraded contract documented at
// usecases/event/dashboard/get_page_data.go.
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated
// by the upstream HTTP view middleware (the dashboard URL resolves through
// cyta's RegisterRoutes only when the session is authenticated). This
// matches the pattern adopted by the Integration Dashboard at
// `service/dashboard/integration/get_dashboard.go`. Future Wave B
// candidates that surface sensitive cross-workspace data via the dashboard
// MAY revisit this omission per Q-PERMQ-AUTHCHECK reasoning.
type GetScheduleDashboardUseCase struct {
	repositories GetScheduleDashboardRepositories
	services     GetScheduleDashboardServices
}

// NewGetScheduleDashboardUseCase wires the wrapper from grouped dependencies.
func NewGetScheduleDashboardUseCase(
	repositories GetScheduleDashboardRepositories,
	services GetScheduleDashboardServices,
) *GetScheduleDashboardUseCase {
	return &GetScheduleDashboardUseCase{repositories: repositories, services: services}
}

// Execute runs the entity-layer schedule-dashboard aggregation with proto-
// shaped IO.
//
// The proto NowMillis is unix milliseconds; zero or unset falls back to
// server time (matches the entity-layer zero-value time.Time semantics
// where the use case substitutes time.Now()). All failure modes degrade
// gracefully — the entity-layer use case never returns an error in the
// current shape, but the wrapper is defensive in case future repo
// implementations introduce error paths.
//
// When the wrapper or its entity-layer dependency is nil (mock_db build,
// non-postgres provider), the response carries a zero-valued Stats envelope
// and empty repeated fields; the view layer renders empty state.
func (uc *GetScheduleDashboardUseCase) Execute(
	ctx context.Context,
	req *scheduledashpb.GetScheduleDashboardRequest,
) (*scheduledashpb.GetScheduleDashboardResponse, error) {
	if uc == nil || uc.repositories.EntityDashboard == nil {
		return &scheduledashpb.GetScheduleDashboardResponse{
			Success: true,
			Stats:   &scheduledashpb.ScheduleStats{},
		}, nil
	}

	if req == nil {
		req = &scheduledashpb.GetScheduleDashboardRequest{}
	}

	now := time.Time{}
	if req.NowMillis != nil && *req.NowMillis != 0 {
		now = time.UnixMilli(*req.NowMillis)
	}

	entityResp, err := uc.repositories.EntityDashboard.Execute(ctx, &eventdashboard.GetScheduleDashboardPageDataRequest{
		WorkspaceID: req.GetWorkspaceId(),
		Now:         now,
	})
	if err != nil {
		return nil, err
	}
	if entityResp == nil {
		return &scheduledashpb.GetScheduleDashboardResponse{
			Success: true,
			Stats:   &scheduledashpb.ScheduleStats{},
		}, nil
	}

	// Translate ByTag map[string]int64 → []*ScheduleTagSlice for stable
	// ordering across language clients (Q-SDM-DASHBOARD-SHARED-TYPES
	// rationale: proto3 maps are unordered).
	// Per codex review P1 2026-05-20: Go map iteration is nondeterministic;
	// future authors copying this pattern MUST sort keys before emitting to
	// keep proto output deterministic (cross-language clients depend on it).
	tagKeys := make([]string, 0, len(entityResp.ByTag))
	for tag := range entityResp.ByTag {
		tagKeys = append(tagKeys, tag)
	}
	sort.Strings(tagKeys)
	byTagSlices := make([]*scheduledashpb.ScheduleTagSlice, 0, len(tagKeys))
	for _, tag := range tagKeys {
		byTagSlices = append(byTagSlices, &scheduledashpb.ScheduleTagSlice{
			Tag:   tag,
			Count: entityResp.ByTag[tag],
		})
	}

	return &scheduledashpb.GetScheduleDashboardResponse{
		Success: true,
		Stats: &scheduledashpb.ScheduleStats{
			Today:          entityResp.Stats.Today,
			ThisWeek:       entityResp.Stats.ThisWeek,
			ByTag:          entityResp.Stats.ByTag,
			UtilizationPct: entityResp.Stats.UtilizationPct,
		},
		ByDayLabels: entityResp.ByDayLabels,
		ByDayValues: entityResp.ByDayValues,
		ByTag:       byTagSlices,
		Upcoming:    entityResp.Upcoming,
	}, nil
}
