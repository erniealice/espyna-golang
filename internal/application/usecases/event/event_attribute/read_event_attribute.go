package event_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

// ReadEventAttributeUseCase handles the business logic for reading an event attribute
// ReadEventAttributeRepositories groups all repository dependencies
type ReadEventAttributeRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer // Primary entity repository
}

// ReadEventAttributeServices groups all business service dependencies
type ReadEventAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadEventAttributeUseCase handles the business logic for reading an event attribute
type ReadEventAttributeUseCase struct {
	repositories ReadEventAttributeRepositories
	services     ReadEventAttributeServices
}

// NewReadEventAttributeUseCase creates a new ReadEventAttributeUseCase
func NewReadEventAttributeUseCase(
	repositories ReadEventAttributeRepositories,
	services ReadEventAttributeServices,
) *ReadEventAttributeUseCase {
	return &ReadEventAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event attribute operation
func (uc *ReadEventAttributeUseCase) Execute(ctx context.Context, req *eventattributepb.ReadEventAttributeRequest) (*eventattributepb.ReadEventAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.authorization_failed", "Authorization failed for event attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventAttribute, ports.ActionRead)
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

	// Call repository
	resp, err := uc.repositories.EventAttribute.ReadEventAttribute(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "event_attribute.errors.not_found", map[string]interface{}{"eventAttributeId": req.Data.Id}, "Event attribute not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.errors.read_failed", "Failed to read event attribute")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadEventAttributeUseCase) validateInput(ctx context.Context, req *eventattributepb.ReadEventAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.data_required", "event attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_attribute.validation.id_required", "event attribute ID is required"))
	}
	return nil
}
