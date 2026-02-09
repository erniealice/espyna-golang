package client_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// GetClientAttributeItemPageDataUseCase handles the business logic for getting client attribute item page data
// GetClientAttributeItemPageDataRepositories groups all repository dependencies
type GetClientAttributeItemPageDataRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
}

// GetClientAttributeItemPageDataServices groups all business service dependencies
type GetClientAttributeItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetClientAttributeItemPageDataUseCase handles the business logic for getting client attribute item page data
type GetClientAttributeItemPageDataUseCase struct {
	repositories GetClientAttributeItemPageDataRepositories
	services     GetClientAttributeItemPageDataServices
}

// NewGetClientAttributeItemPageDataUseCase creates use case with grouped dependencies
func NewGetClientAttributeItemPageDataUseCase(
	repositories GetClientAttributeItemPageDataRepositories,
	services GetClientAttributeItemPageDataServices,
) *GetClientAttributeItemPageDataUseCase {
	return &GetClientAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetClientAttributeItemPageDataUseCaseUngrouped creates a new GetClientAttributeItemPageDataUseCase
// Deprecated: Use NewGetClientAttributeItemPageDataUseCase with grouped parameters instead
func NewGetClientAttributeItemPageDataUseCaseUngrouped(clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer) *GetClientAttributeItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetClientAttributeItemPageDataRepositories{
		ClientAttribute: clientAttributeRepo,
	}

	services := GetClientAttributeItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetClientAttributeItemPageDataUseCase(repositories, services)
}

// Execute performs the get client attribute item page data operation
func (uc *GetClientAttributeItemPageDataUseCase) Execute(ctx context.Context, req *clientattributepb.GetClientAttributeItemPageDataRequest) (*clientattributepb.GetClientAttributeItemPageDataResponse, error) {
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

	// Call repository
	resp, err := uc.repositories.ClientAttribute.GetClientAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.errors.item_page_data_failed", "Failed to retrieve client attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetClientAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *clientattributepb.GetClientAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", "Request is required for client attributes [DEFAULT]"))
	}

	// Validate client attribute ID
	if strings.TrimSpace(req.ClientAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.id_required", "Client attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.ClientAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.id_too_short", "Client attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
