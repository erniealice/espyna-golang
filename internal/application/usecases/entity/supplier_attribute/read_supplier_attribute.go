package supplier_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// ReadSupplierAttributeRepositories groups all repository dependencies
type ReadSupplierAttributeRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
}

// ReadSupplierAttributeServices groups all business service dependencies
type ReadSupplierAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadSupplierAttributeUseCase handles the business logic for reading supplier attributes
type ReadSupplierAttributeUseCase struct {
	repositories ReadSupplierAttributeRepositories
	services     ReadSupplierAttributeServices
}

// NewReadSupplierAttributeUseCase creates use case with grouped dependencies
func NewReadSupplierAttributeUseCase(
	repositories ReadSupplierAttributeRepositories,
	services ReadSupplierAttributeServices,
) *ReadSupplierAttributeUseCase {
	return &ReadSupplierAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadSupplierAttributeUseCaseUngrouped creates a new ReadSupplierAttributeUseCase
// Deprecated: Use NewReadSupplierAttributeUseCase with grouped parameters instead
func NewReadSupplierAttributeUseCaseUngrouped(supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer) *ReadSupplierAttributeUseCase {
	repositories := ReadSupplierAttributeRepositories{
		SupplierAttribute: supplierAttributeRepo,
	}

	services := ReadSupplierAttributeServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadSupplierAttributeUseCase(repositories, services)
}

// Execute performs the read supplier attribute operation
func (uc *ReadSupplierAttributeUseCase) Execute(ctx context.Context, req *supplierattributepb.ReadSupplierAttributeRequest) (*supplierattributepb.ReadSupplierAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier_attribute", ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.ReadSupplierAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadSupplierAttributeUseCase) validateInput(ctx context.Context, req *supplierattributepb.ReadSupplierAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.data_required", "[ERR-DEFAULT] Supplier attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier_attribute.validation.id_required", "[ERR-DEFAULT] Attribute ID is required"))
	}
	return nil
}
