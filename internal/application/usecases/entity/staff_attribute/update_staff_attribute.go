package staff_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// UpdateStaffAttributeUseCase handles the business logic for updating staff attributes
// UpdateStaffAttributeRepositories groups all repository dependencies
type UpdateStaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
	Staff          staffpb.StaffDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// UpdateStaffAttributeServices groups all business service dependencies
type UpdateStaffAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateStaffAttributeUseCase handles the business logic for updating staff attributes
type UpdateStaffAttributeUseCase struct {
	repositories UpdateStaffAttributeRepositories
	services     UpdateStaffAttributeServices
}

// NewUpdateStaffAttributeUseCase creates use case with grouped dependencies
func NewUpdateStaffAttributeUseCase(
	repositories UpdateStaffAttributeRepositories,
	services UpdateStaffAttributeServices,
) *UpdateStaffAttributeUseCase {
	return &UpdateStaffAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateStaffAttributeUseCaseUngrouped creates a new UpdateStaffAttributeUseCase
// Deprecated: Use NewUpdateStaffAttributeUseCase with grouped parameters instead
func NewUpdateStaffAttributeUseCaseUngrouped(
	staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer,
	staffRepo staffpb.StaffDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateStaffAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateStaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
		Staff:          staffRepo,
		Attribute:      attributeRepo,
	}

	services := UpdateStaffAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateStaffAttributeUseCase(repositories, services)
}

// Execute performs the update staff attribute operation
func (uc *UpdateStaffAttributeUseCase) Execute(ctx context.Context, req *staffattributepb.UpdateStaffAttributeRequest) (*staffattributepb.UpdateStaffAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichStaffAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.UpdateStaffAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.update_failed", "Staff attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateStaffAttributeUseCase) validateInput(ctx context.Context, req *staffattributepb.UpdateStaffAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", "Request is required for staff attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.data_required", "Staff attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.id_required", "Staff attribute ID is required [DEFAULT]"))
	}
	if req.Data.StaffId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.staff_id_required", "Staff ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichStaffAttributeData adds updated audit information
func (uc *UpdateStaffAttributeUseCase) enrichStaffAttributeData(staffAttribute *staffattributepb.StaffAttribute) error {
	now := time.Now()

	// Update modification timestamp
	staffAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	staffAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateStaffAttributeUseCase) validateBusinessRules(ctx context.Context, staffAttribute *staffattributepb.StaffAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(staffAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(staffAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Validate staff and attribute exist
	// Example: Validate attribute type constraints
	// Example: Check permissions for updating this attribute
	// For now, allow all updates

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateStaffAttributeUseCase) validateEntityReferences(ctx context.Context, staffAttribute *staffattributepb.StaffAttribute) error {
	// Validate Staff entity reference
	if staffAttribute.StaffId != "" {
		staff, err := uc.repositories.Staff.ReadStaff(ctx, &staffpb.ReadStaffRequest{
			Data: &staffpb.Staff{Id: staffAttribute.StaffId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.staff_reference_validation_failed", "Failed to validate staff entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if staff == nil || staff.Data == nil || len(staff.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.staff_not_found", "Referenced staff with ID '{staffId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{staffId}", staffAttribute.StaffId)
			return errors.New(translatedError)
		}
		if !staff.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.staff_not_active", "Referenced staff with ID '{staffId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{staffId}", staffAttribute.StaffId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if staffAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: staffAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", staffAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", staffAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
