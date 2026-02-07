package client_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

// DeleteClientAttributeUseCase handles the business logic for deleting client attributes
// DeleteClientAttributeRepositories groups all repository dependencies
type DeleteClientAttributeRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
}

// DeleteClientAttributeServices groups all business service dependencies
type DeleteClientAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteClientAttributeUseCase handles the business logic for deleting client attributes
type DeleteClientAttributeUseCase struct {
	repositories DeleteClientAttributeRepositories
	services     DeleteClientAttributeServices
}

// NewDeleteClientAttributeUseCase creates use case with grouped dependencies
func NewDeleteClientAttributeUseCase(
	repositories DeleteClientAttributeRepositories,
	services DeleteClientAttributeServices,
) *DeleteClientAttributeUseCase {
	return &DeleteClientAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteClientAttributeUseCaseUngrouped creates a new DeleteClientAttributeUseCase
// Deprecated: Use NewDeleteClientAttributeUseCase with grouped parameters instead
func NewDeleteClientAttributeUseCaseUngrouped(clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer) *DeleteClientAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
	}

	services := DeleteClientAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewDeleteClientAttributeUseCase(repositories, services)
}

// Execute performs the delete client attribute operation
func (uc *DeleteClientAttributeUseCase) Execute(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) (*clientattributepb.DeleteClientAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.DeleteClientAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.deletion_failed", "Client attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteClientAttributeUseCase) validateInput(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "Request is required for client attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.data_required", "Client attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.id_required", "Client attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteClientAttributeUseCase) validateBusinessRules(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) error {
	// TODO: Additional business rules
	// Example: Check if attribute is required and cannot be deleted
	// Example: Check permissions for deleting this attribute
	// Example: Validate cascading effects
	// For now, allow all deletions

	return nil
}
