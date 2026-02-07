package location_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// UpdateLocationAttributeRepositories groups all repository dependencies
type UpdateLocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
	Location          locationpb.LocationDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// UpdateLocationAttributeServices groups all business service dependencies
type UpdateLocationAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateLocationAttributeUseCase handles the business logic for updating location attributes
type UpdateLocationAttributeUseCase struct {
	repositories UpdateLocationAttributeRepositories
	services     UpdateLocationAttributeServices
}

// NewUpdateLocationAttributeUseCase creates use case with grouped dependencies
func NewUpdateLocationAttributeUseCase(
	repositories UpdateLocationAttributeRepositories,
	services UpdateLocationAttributeServices,
) *UpdateLocationAttributeUseCase {
	return &UpdateLocationAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateLocationAttributeUseCase with grouped parameters instead
func NewUpdateLocationAttributeUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
	locationRepo locationpb.LocationDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateLocationAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateLocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
		Location:          locationRepo,
		Attribute:         attributeRepo,
	}

	services := UpdateLocationAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateLocationAttributeUseCase(repositories, services)
}

func (uc *UpdateLocationAttributeUseCase) Execute(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateLocationAttributeUseCase) validateInput(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.request_required", "Request is required for location attributes"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.data_required", "Location attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.id_required", "Location attribute ID is required"))
	}
	if req.Data.LocationId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.location_id_required", "Location ID is required"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.attribute_id_required", "Attribute ID is required"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_required", "Value is required"))
	}
	return nil
}

// enrichLocationAttributeData adds updated audit information
func (uc *UpdateLocationAttributeUseCase) enrichLocationAttributeData(locationAttribute *locationattributepb.LocationAttribute) error {
	now := time.Now()

	// Update modification timestamp
	locationAttribute.DateModified = &[]int64{now.Unix()}[0]
	locationAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateLocationAttributeUseCase) validateBusinessRules(ctx context.Context, locationAttribute *locationattributepb.LocationAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(locationAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_empty", "Value cannot be empty"))
	}

	if len(locationAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_too_long", "Value cannot exceed 1000 characters"))
	}

	// TODO: Additional business rules
	// Example: Validate location and attribute exist
	// Example: Validate attribute type constraints
	// Example: Check permissions for updating this attribute
	// For now, allow all updates

	return nil
}

// executeWithTransaction executes location attribute update within a transaction
func (uc *UpdateLocationAttributeUseCase) executeWithTransaction(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	var result *locationattributepb.UpdateLocationAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "location_attribute.errors.update_failed", "Location attribute update failed")
			return fmt.Errorf("%s: %w", translatedError, err)
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
func (uc *UpdateLocationAttributeUseCase) executeCore(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.input_validation_failed", "Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichLocationAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.enrichment_failed", "Business logic enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.reference_validation_failed", "Entity reference validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.UpdateLocationAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.update_failed", "Location attribute update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateLocationAttributeUseCase) validateEntityReferences(ctx context.Context, locationAttribute *locationattributepb.LocationAttribute) error {
	// Validate Location entity reference
	if locationAttribute.LocationId != "" {
		location, err := uc.repositories.Location.ReadLocation(ctx, &locationpb.ReadLocationRequest{
			Data: &locationpb.Location{Id: locationAttribute.LocationId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.location_reference_validation_failed", "Failed to validate location entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if location == nil || location.Data == nil || len(location.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.location_not_found", "Referenced location with ID '{locationId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{locationId}", locationAttribute.LocationId)
			return errors.New(translatedError)
		}
		if !location.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.location_not_active", "Referenced location with ID '{locationId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{locationId}", locationAttribute.LocationId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if locationAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: locationAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", locationAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", locationAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
