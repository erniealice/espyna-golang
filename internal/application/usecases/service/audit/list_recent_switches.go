// Package audit hosts the service-driven Audit use cases.
//
// ListRecentSwitches returns the most recent workspace-switch audit entries
// for a given actor (user). This replaces the raw SQL query that previously
// lived in apps/service-admin/internal/composition/options.go.
package audit

import (
	"context"
	"errors"
	"log"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
)

// ListRecentSwitchesRequest is the input for the ListRecentSwitches use case.
type ListRecentSwitchesRequest struct {
	UserID string
	Limit  int
}

// ListRecentSwitchesEntry is one row in the recent-switches result.
type ListRecentSwitchesEntry struct {
	OccurredAt string // RFC3339 timestamp
	UseCase    string // e.g. switch_url_rotate, switch_explicit
	RequestURL string
	Referer    string
}

// ListRecentSwitchesResponse carries the result list.
type ListRecentSwitchesResponse struct {
	Entries []ListRecentSwitchesEntry
}

// ListRecentSwitchesUseCase queries audit_entry for the most recent
// workspace-switch events by a given user.
type ListRecentSwitchesUseCase struct {
	auditService infraports.AuditService
}

// NewListRecentSwitchesUseCase wires the use case.
func NewListRecentSwitchesUseCase(
	auditService infraports.AuditService,
) *ListRecentSwitchesUseCase {
	return &ListRecentSwitchesUseCase{auditService: auditService}
}

// Execute returns the most recent N workspace-switch audit entries for the
// given user. Degrades gracefully: returns empty on nil audit service,
// nil request, or query errors (so the page still renders).
func (uc *ListRecentSwitchesUseCase) Execute(
	ctx context.Context,
	req *ListRecentSwitchesRequest,
) (*ListRecentSwitchesResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if uc.auditService == nil {
		return &ListRecentSwitchesResponse{}, nil
	}

	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	resp, err := uc.auditService.ListByActor(ctx, &infraports.ListByActorRequest{
		ActorID:       req.UserID,
		UseCasePrefix: "switch_",
		Limit:         limit,
	})
	if err != nil {
		// Audit_entry table may not exist (test seeds without it) or
		// request_url/referer columns may be absent in older schemas;
		// return empty rather than erroring so the page renders.
		log.Printf("[me/recent-activity] ListByActor error: %v", err)
		return &ListRecentSwitchesResponse{}, nil
	}
	if resp == nil {
		return &ListRecentSwitchesResponse{}, nil
	}

	entries := make([]ListRecentSwitchesEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entries[i] = ListRecentSwitchesEntry{
			OccurredAt: e.OccurredAt,
			UseCase:    e.UseCase,
			RequestURL: e.RequestURL,
			Referer:    e.Referer,
		}
	}
	return &ListRecentSwitchesResponse{Entries: entries}, nil
}
