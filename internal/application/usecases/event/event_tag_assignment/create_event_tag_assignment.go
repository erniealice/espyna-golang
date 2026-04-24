package eventtagassignment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
	eventtagassignmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag_assignment"
)

// CreateEventTagAssignmentRepositories groups all repository dependencies
type CreateEventTagAssignmentRepositories struct {
	EventTagAssignment eventtagassignmentpb.EventTagAssignmentDomainServiceServer // Primary entity repository
	Event              eventpb.EventDomainServiceServer                           // Entity reference validation
	EventTag           eventtagpb.EventTagDomainServiceServer                     // Entity reference validation
}

// CreateEventTagAssignmentServices groups all business service dependencies
type CreateEventTagAssignmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEventTagAssignmentUseCase handles the business logic for creating event_tag_assignment associations
type CreateEventTagAssignmentUseCase struct {
	repositories CreateEventTagAssignmentRepositories
	services     CreateEventTagAssignmentServices
}

// NewCreateEventTagAssignmentUseCase creates use case with grouped dependencies
func NewCreateEventTagAssignmentUseCase(
	repositories CreateEventTagAssignmentRepositories,
	services CreateEventTagAssignmentServices,
) *CreateEventTagAssignmentUseCase {
	return &CreateEventTagAssignmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event_tag_assignment operation
func (uc *CreateEventTagAssignmentUseCase) Execute(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventTagAssignment, ports.ActionCreate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	if err := uc.enrichAssignmentData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeWithoutTransaction(ctx, req)
}

func (uc *CreateEventTagAssignmentUseCase) shouldUseTransaction(ctx context.Context) bool {
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}
	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}
	return true
}

func (uc *CreateEventTagAssignmentUseCase) executeWithTransaction(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	var response *eventtagassignmentpb.CreateEventTagAssignmentResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			return err
		}

		createResponse, err := uc.repositories.EventTagAssignment.CreateEventTagAssignment(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event_tag_assignment: %w", err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

func (uc *CreateEventTagAssignmentUseCase) executeWithoutTransaction(ctx context.Context, req *eventtagassignmentpb.CreateEventTagAssignmentRequest) (*eventtagassignmentpb.CreateEventTagAssignmentResponse, error) {
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	return uc.repositories.EventTagAssignment.CreateEventTagAssignment(ctx, req)
}

func (uc *CreateEventTagAssignmentUseCase) validateInput(req *eventtagassignmentpb.CreateEventTagAssignmentRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event_tag_assignment data is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event_id is required")
	}
	if req.Data.EventTagId == "" {
		return errors.New("event_tag_id is required")
	}
	return nil
}

func (uc *CreateEventTagAssignmentUseCase) enrichAssignmentData(assignment *eventtagassignmentpb.EventTagAssignment) error {
	now := time.Now()

	if assignment.Id == "" {
		assignment.Id = uc.services.IDService.GenerateID()
	}

	assignment.DateCreated = &[]int64{now.UnixMilli()}[0]
	assignment.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	assignment.DateModified = &[]int64{now.UnixMilli()}[0]
	assignment.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	assignment.Active = true

	return nil
}

func (uc *CreateEventTagAssignmentUseCase) validateBusinessRules(assignment *eventtagassignmentpb.EventTagAssignment) error {
	if assignment.WorkspaceId == "" {
		return errors.New("workspace_id is required for event_tag_assignment")
	}
	return nil
}

func (uc *CreateEventTagAssignmentUseCase) validateEntityReferences(ctx context.Context, assignment *eventtagassignmentpb.EventTagAssignment) error {
	if assignment.EventId != "" && uc.repositories.Event != nil {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: assignment.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", assignment.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", assignment.EventId)
		}
	}

	if assignment.EventTagId != "" && uc.repositories.EventTag != nil {
		tag, err := uc.repositories.EventTag.ReadEventTag(ctx, &eventtagpb.ReadEventTagRequest{
			Data: &eventtagpb.EventTag{Id: assignment.EventTagId},
		})
		if err != nil {
			return err
		}
		if tag == nil || tag.Data == nil || len(tag.Data) == 0 {
			return fmt.Errorf("referenced event_tag with ID '%s' does not exist", assignment.EventTagId)
		}
		if !tag.Data[0].Active {
			return fmt.Errorf("referenced event_tag with ID '%s' is not active", assignment.EventTagId)
		}
	}

	return nil
}
