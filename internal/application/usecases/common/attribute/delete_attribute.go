package attribute

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// DeleteAttributeUseCase handles the business logic for deleting attributes
// DeleteAttributeRepositories groups all repository dependencies
type DeleteAttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// DeleteAttributeServices groups all business service dependencies
type DeleteAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteAttributeUseCase handles the business logic for deleting attributes
type DeleteAttributeUseCase struct {
	repositories DeleteAttributeRepositories
	services     DeleteAttributeServices
}

// NewDeleteAttributeUseCase creates use case with grouped dependencies
func NewDeleteAttributeUseCase(
	repositories DeleteAttributeRepositories,
	services DeleteAttributeServices,
) *DeleteAttributeUseCase {
	return &DeleteAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteAttributeUseCaseUngrouped creates a new DeleteAttributeUseCase
// Deprecated: Use NewDeleteAttributeUseCase with grouped parameters instead
func NewDeleteAttributeUseCaseUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *DeleteAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteAttributeRepositories{
		Attribute: attributeRepo,
	}

	services := DeleteAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewDeleteAttributeUseCase(repositories, services)
}

// Execute performs the delete attribute operation
func (uc *DeleteAttributeUseCase) Execute(ctx context.Context, req *attributepb.DeleteAttributeRequest) (*attributepb.DeleteAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository (performs soft delete)
	return uc.repositories.Attribute.DeleteAttribute(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteAttributeUseCase) validateInput(req *attributepb.DeleteAttributeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("attribute data is required")
	}
	if req.Data.Id == "" {
		return errors.New("attribute ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteAttributeUseCase) validateBusinessRules(ctx context.Context, attribute *attributepb.Attribute) error {
	// TODO: Add validation to prevent deletion of attributes that are in use
	// This would require checking if any client_attribute, location_attribute, or product_attribute
	// records reference this attribute ID. For now, we allow deletion.

	// Future enhancement: Implement cascade delete or prevent deletion of referenced attributes
	// Example:
	// - Check client_attribute table for references
	// - Check location_attribute table for references
	// - Check product_attribute table for references
	// - Return error if references exist, or implement cascade delete

	return nil
}
