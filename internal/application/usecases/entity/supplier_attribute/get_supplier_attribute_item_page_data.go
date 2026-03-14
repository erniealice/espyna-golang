package supplier_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// GetSupplierAttributeItemPageDataRepositories groups all repository dependencies
type GetSupplierAttributeItemPageDataRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
}

// GetSupplierAttributeItemPageDataServices groups all business service dependencies
type GetSupplierAttributeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierAttributeItemPageDataUseCase handles the business logic for getting supplier attribute item page data
type GetSupplierAttributeItemPageDataUseCase struct {
	repositories GetSupplierAttributeItemPageDataRepositories
	services     GetSupplierAttributeItemPageDataServices
}

// NewGetSupplierAttributeItemPageDataUseCase creates use case with grouped dependencies
func NewGetSupplierAttributeItemPageDataUseCase(
	repositories GetSupplierAttributeItemPageDataRepositories,
	services GetSupplierAttributeItemPageDataServices,
) *GetSupplierAttributeItemPageDataUseCase {
	return &GetSupplierAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetSupplierAttributeItemPageDataUseCaseUngrouped creates a new GetSupplierAttributeItemPageDataUseCase
// Deprecated: Use NewGetSupplierAttributeItemPageDataUseCase with grouped parameters instead
func NewGetSupplierAttributeItemPageDataUseCaseUngrouped(supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer) *GetSupplierAttributeItemPageDataUseCase {
	repositories := GetSupplierAttributeItemPageDataRepositories{
		SupplierAttribute: supplierAttributeRepo,
	}

	services := GetSupplierAttributeItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetSupplierAttributeItemPageDataUseCase(repositories, services)
}

// Execute performs the get supplier attribute item page data operation
func (uc *GetSupplierAttributeItemPageDataUseCase) Execute(ctx context.Context, req *supplierattributepb.GetSupplierAttributeItemPageDataRequest) (*supplierattributepb.GetSupplierAttributeItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.GetSupplierAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.errors.item_page_data_failed", "Failed to retrieve supplier attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetSupplierAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *supplierattributepb.GetSupplierAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "Request is required for supplier attributes [DEFAULT]"))
	}

	if strings.TrimSpace(req.SupplierAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.id_required", "Supplier attribute ID is required [DEFAULT]"))
	}

	if len(req.SupplierAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.id_too_short", "Supplier attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
