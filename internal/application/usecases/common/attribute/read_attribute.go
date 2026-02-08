package attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ReadAttributeUseCase handles the business logic for reading attributes
// ReadAttributeRepositories groups all repository dependencies
type ReadAttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// ReadAttributeServices groups all business service dependencies
type ReadAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadAttributeUseCase handles the business logic for reading attributes
type ReadAttributeUseCase struct {
	repositories ReadAttributeRepositories
	services     ReadAttributeServices
}

// NewReadAttributeUseCase creates use case with grouped dependencies
func NewReadAttributeUseCase(
	repositories ReadAttributeRepositories,
	services ReadAttributeServices,
) *ReadAttributeUseCase {
	return &ReadAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadAttributeUseCaseUngrouped creates a new ReadAttributeUseCase
// Deprecated: Use NewReadAttributeUseCase with grouped parameters instead
func NewReadAttributeUseCaseUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *ReadAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadAttributeRepositories{
		Attribute: attributeRepo,
	}

	services := ReadAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
	}

	return NewReadAttributeUseCase(repositories, services)
}

// Execute performs the read attribute operation
func (uc *ReadAttributeUseCase) Execute(ctx context.Context, req *attributepb.ReadAttributeRequest) (*attributepb.ReadAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Attribute.ReadAttribute(ctx, req)
}

// validateInput validates the input request
func (uc *ReadAttributeUseCase) validateInput(req *attributepb.ReadAttributeRequest) error {
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
