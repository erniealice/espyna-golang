package eventattendee

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattendeepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attendee"
)

// CreateEventAttendeeRepositories groups all repository dependencies
type CreateEventAttendeeRepositories struct {
	EventAttendee eventattendeepb.EventAttendeeDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// CreateEventAttendeeServices groups all business service dependencies
type CreateEventAttendeeServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateEventAttendeeUseCase handles the business logic for creating event attendee associations
type CreateEventAttendeeUseCase struct {
	repositories CreateEventAttendeeRepositories
	services     CreateEventAttendeeServices
}

// NewCreateEventAttendeeUseCase creates use case with grouped dependencies
func NewCreateEventAttendeeUseCase(
	repositories CreateEventAttendeeRepositories,
	services CreateEventAttendeeServices,
) *CreateEventAttendeeUseCase {
	return &CreateEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventAttendeeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventAttendeeUseCase with grouped parameters instead
func NewCreateEventAttendeeUseCaseUngrouped(
	eventAttendeeRepo eventattendeepb.EventAttendeeDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
) *CreateEventAttendeeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateEventAttendeeRepositories{
		EventAttendee: eventAttendeeRepo,
		Event:         eventRepo,
	}

	services := CreateEventAttendeeServices{
		Authorizer:  nil, // Will be injected later if needed
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return &CreateEventAttendeeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event attendee operation
func (uc *CreateEventAttendeeUseCase) Execute(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventAttendee,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichEventAttendeeData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *CreateEventAttendeeUseCase) shouldUseTransaction(ctx context.Context) bool {
	// Use transaction if:
	// 1. Transactor is available, AND
	// 2. We're not already in a transaction context
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return false
	}

	// Don't start a nested transaction if we're already in one
	if uc.services.Transactor.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreateEventAttendeeUseCase) executeWithTransaction(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	var response *eventattendeepb.CreateEventAttendeeResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// All validations and operations within transaction

		// Business rule validation (check first to avoid unnecessary DB calls)
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			return err
		}

		// Create EventAttendee (will participate in transaction)
		createResponse, err := uc.repositories.EventAttendee.CreateEventAttendee(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event attendee: %w", err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction (backward compatibility)
func (uc *CreateEventAttendeeUseCase) executeWithoutTransaction(ctx context.Context, req *eventattendeepb.CreateEventAttendeeRequest) (*eventattendeepb.CreateEventAttendeeResponse, error) {
	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	// Call repository (no transaction)
	return uc.repositories.EventAttendee.CreateEventAttendee(ctx, req)
}

// validateInput validates the input request
func (uc *CreateEventAttendeeUseCase) validateInput(req *eventattendeepb.CreateEventAttendeeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event attendee data is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	return nil
}

// enrichEventAttendeeData adds generated fields and audit information
func (uc *CreateEventAttendeeUseCase) enrichEventAttendeeData(eventAttendee *eventattendeepb.EventAttendee) error {
	now := time.Now()

	// Generate EventAttendee ID if not provided
	if eventAttendee.Id == "" {
		eventAttendee.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	eventAttendee.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventAttendee.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventAttendee.DateModified = &[]int64{now.UnixMilli()}[0]
	eventAttendee.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	eventAttendee.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateEventAttendeeUseCase) validateBusinessRules(eventAttendee *eventattendeepb.EventAttendee) error {
	// At least one attendee identity must be provided
	hasClient := eventAttendee.ClientId != nil && *eventAttendee.ClientId != ""
	hasWorkspaceUser := eventAttendee.WorkspaceUserId != nil && *eventAttendee.WorkspaceUserId != ""

	if !hasClient && !hasWorkspaceUser && (eventAttendee.DisplayName == nil || *eventAttendee.DisplayName == "") {
		return errors.New("attendee must have a client_id, workspace_user_id, or display_name")
	}

	// Additional business rules can be added here
	// - Check event capacity
	// - Check for duplicate attendees (unique_together: event_id,client_id)
	// - Validate scheduling conflicts

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateEventAttendeeUseCase) validateEntityReferences(ctx context.Context, eventAttendee *eventattendeepb.EventAttendee) error {
	// Validate Event entity reference
	if eventAttendee.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventAttendee.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventAttendee.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventAttendee.EventId)
		}
	}

	return nil
}
