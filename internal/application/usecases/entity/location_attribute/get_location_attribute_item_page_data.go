package location_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

type GetLocationAttributeItemPageDataRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer
}

type GetLocationAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetLocationAttributeItemPageDataUseCase handles the business logic for getting location attribute item page data
type GetLocationAttributeItemPageDataUseCase struct {
	repositories GetLocationAttributeItemPageDataRepositories
	services     GetLocationAttributeItemPageDataServices
}

// NewGetLocationAttributeItemPageDataUseCase creates a new GetLocationAttributeItemPageDataUseCase
func NewGetLocationAttributeItemPageDataUseCase(
	repositories GetLocationAttributeItemPageDataRepositories,
	services GetLocationAttributeItemPageDataServices,
) *GetLocationAttributeItemPageDataUseCase {
	return &GetLocationAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get location attribute item page data operation
func (uc *GetLocationAttributeItemPageDataUseCase) Execute(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeItemPageDataRequest,
) (*locationattributepb.GetLocationAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.LocationAttributeId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes location attribute item page data retrieval within a transaction
func (uc *GetLocationAttributeItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeItemPageDataRequest,
) (*locationattributepb.GetLocationAttributeItemPageDataResponse, error) {
	var result *locationattributepb.GetLocationAttributeItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"location_attribute.errors.item_page_data_failed",
				"location attribute item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting location attribute item page data
func (uc *GetLocationAttributeItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeItemPageDataRequest,
) (*locationattributepb.GetLocationAttributeItemPageDataResponse, error) {
	// Create read request for the location attribute
	readReq := &locationattributepb.ReadLocationAttributeRequest{
		Data: &locationattributepb.LocationAttribute{
			Id: req.LocationAttributeId,
		},
	}

	// Retrieve the location attribute
	readResp, err := uc.repositories.LocationAttribute.ReadLocationAttribute(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.errors.read_failed",
			"failed to retrieve location attribute: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.errors.not_found",
			"location attribute not found",
		))
	}

	// Get the location attribute (should be only one)
	locationAttribute := readResp.Data[0]

	// Validate that we got the expected location attribute
	if locationAttribute.Id != req.LocationAttributeId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.errors.id_mismatch",
			"retrieved location attribute ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (location details, attribute details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the location attribute as-is
	return &locationattributepb.GetLocationAttributeItemPageDataResponse{
		LocationAttribute: locationAttribute,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetLocationAttributeItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.validation.request_required",
			"request is required",
		))
	}

	if req.LocationAttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.validation.id_required",
			"location attribute ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading location attribute item page data
func (uc *GetLocationAttributeItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	locationAttributeId string,
) error {
	// Validate location attribute ID format
	if len(locationAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"location_attribute.validation.id_too_short",
			"location attribute ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this location attribute
	// - Validate location attribute belongs to the current user's organization
	// - Check if location attribute is in a state that allows viewing
	// - Rate limiting for location attribute access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like location and attribute details
// This would be called from executeCore if needed
func (uc *GetLocationAttributeItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	locationAttribute *locationattributepb.LocationAttribute,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to location and attribute repositories
	// to populate the nested location and attribute objects if they're not already loaded

	// Example implementation would be:
	// if locationAttribute.Location == nil && locationAttribute.LocationId != "" {
	//     // Load location data
	// }
	// if locationAttribute.Attribute == nil && locationAttribute.AttributeId != "" {
	//     // Load attribute data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetLocationAttributeItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	locationAttribute *locationattributepb.LocationAttribute,
) *locationattributepb.LocationAttribute {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return locationAttribute
}

// checkAccessPermissions validates user has permission to access this location attribute
func (uc *GetLocationAttributeItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	locationAttributeId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating location attribute belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
