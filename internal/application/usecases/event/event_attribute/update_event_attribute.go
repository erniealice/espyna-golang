package event_attribute

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

// UpdateEventAttributeUseCase handles the business logic for updating event attributes
// UpdateEventAttributeRepositories groups all repository dependencies
type UpdateEventAttributeRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
	Event          eventpb.EventDomainServiceServer
	Attribute      attributepb.AttributeDomainServiceServer
}

// UpdateEventAttributeServices groups all business service dependencies
type UpdateEventAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateEventAttributeUseCase handles the business logic for updating event attributes
type UpdateEventAttributeUseCase struct {
	repositories UpdateEventAttributeRepositories
	services     UpdateEventAttributeServices
}

// NewUpdateEventAttributeUseCase creates a new UpdateEventAttributeUseCase
func NewUpdateEventAttributeUseCase(
	repositories UpdateEventAttributeRepositories,
	services UpdateEventAttributeServices,
) *UpdateEventAttributeUseCase {
	return &UpdateEventAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event attribute operation
func (uc *UpdateEventAttributeUseCase) Execute(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) (*eventattributepb.UpdateEventAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event attribute update within a transaction
func (uc *UpdateEventAttributeUseCase) executeWithTransaction(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) (*eventattributepb.UpdateEventAttributeResponse, error) {
	var result *eventattributepb.UpdateEventAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
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
func (uc *UpdateEventAttributeUseCase) executeCore(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) (*eventattributepb.UpdateEventAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttribute, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichEventAttributeData(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.EventAttribute.UpdateEventAttribute(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateEventAttributeUseCase) validateInput(ctx context.Context, req *eventattributepb.UpdateEventAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.data_required", "Event attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.id_required", "Event attribute ID is required"))
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
func (uc *UpdateEventAttributeUseCase) enrichEventAttributeData(eventAttribute *eventattributepb.EventAttribute) error {
	now := time.Now()

	// Update audit fields
	eventAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	eventAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for event attributes
func (uc *UpdateEventAttributeUseCase) validateBusinessRules(ctx context.Context, eventAttribute *eventattributepb.EventAttribute) error {
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
func (uc *UpdateEventAttributeUseCase) validateEntityReferences(ctx context.Context, eventAttribute *eventattributepb.EventAttribute) error {
	// Validate Event entity reference
	if eventAttribute.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventAttribute.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_attribute.errors.event_not_found", map[string]interface{}{"eventId": eventAttribute.EventId}, "Referenced event not found")
			return errors.New(translatedError)
		}
		if !event.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_attribute.errors.event_not_active", map[string]interface{}{"eventId": eventAttribute.EventId}, "Referenced event not active")
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if eventAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: eventAttribute.AttributeId},
		})
		if err != nil {
			return err
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_attribute.errors.attribute_not_found", map[string]interface{}{"attributeId": eventAttribute.AttributeId}, "Referenced attribute not found")
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_attribute.errors.attribute_not_active", map[string]interface{}{"attributeId": eventAttribute.AttributeId}, "Referenced attribute not active")
			return errors.New(translatedError)
		}
	}

	return nil
}
