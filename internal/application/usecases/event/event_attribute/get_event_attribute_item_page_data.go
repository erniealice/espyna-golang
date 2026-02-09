package event_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	eventattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_attribute"
)

type GetEventAttributeItemPageDataRepositories struct {
	EventAttribute eventattributepb.EventAttributeDomainServiceServer
}

type GetEventAttributeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetEventAttributeItemPageDataUseCase handles the business logic for getting event attribute item page data
type GetEventAttributeItemPageDataUseCase struct {
	repositories GetEventAttributeItemPageDataRepositories
	services     GetEventAttributeItemPageDataServices
}

// NewGetEventAttributeItemPageDataUseCase creates a new GetEventAttributeItemPageDataUseCase
func NewGetEventAttributeItemPageDataUseCase(
	repositories GetEventAttributeItemPageDataRepositories,
	services GetEventAttributeItemPageDataServices,
) *GetEventAttributeItemPageDataUseCase {
	return &GetEventAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get event attribute item page data operation
func (uc *GetEventAttributeItemPageDataUseCase) Execute(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeItemPageDataRequest,
) (*eventattributepb.GetEventAttributeItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.EventAttributeId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes event attribute item page data retrieval within a transaction
func (uc *GetEventAttributeItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeItemPageDataRequest,
) (*eventattributepb.GetEventAttributeItemPageDataResponse, error) {
	var result *eventattributepb.GetEventAttributeItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"event_attribute.errors.item_page_data_failed",
				"event attribute item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting event attribute item page data
func (uc *GetEventAttributeItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeItemPageDataRequest,
) (*eventattributepb.GetEventAttributeItemPageDataResponse, error) {
	// Create read request for the event attribute
	readReq := &eventattributepb.ReadEventAttributeRequest{
		Data: &eventattributepb.EventAttribute{
			Id: req.EventAttributeId,
		},
	}

	// Retrieve the event attribute
	readResp, err := uc.repositories.EventAttribute.ReadEventAttribute(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.errors.read_failed",
			"failed to retrieve event attribute: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.errors.not_found",
			"event attribute not found",
		))
	}

	// Get the event attribute (should be only one)
	eventAttribute := readResp.Data[0]

	// Validate that we got the expected event attribute
	if eventAttribute.Id != req.EventAttributeId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.errors.id_mismatch",
			"retrieved event attribute ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (event details, attribute details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the event attribute as-is
	return &eventattributepb.GetEventAttributeItemPageDataResponse{
		EventAttribute: eventAttribute,
		Success:        true,
	}, nil
}

// validateInput validates the input request
func (uc *GetEventAttributeItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *eventattributepb.GetEventAttributeItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.validation.request_required",
			"Request is required for event attributes [DEFAULT]",
		))
	}

	// Validate event attribute ID - uses direct field NOT nested Data
	if strings.TrimSpace(req.EventAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.validation.id_required",
			"Event attribute ID is required [DEFAULT]",
		))
	}

	// Basic ID format validation
	if len(req.EventAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.validation.id_too_short",
			"Event attribute ID must be at least 3 characters [DEFAULT]",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading event attribute item page data
func (uc *GetEventAttributeItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	eventAttributeId string,
) error {
	// Validate event attribute ID format
	if len(eventAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"event_attribute.validation.id_too_short",
			"event attribute ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this event attribute
	// - Validate event attribute belongs to the current user's organization
	// - Check if event attribute is in a state that allows viewing
	// - Rate limiting for event attribute access
	// - Audit logging requirements

	return nil
}
