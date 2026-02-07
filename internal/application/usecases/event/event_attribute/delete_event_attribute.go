package event_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	eventattributepb "leapfor.xyz/esqyma/golang/v1/domain/event/event_attribute"
)

// DeleteEventAttributeUseCase handles the business logic for deleting event attributes
// DeleteEventAttributeRepositories groups all repository dependencies
type DeleteEventAttributeRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
}

// DeleteEventAttributeServices groups all business service dependencies
type DeleteEventAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteEventAttributeUseCase handles the business logic for deleting event attributes
type DeleteEventAttributeUseCase struct {
	repositories DeleteEventAttributeRepositories
	services     DeleteEventAttributeServices
}

// NewDeleteEventAttributeUseCase creates a new DeleteEventAttributeUseCase
func NewDeleteEventAttributeUseCase(
	repositories DeleteEventAttributeRepositories,
	services DeleteEventAttributeServices,
) *DeleteEventAttributeUseCase {
	return &DeleteEventAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event attribute operation
func (uc *DeleteEventAttributeUseCase) Execute(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) (*eventattributepb.DeleteEventAttributeResponse, error) {
	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event attribute deletion within a transaction
func (uc *DeleteEventAttributeUseCase) executeWithTransaction(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) (*eventattributepb.DeleteEventAttributeResponse, error) {
	var result *eventattributepb.DeleteEventAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting an event attribute
func (uc *DeleteEventAttributeUseCase) executeCore(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) (*eventattributepb.DeleteEventAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttribute, ports.ActionDelete)
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
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.EventAttribute.DeleteEventAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.deletion_failed", "Event attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteEventAttributeUseCase) validateInput(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.data_required", "Event attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.id_required", "Event attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for event attribute deletion
func (uc *DeleteEventAttributeUseCase) validateBusinessRules(ctx context.Context, req *eventattributepb.DeleteEventAttributeRequest) error {
	// Additional business rule validation can be added here
	// For example: check if event attribute is referenced by other entities
	if uc.isEventAttributeInUse(ctx, req.Data.EventId, req.Data.AttributeId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.in_use", "Event attribute is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isEventAttributeInUse checks if the event attribute is referenced by other entities
func (uc *DeleteEventAttributeUseCase) isEventAttributeInUse(ctx context.Context, eventID, attributeID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for event attribute usage
	return false
}
