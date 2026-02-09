package location_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// CreateLocationAttributeRepositories groups all repository dependencies
type CreateLocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
	Location          locationpb.LocationDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// CreateLocationAttributeServices groups all business service dependencies
type CreateLocationAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateLocationAttributeUseCase handles the business logic for creating location attributes
type CreateLocationAttributeUseCase struct {
	repositories CreateLocationAttributeRepositories
	services     CreateLocationAttributeServices
}

// NewCreateLocationAttributeUseCase creates use case with grouped dependencies
func NewCreateLocationAttributeUseCase(
	repositories CreateLocationAttributeRepositories,
	services CreateLocationAttributeServices,
) *CreateLocationAttributeUseCase {
	return &CreateLocationAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateLocationAttributeUseCase with grouped parameters instead
func NewCreateLocationAttributeUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
	locationRepo locationpb.LocationDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateLocationAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateLocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
		Location:          locationRepo,
		Attribute:         attributeRepo,
	}

	services := CreateLocationAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateLocationAttributeUseCase(repositories, services)
}

func (uc *CreateLocationAttributeUseCase) Execute(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocationAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes location attribute creation within a transaction
func (uc *CreateLocationAttributeUseCase) executeWithTransaction(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	var result *locationattributepb.CreateLocationAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "location_attribute.errors.creation_failed", "Location attribute creation failed")
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
func (uc *CreateLocationAttributeUseCase) executeCore(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichLocationAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.enrichment_failed", "Business logic enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.CreateLocationAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.creation_failed", "Location attribute creation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateLocationAttributeUseCase) validateInput(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.data_required", ""))
	}
	if req.Data.LocationId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.location_id_required", ""))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.attribute_id_required", ""))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_required", ""))
	}
	return nil
}

// enrichLocationAttributeData adds generated fields and audit information
func (uc *CreateLocationAttributeUseCase) enrichLocationAttributeData(locationAttribute *locationattributepb.LocationAttribute) error {
	now := time.Now()

	// Generate LocationAttribute ID
	if locationAttribute.Id == "" {
		locationAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set location attribute audit fields
	locationAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	locationAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	locationAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	locationAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	// Note: LocationAttribute protobuf does not include active field

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateLocationAttributeUseCase) validateBusinessRules(ctx context.Context, locationAttribute *locationattributepb.LocationAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(locationAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_empty", "Value cannot be empty"))
	}

	if len(locationAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.value_too_long", "Value cannot exceed 1000 characters"))
	}

	// TODO: Additional business rules
	// Example: Check for duplicate location-attribute combinations
	// Example: Validate location and attribute exist
	// Example: Validate attribute type constraints
	// For now, allow all combinations

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateLocationAttributeUseCase) validateEntityReferences(ctx context.Context, locationAttribute *locationattributepb.LocationAttribute) error {
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
