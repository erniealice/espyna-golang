package supplier_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
)

// ListSupplierAttributesRepositories groups all repository dependencies
type ListSupplierAttributesRepositories struct {
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer // Primary entity repository
}

// ListSupplierAttributesServices groups all business service dependencies
type ListSupplierAttributesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListSupplierAttributesUseCase handles the business logic for listing supplier attributes
type ListSupplierAttributesUseCase struct {
	repositories ListSupplierAttributesRepositories
	services     ListSupplierAttributesServices
}

// NewListSupplierAttributesUseCase creates use case with grouped dependencies
func NewListSupplierAttributesUseCase(
	repositories ListSupplierAttributesRepositories,
	services ListSupplierAttributesServices,
) *ListSupplierAttributesUseCase {
	return &ListSupplierAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListSupplierAttributesUseCaseUngrouped creates a new ListSupplierAttributesUseCase
// Deprecated: Use NewListSupplierAttributesUseCase with grouped parameters instead
func NewListSupplierAttributesUseCaseUngrouped(supplierAttributeRepo supplierattributepb.SupplierAttributeDomainServiceServer) *ListSupplierAttributesUseCase {
	repositories := ListSupplierAttributesRepositories{
		SupplierAttribute: supplierAttributeRepo,
	}

	services := ListSupplierAttributesServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListSupplierAttributesUseCase(repositories, services)
}

// Execute performs the list supplier attributes operation
func (uc *ListSupplierAttributesUseCase) Execute(ctx context.Context, req *supplierattributepb.ListSupplierAttributesRequest) (*supplierattributepb.ListSupplierAttributesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "supplier_attribute",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.SupplierAttribute.ListSupplierAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_attribute.errors.list_failed", "Failed to retrieve supplier attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListSupplierAttributesUseCase) validateInput(ctx context.Context, req *supplierattributepb.ListSupplierAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_attribute.validation.request_required", "Request is required for supplier attributes [DEFAULT]"))
	}
	return nil
}
