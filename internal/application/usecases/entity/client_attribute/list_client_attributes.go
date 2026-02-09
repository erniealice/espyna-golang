package client_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// ListClientAttributesUseCase handles the business logic for listing client attributes
// ListClientAttributesRepositories groups all repository dependencies
type ListClientAttributesRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
}

// ListClientAttributesServices groups all business service dependencies
type ListClientAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListClientAttributesUseCase handles the business logic for listing client attributes
type ListClientAttributesUseCase struct {
	repositories ListClientAttributesRepositories
	services     ListClientAttributesServices
}

// NewListClientAttributesUseCase creates use case with grouped dependencies
func NewListClientAttributesUseCase(
	repositories ListClientAttributesRepositories,
	services ListClientAttributesServices,
) *ListClientAttributesUseCase {
	return &ListClientAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListClientAttributesUseCaseUngrouped creates a new ListClientAttributesUseCase
// Deprecated: Use NewListClientAttributesUseCase with grouped parameters instead
func NewListClientAttributesUseCaseUngrouped(clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer) *ListClientAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListClientAttributesRepositories{
		ClientAttribute: clientAttributeRepo,
	}

	services := ListClientAttributesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListClientAttributesUseCase(repositories, services)
}

// Execute performs the list client attributes operation
func (uc *ListClientAttributesUseCase) Execute(ctx context.Context, req *clientattributepb.ListClientAttributesRequest) (*clientattributepb.ListClientAttributesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClientAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.ListClientAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.list_failed", "Failed to retrieve client attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListClientAttributesUseCase) validateInput(ctx context.Context, req *clientattributepb.ListClientAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "Request is required for client attributes [DEFAULT]"))
	}

	// No additional business rules for listing client attributes
	// Pagination is not supported in current protobuf definition

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *ListClientAttributesUseCase) applyDefaults(req *clientattributepb.ListClientAttributesRequest) error {
	// No defaults to apply
	// Pagination is not supported in current protobuf definition
	return nil
}
