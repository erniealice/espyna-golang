package eventtagassignment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// SetEventTagAssignmentsRepositories groups all repository dependencies
type SetEventTagAssignmentsRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer
	Event              eventpb.EventDomainServiceServer
	EventTag           eventtagpb.EventTagDomainServiceServer
}

// SetEventTagAssignmentsServices groups all business service dependencies
type SetEventTagAssignmentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// SetEventTagAssignmentsRequest is the input for the atomic replace operation.
// Given an event_id and an ordered list of tag_ids, this use case deletes the
// existing active assignments for the event and inserts fresh rows with
// positions 0..N-1 matching the input order.
type SetEventTagAssignmentsRequest struct {
	EventID     string
	WorkspaceID string
	TagIDs      []string
}

// SetEventTagAssignmentsResponse returns the newly persisted assignments.
type SetEventTagAssignmentsResponse struct {
	Assignments []*eventtagassignmentpb.EventTagAssignment
	Success     bool
}

// SetEventTagAssignmentsUseCase implements the atomic replace (delete-then-create)
// for the per-event tag picker.
type SetEventTagAssignmentsUseCase struct {
	repositories SetEventTagAssignmentsRepositories
	services     SetEventTagAssignmentsServices
}

// NewSetEventTagAssignmentsUseCase creates use case with grouped dependencies
func NewSetEventTagAssignmentsUseCase(
	repositories SetEventTagAssignmentsRepositories,
	services SetEventTagAssignmentsServices,
) *SetEventTagAssignmentsUseCase {
	return &SetEventTagAssignmentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the atomic replace: delete existing assignments for the
// event, then insert new rows with positions 0..N-1.
func (uc *SetEventTagAssignmentsUseCase) Execute(ctx context.Context, req *SetEventTagAssignmentsRequest) (*SetEventTagAssignmentsResponse, error) {
	// Authorization: this is effectively a create operation on assignments.
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTagAssignment, ports.ActionCreate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeWithoutTransaction(ctx, req)
}

func (uc *SetEventTagAssignmentsUseCase) shouldUseTransaction(ctx context.Context) bool {
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}
	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}
	return true
}

func (uc *SetEventTagAssignmentsUseCase) executeWithTransaction(ctx context.Context, req *SetEventTagAssignmentsRequest) (*SetEventTagAssignmentsResponse, error) {
	var response *SetEventTagAssignmentsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		resp, coreErr := uc.executeCore(txCtx, req)
		if coreErr != nil {
			return coreErr
		}
		response = resp
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}
	return response, nil
}

func (uc *SetEventTagAssignmentsUseCase) executeWithoutTransaction(ctx context.Context, req *SetEventTagAssignmentsRequest) (*SetEventTagAssignmentsResponse, error) {
	return uc.executeCore(ctx, req)
}

func (uc *SetEventTagAssignmentsUseCase) executeCore(ctx context.Context, req *SetEventTagAssignmentsRequest) (*SetEventTagAssignmentsResponse, error) {
	// 1. Validate referenced entities exist.
	if err := uc.validateEntityReferences(ctx, req); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	// 2. Find existing assignments for the event.
	listReq := &eventtagassignmentpb.ListEventTagAssignmentsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "event_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    req.EventID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}
	existing, err := uc.repositories.EventTagAssignment.ListEventTagAssignments(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing assignments: %w", err)
	}

	// 3. Delete the existing rows (soft delete via repository).
	if existing != nil {
		for _, row := range existing.Data {
			if row == nil || !row.Active {
				continue
			}
			delReq := &eventtagassignmentpb.DeleteEventTagAssignmentRequest{
				Data: &eventtagassignmentpb.EventTagAssignment{Id: row.Id},
			}
			if _, delErr := uc.repositories.EventTagAssignment.DeleteEventTagAssignment(ctx, delReq); delErr != nil {
				return nil, fmt.Errorf("failed to delete existing assignment %s: %w", row.Id, delErr)
			}
		}
	}

	// 4. Create fresh rows with positions 0..N-1.
	now := time.Now()
	created := make([]*eventtagassignmentpb.EventTagAssignment, 0, len(req.TagIDs))
	for i, tagID := range req.TagIDs {
		if tagID == "" {
			continue
		}
		row := &eventtagassignmentpb.EventTagAssignment{
			EventId:            req.EventID,
			EventTagId:         tagID,
			Position:           int32(i),
			WorkspaceId:        req.WorkspaceID,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
			Active:             true,
		}
		if uc.services.IDService != nil {
			row.Id = uc.services.IDService.GenerateID()
		}

		createReq := &eventtagassignmentpb.CreateEventTagAssignmentRequest{Data: row}
		resp, createErr := uc.repositories.EventTagAssignment.CreateEventTagAssignment(ctx, createReq)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create assignment for tag %s: %w", tagID, createErr)
		}
		if resp != nil && len(resp.Data) > 0 {
			created = append(created, resp.Data[0])
		}
	}

	return &SetEventTagAssignmentsResponse{
		Assignments: created,
		Success:     true,
	}, nil
}

func (uc *SetEventTagAssignmentsUseCase) validateInput(req *SetEventTagAssignmentsRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.EventID == "" {
		return errors.New("event_id is required")
	}
	if req.WorkspaceID == "" {
		return errors.New("workspace_id is required")
	}
	// Empty TagIDs is a valid input — it means "clear all tags from this event".
	return nil
}

func (uc *SetEventTagAssignmentsUseCase) validateEntityReferences(ctx context.Context, req *SetEventTagAssignmentsRequest) error {
	if uc.repositories.Event != nil {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: req.EventID},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", req.EventID)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", req.EventID)
		}
	}

	if uc.repositories.EventTag != nil {
		for _, tagID := range req.TagIDs {
			if tagID == "" {
				continue
			}
			tag, err := uc.repositories.EventTag.ReadEventTag(ctx, &eventtagpb.ReadEventTagRequest{
				Data: &eventtagpb.EventTag{Id: tagID},
			})
			if err != nil {
				return err
			}
			if tag == nil || tag.Data == nil || len(tag.Data) == 0 {
				return fmt.Errorf("referenced event_tag with ID '%s' does not exist", tagID)
			}
			if !tag.Data[0].Active {
				return fmt.Errorf("referenced event_tag with ID '%s' is not active", tagID)
			}
		}
	}

	return nil
}
