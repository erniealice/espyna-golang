package staff_attribute

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
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// CreateStaffAttributeRepositories groups all repository dependencies
type CreateStaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
	Staff          staffpb.StaffDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// CreateStaffAttributeServices groups all business service dependencies
type CreateStaffAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateStaffAttributeUseCase handles the business logic for creating staff attributes
type CreateStaffAttributeUseCase struct {
	repositories CreateStaffAttributeRepositories
	services     CreateStaffAttributeServices
}

// NewCreateStaffAttributeUseCase creates use case with grouped dependencies
func NewCreateStaffAttributeUseCase(
	repositories CreateStaffAttributeRepositories,
	services CreateStaffAttributeServices,
) *CreateStaffAttributeUseCase {
	return &CreateStaffAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateStaffAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateStaffAttributeUseCase with grouped parameters instead
func NewCreateStaffAttributeUseCaseUngrouped(
	staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer,
	staffRepo staffpb.StaffDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateStaffAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateStaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
		Staff:          staffRepo,
		Attribute:      attributeRepo,
	}

	services := CreateStaffAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateStaffAttributeUseCase(repositories, services)
}

// NewCreateStaffAttributeUseCaseWithTransaction creates a new CreateStaffAttributeUseCase with transaction support
// Deprecated: Use NewCreateStaffAttributeUseCase with grouped parameters instead

// Execute performs the create staff attribute operation
func (uc *CreateStaffAttributeUseCase) Execute(ctx context.Context, req *staffattributepb.CreateStaffAttributeRequest) (*staffattributepb.CreateStaffAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityStaffAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichStaffAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.CreateStaffAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.creation_failed", "Staff attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateStaffAttributeUseCase) validateInput(ctx context.Context, req *staffattributepb.CreateStaffAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.data_required", ""))
	}
	if req.Data.StaffId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.staff_id_required", ""))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.attribute_id_required", ""))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_required", ""))
	}
	return nil
}

// enrichStaffAttributeData adds generated fields and audit information
func (uc *CreateStaffAttributeUseCase) enrichStaffAttributeData(staffAttribute *staffattributepb.StaffAttribute) error {
	now := time.Now()

	// Generate StaffAttribute ID
	if staffAttribute.Id == "" {
		staffAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set staff attribute audit fields
	staffAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	staffAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	staffAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	staffAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	staffAttribute.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateStaffAttributeUseCase) validateBusinessRules(ctx context.Context, staffAttribute *staffattributepb.StaffAttribute) error {
	// Validate value length
	if len(strings.TrimSpace(staffAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(staffAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	// TODO: Additional business rules
	// Example: Check for duplicate staff-attribute combinations
	// Example: Validate attribute type constraints
	// For now, allow all combinations

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateStaffAttributeUseCase) validateEntityReferences(ctx context.Context, staffAttribute *staffattributepb.StaffAttribute) error {
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.staff_not_found", "")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.attribute_not_found", "")
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
