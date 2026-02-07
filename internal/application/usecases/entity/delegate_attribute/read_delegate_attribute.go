package delegate_attribute

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// ReadDelegateAttributeUseCase handles the business logic for reading delegate attributes
// ReadDelegateAttributeRepositories groups all repository dependencies
type ReadDelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
}

// ReadDelegateAttributeServices groups all business service dependencies
type ReadDelegateAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadDelegateAttributeUseCase handles the business logic for reading delegate attributes
type ReadDelegateAttributeUseCase struct {
	repositories ReadDelegateAttributeRepositories
	services     ReadDelegateAttributeServices
}

// NewReadDelegateAttributeUseCase creates use case with grouped dependencies
func NewReadDelegateAttributeUseCase(
	repositories ReadDelegateAttributeRepositories,
	services ReadDelegateAttributeServices,
) *ReadDelegateAttributeUseCase {
	return &ReadDelegateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadDelegateAttributeUseCaseUngrouped creates a new ReadDelegateAttributeUseCase
// Deprecated: Use NewReadDelegateAttributeUseCase with grouped parameters instead
func NewReadDelegateAttributeUseCaseUngrouped(delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer) *ReadDelegateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadDelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
	}

	services := ReadDelegateAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewReadDelegateAttributeUseCase(repositories, services)
}

// Execute performs the read delegate attribute operation
func (uc *ReadDelegateAttributeUseCase) Execute(ctx context.Context, req *delegateattributepb.ReadDelegateAttributeRequest) (*delegateattributepb.ReadDelegateAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.DelegateAttribute.ReadDelegateAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadDelegateAttributeUseCase) validateInput(ctx context.Context, req *delegateattributepb.ReadDelegateAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate_attribute.validation.id_required", ""))
	}
	return nil
}
