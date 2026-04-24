package eventtagassignment

import (
	"context"
	"errors"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// ListEventTagAssignmentsByEventUseCase wraps ListEventTagAssignments with a
// server-side filter on event_id. Intended for per-event UI flows where we need
// only the assignments that belong to a single event (e.g. the multi-picker
// drawer save flow in Phase 4).
type ListEventTagAssignmentsByEventUseCase struct {
	list *ListEventTagAssignmentsUseCase
}

// NewListEventTagAssignmentsByEventUseCase wires the helper on top of the main
// list use case so authorization, translation and filter behavior stay identical.
func NewListEventTagAssignmentsByEventUseCase(list *ListEventTagAssignmentsUseCase) *ListEventTagAssignmentsByEventUseCase {
	return &ListEventTagAssignmentsByEventUseCase{list: list}
}

// Execute lists active event_tag_assignment rows for the given event_id.
func (uc *ListEventTagAssignmentsByEventUseCase) Execute(ctx context.Context, eventID string) (*eventtagassignmentpb.ListEventTagAssignmentsResponse, error) {
	if uc.list == nil {
		return nil, errors.New("list use case dependency is not initialized")
	}
	if eventID == "" {
		return nil, errors.New("event_id is required")
	}

	filter := &commonpb.TypedFilter{
		Field: "event_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    eventID,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}

	req := &eventtagassignmentpb.ListEventTagAssignmentsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{filter},
		},
	}

	return uc.list.Execute(ctx, req)
}
