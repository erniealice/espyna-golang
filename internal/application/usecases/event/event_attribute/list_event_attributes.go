package event_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	eventattributepb "leapfor.xyz/esqyma/golang/v1/domain/event/event_attribute"
)

// ListEventAttributesUseCase handles the business logic for listing event attributes
// ListEventAttributesRepositories groups all repository dependencies
type ListEventAttributesRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
}

// ListEventAttributesServices groups all business service dependencies
type ListEventAttributesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListEventAttributesUseCase handles the business logic for listing event attributes
type ListEventAttributesUseCase struct {
	repositories ListEventAttributesRepositories
	services     ListEventAttributesServices
}

// NewListEventAttributesUseCase creates a new ListEventAttributesUseCase
func NewListEventAttributesUseCase(
	repositories ListEventAttributesRepositories,
	services ListEventAttributesServices,
) *ListEventAttributesUseCase {
	return &ListEventAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event attributes operation
func (uc *ListEventAttributesUseCase) Execute(ctx context.Context, req *eventattributepb.ListEventAttributesRequest) (*eventattributepb.ListEventAttributesResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttribute, ports.ActionList)
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

	// Call repository
	resp, err := uc.repositories.EventAttribute.ListEventAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.list_failed", "Failed to retrieve event attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListEventAttributesUseCase) validateInput(ctx context.Context, req *eventattributepb.ListEventAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
