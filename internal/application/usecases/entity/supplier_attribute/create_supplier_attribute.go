package supplier_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// CreateSupplierAttributeRepositories groups all repository dependencies
type CreateSupplierAttributeRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
	Supplier          supplierpb.SupplierDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// CreateSupplierAttributeServices groups all business service dependencies
type CreateSupplierAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierAttributeUseCase handles the business logic for creating supplier attributes
type CreateSupplierAttributeUseCase struct {
	repositories CreateSupplierAttributeRepositories
	services     CreateSupplierAttributeServices
}

// NewCreateSupplierAttributeUseCase creates use case with grouped dependencies
func NewCreateSupplierAttributeUseCase(
	repositories CreateSupplierAttributeRepositories,
	services CreateSupplierAttributeServices,
) *CreateSupplierAttributeUseCase {
	return &CreateSupplierAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateSupplierAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateSupplierAttributeUseCase with grouped parameters instead
func NewCreateSupplierAttributeUseCaseUngrouped(
	supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer,
	supplierRepo supplierpb.SupplierDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateSupplierAttributeUseCase {
	repositories := CreateSupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
		Supplier:          supplierRepo,
		Attribute:         attributeRepo,
	}

	services := CreateSupplierAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateSupplierAttributeUseCase(repositories, services)
}

// Execute performs the create supplier attribute operation
func (uc *CreateSupplierAttributeUseCase) Execute(ctx context.Context, req *supplierattributepb.CreateSupplierAttributeRequest) (*supplierattributepb.CreateSupplierAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichSupplierAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.CreateSupplierAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.creation_failed", "Supplier attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateSupplierAttributeUseCase) validateInput(ctx context.Context, req *supplierattributepb.CreateSupplierAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.data_required", "[ERR-DEFAULT] Supplier attribute data is required"))
	}
	if req.Data.SupplierId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.supplier_id_required", "[ERR-DEFAULT] Supplier ID is required"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.attribute_id_required", "[ERR-DEFAULT] Attribute ID is required"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.value_required", "[ERR-DEFAULT] Attribute value is required"))
	}
	return nil
}

// enrichSupplierAttributeData adds generated fields and audit information
func (uc *CreateSupplierAttributeUseCase) enrichSupplierAttributeData(supplierAttribute *supplierattributepb.SupplierAttribute) error {
	now := time.Now()

	// Generate SupplierAttribute ID
	if supplierAttribute.Id == "" {
		supplierAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set supplier attribute audit fields
	supplierAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	supplierAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	supplierAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	supplierAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	supplierAttribute.Active = true

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateSupplierAttributeUseCase) validateEntityReferences(ctx context.Context, supplierAttribute *supplierattributepb.SupplierAttribute) error {
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.supplier_not_found", "[ERR-DEFAULT] Supplier not found")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.attribute_not_found", "[ERR-DEFAULT] Attribute not found")
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
