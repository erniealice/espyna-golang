package location_attribute

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// ReadLocationAttributeRepositories groups all repository dependencies
type ReadLocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
}

// ReadLocationAttributeServices groups all business service dependencies
type ReadLocationAttributeServices struct {
	TransactionService ports.TransactionService // Current: Database transactions
	TranslationService ports.TranslationService
}

// ReadLocationAttributeUseCase handles the business logic for reading location attributes
type ReadLocationAttributeUseCase struct {
	repositories ReadLocationAttributeRepositories
	services     ReadLocationAttributeServices
}

// NewReadLocationAttributeUseCase creates a new ReadLocationAttributeUseCase
func NewReadLocationAttributeUseCase(
	repositories ReadLocationAttributeRepositories,
	services ReadLocationAttributeServices,
) *ReadLocationAttributeUseCase {
	return &ReadLocationAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadLocationAttributeUseCase with grouped parameters instead
func NewReadLocationAttributeUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
) *ReadLocationAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadLocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
	}

	services := ReadLocationAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewReadLocationAttributeUseCase(repositories, services)
}

func (uc *ReadLocationAttributeUseCase) Execute(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) (*locationattributepb.ReadLocationAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.ReadLocationAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadLocationAttributeUseCase) validateInput(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.request_required", "Request is required for location attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.data_required", "Location attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.id_required", "Location attribute ID is required [DEFAULT]"))
	}
	return nil
}
