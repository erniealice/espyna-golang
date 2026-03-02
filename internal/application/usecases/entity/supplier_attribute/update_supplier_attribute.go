package supplier_attribute

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
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// UpdateSupplierAttributeRepositories groups all repository dependencies
type UpdateSupplierAttributeRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
	Supplier          supplierpb.SupplierDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// UpdateSupplierAttributeServices groups all business service dependencies
type UpdateSupplierAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateSupplierAttributeUseCase handles the business logic for updating supplier attributes
type UpdateSupplierAttributeUseCase struct {
	repositories UpdateSupplierAttributeRepositories
	services     UpdateSupplierAttributeServices
}

// NewUpdateSupplierAttributeUseCase creates use case with grouped dependencies
func NewUpdateSupplierAttributeUseCase(
	repositories UpdateSupplierAttributeRepositories,
	services UpdateSupplierAttributeServices,
) *UpdateSupplierAttributeUseCase {
	return &UpdateSupplierAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateSupplierAttributeUseCaseUngrouped creates a new UpdateSupplierAttributeUseCase
// Deprecated: Use NewUpdateSupplierAttributeUseCase with grouped parameters instead
func NewUpdateSupplierAttributeUseCaseUngrouped(
	supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer,
	supplierRepo supplierpb.SupplierDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
) *UpdateSupplierAttributeUseCase {
	repositories := UpdateSupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
		Supplier:          supplierRepo,
		Attribute:         attributeRepo,
	}

	services := UpdateSupplierAttributeServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateSupplierAttributeUseCase(repositories, services)
}

// Execute performs the update supplier attribute operation
func (uc *UpdateSupplierAttributeUseCase) Execute(ctx context.Context, req *supplierattributepb.UpdateSupplierAttributeRequest) (*supplierattributepb.UpdateSupplierAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichSupplierAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.UpdateSupplierAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.update_failed", "Supplier attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateSupplierAttributeUseCase) validateInput(ctx context.Context, req *supplierattributepb.UpdateSupplierAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "Request is required for supplier attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.data_required", "Supplier attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.id_required", "Supplier attribute ID is required [DEFAULT]"))
	}
	if req.Data.SupplierId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.supplier_id_required", "Supplier ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichSupplierAttributeData adds updated audit information
func (uc *UpdateSupplierAttributeUseCase) enrichSupplierAttributeData(supplierAttribute *supplierattributepb.SupplierAttribute) error {
	now := time.Now()

	// Update modification timestamp
	supplierAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	supplierAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateSupplierAttributeUseCase) validateBusinessRules(ctx context.Context, supplierAttribute *supplierattributepb.SupplierAttribute) error {
	if len(strings.TrimSpace(supplierAttribute.Value)) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.value_empty", "Value cannot be empty [DEFAULT]"))
	}

	if len(supplierAttribute.Value) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.value_too_long", "Value cannot exceed 1000 characters [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateSupplierAttributeUseCase) validateEntityReferences(ctx context.Context, supplierAttribute *supplierattributepb.SupplierAttribute) error {
	// Validate Supplier entity reference
	if supplierAttribute.SupplierId != "" {
		supplier, err := uc.repositories.Supplier.ReadSupplier(ctx, &supplierpb.ReadSupplierRequest{
			Data: &supplierpb.Supplier{Id: supplierAttribute.SupplierId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.supplier_reference_validation_failed", "Failed to validate supplier entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if supplier == nil || supplier.Data == nil || len(supplier.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.supplier_not_found", "Referenced supplier with ID '{supplierId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{supplierId}", supplierAttribute.SupplierId)
			return errors.New(translatedError)
		}
		if !supplier.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.supplier_not_active", "Referenced supplier with ID '{supplierId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{supplierId}", supplierAttribute.SupplierId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if supplierAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: supplierAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.attribute_not_found", "Referenced attribute with ID '{attributeId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", supplierAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", supplierAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
