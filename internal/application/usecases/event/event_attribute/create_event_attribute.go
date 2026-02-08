package event_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

// CreateEventAttributeUseCase handles the business logic for creating event attributes
// CreateEventAttributeRepositories groups all repository dependencies
type CreateEventAttributeRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
	Event          eventpb.EventDomainServiceServer
	Attribute      attributepb.AttributeDomainServiceServer
}

// CreateEventAttributeServices groups all business service dependencies
type CreateEventAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateEventAttributeUseCase handles the business logic for creating event attributes
type CreateEventAttributeUseCase struct {
	repositories CreateEventAttributeRepositories
	services     CreateEventAttributeServices
}

// NewCreateEventAttributeUseCase creates a new CreateEventAttributeUseCase
func NewCreateEventAttributeUseCase(
	repositories CreateEventAttributeRepositories,
	services CreateEventAttributeServices,
) *CreateEventAttributeUseCase {
	return &CreateEventAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event attribute operation
func (uc *CreateEventAttributeUseCase) Execute(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) (*eventattributepb.CreateEventAttributeResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event attribute creation within a transaction
func (uc *CreateEventAttributeUseCase) executeWithTransaction(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) (*eventattributepb.CreateEventAttributeResponse, error) {
	var result *eventattributepb.CreateEventAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "event_attribute.errors.creation_failed", "Event attribute creation failed [DEFAULT]"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateEventAttributeUseCase) executeCore(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) (*eventattributepb.CreateEventAttributeResponse, error) {
	// TODO: Re-enable workspace-scoped authorization check once WorkspaceId is available
	// userID, err := contextutil.RequireUserIDFromContext(ctx)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// permission := ports.EntityPermission(ports.EntityEventAttribute, ports.ActionCreate)
	// hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// if !hasPerm {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.validation_failed", "Input validation failed [DEFAULT]"), err)
	}

	// Business logic and enrichment
	if err := uc.enrichEventAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]"), err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]"), err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]"), err)
	}

	// Call repository
	resp, err := uc.repositories.EventAttribute.CreateEventAttribute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.creation_failed", "Event attribute creation failed [DEFAULT]"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreateEventAttributeUseCase) validateInput(ctx context.Context, req *eventattributepb.CreateEventAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.data_required", "Event attribute data is required [DEFAULT]"))
	}
	if req.Data.EventId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.event_id_required", "Event ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

// enrichEventAttributeData adds generated fields and audit information
func (uc *CreateEventAttributeUseCase) enrichEventAttributeData(eventAttribute *eventattributepb.EventAttribute) error {
	now := time.Now()

	// Generate EventAttribute ID if not provided
	if eventAttribute.Id == "" {
		if uc.services.IDService != nil {
			eventAttribute.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			eventAttribute.Id = fmt.Sprintf("event-attr-%d", now.UnixNano())
		}
	}

	// Set audit fields
	eventAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	eventAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for event attributes
func (uc *CreateEventAttributeUseCase) validateBusinessRules(ctx context.Context, eventAttribute *eventattributepb.EventAttribute) error {
	// Validate event ID format
	if len(eventAttribute.EventId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.event_id_min_length", "Event ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate attribute ID format
	if len(eventAttribute.AttributeId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.attribute_id_min_length", "Attribute ID must be at least 2 characters long [DEFAULT]"))
	}

	// Validate attribute value length
	value := strings.TrimSpace(eventAttribute.Value)
	if len(value) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.value_not_empty", "Attribute value must not be empty [DEFAULT]"))
	}

	if len(value) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.value_max_length", "Attribute value cannot exceed 500 characters [DEFAULT]"))
	}

	// Normalize value (trim spaces)
	eventAttribute.Value = strings.TrimSpace(eventAttribute.Value)

	// Business constraint: Event attribute must be associated with a valid event
	if eventAttribute.EventId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.event_association_required", "Event attribute must be associated with an event [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateEventAttributeUseCase) validateEntityReferences(ctx context.Context, eventAttribute *eventattributepb.EventAttribute) error {
	// Validate Event entity reference
	if eventAttribute.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventAttribute.EventId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.event_reference_validation_failed", "Failed to validate event entity reference [DEFAULT]"), err)
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.event_not_found", "Referenced event with ID '%s' does not exist [DEFAULT]"), eventAttribute.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.event_not_active", "Referenced event with ID '%s' is not active [DEFAULT]"), eventAttribute.EventId)
		}
	}

	// Validate Attribute entity reference
	if eventAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: eventAttribute.AttributeId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]"), err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.attribute_not_found", "Referenced attribute with ID '%s' does not exist [DEFAULT]"), eventAttribute.AttributeId)
		}
		if !attribute.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.attribute_not_active", "Referenced attribute with ID '%s' is not active [DEFAULT]"), eventAttribute.AttributeId)
		}
	}

	return nil
}
