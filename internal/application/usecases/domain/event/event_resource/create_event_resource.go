package eventresource

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"

	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventresourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_resource"
)

// CreateEventResourceRepositories groups all repository dependencies
type CreateEventResourceRepositories struct {
	EventResource eventresourcepb.EventResourceDomainServiceServer // Primary entity repository
	Event         eventpb.EventDomainServiceServer                 // Entity reference validation
}

// CreateEventResourceServices groups all business service dependencies
type CreateEventResourceServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateEventResourceUseCase handles the business logic for creating event resource assignments
type CreateEventResourceUseCase struct {
	repositories CreateEventResourceRepositories
	services     CreateEventResourceServices
}

// NewCreateEventResourceUseCase creates use case with grouped dependencies
func NewCreateEventResourceUseCase(
	repositories CreateEventResourceRepositories,
	services CreateEventResourceServices,
) *CreateEventResourceUseCase {
	return &CreateEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventResourceUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventResourceUseCase with grouped parameters instead
func NewCreateEventResourceUseCaseUngrouped(
	eventResourceRepo eventresourcepb.EventResourceDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
) *CreateEventResourceUseCase {
	repositories := CreateEventResourceRepositories{
		EventResource: eventResourceRepo,
		Event:         eventRepo,
	}

	services := CreateEventResourceServices{
		Authorizer:  nil, // Will be injected later if needed
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return &CreateEventResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event resource operation
func (uc *CreateEventResourceUseCase) Execute(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventResource,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichEventResourceData(req.Data); err != nil {
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
func (uc *CreateEventResourceUseCase) shouldUseTransaction(ctx context.Context) bool {
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
func (uc *CreateEventResourceUseCase) executeWithTransaction(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	var response *eventresourcepb.CreateEventResourceResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Business rule validation (check first to avoid unnecessary DB calls)
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			return err
		}

		// Create EventResource (will participate in transaction)
		createResponse, err := uc.repositories.EventResource.CreateEventResource(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event resource: %w", err)
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
func (uc *CreateEventResourceUseCase) executeWithoutTransaction(ctx context.Context, req *eventresourcepb.CreateEventResourceRequest) (*eventresourcepb.CreateEventResourceResponse, error) {
	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	// Call repository (no transaction)
	return uc.repositories.EventResource.CreateEventResource(ctx, req)
}

// validateInput validates the input request
func (uc *CreateEventResourceUseCase) validateInput(req *eventresourcepb.CreateEventResourceRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event resource data is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ResourceId == "" {
		return errors.New("resource ID is required")
	}
	return nil
}

// enrichEventResourceData adds generated fields and audit information
func (uc *CreateEventResourceUseCase) enrichEventResourceData(eventResource *eventresourcepb.EventResource) error {
	now := time.Now()

	// Generate EventResource ID if not provided
	if eventResource.Id == "" {
		eventResource.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	eventResource.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventResource.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventResource.DateModified = &[]int64{now.UnixMilli()}[0]
	eventResource.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	eventResource.Active = true

	// Set default status if unspecified
	if eventResource.Status == eventresourcepb.ResourceStatus_RESOURCE_STATUS_UNSPECIFIED {
		eventResource.Status = eventresourcepb.ResourceStatus_RESOURCE_STATUS_ASSIGNED
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateEventResourceUseCase) validateBusinessRules(eventResource *eventresourcepb.EventResource) error {
	// Validate event and resource IDs are not the same
	if eventResource.EventId == eventResource.ResourceId {
		return errors.New("event ID and resource ID cannot be the same")
	}

	// Additional business rules can be added here
	// - Check resource type is valid for the event
	// - Check for scheduling conflicts

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateEventResourceUseCase) validateEntityReferences(ctx context.Context, eventResource *eventresourcepb.EventResource) error {
	// Validate Event entity reference
	if eventResource.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventResource.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventResource.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventResource.EventId)
		}
	}

	return nil
}
