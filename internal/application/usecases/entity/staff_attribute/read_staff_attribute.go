package staff_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// ReadStaffAttributeUseCase handles the business logic for reading staff attributes
// ReadStaffAttributeRepositories groups all repository dependencies
type ReadStaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
}

// ReadStaffAttributeServices groups all business service dependencies
type ReadStaffAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadStaffAttributeUseCase handles the business logic for reading staff attributes
type ReadStaffAttributeUseCase struct {
	repositories ReadStaffAttributeRepositories
	services     ReadStaffAttributeServices
}

// NewReadStaffAttributeUseCase creates use case with grouped dependencies
func NewReadStaffAttributeUseCase(
	repositories ReadStaffAttributeRepositories,
	services ReadStaffAttributeServices,
) *ReadStaffAttributeUseCase {
	return &ReadStaffAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadStaffAttributeUseCaseUngrouped creates a new ReadStaffAttributeUseCase
// Deprecated: Use NewReadStaffAttributeUseCase with grouped parameters instead
func NewReadStaffAttributeUseCaseUngrouped(staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer) *ReadStaffAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadStaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
	}

	services := ReadStaffAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewReadStaffAttributeUseCase(repositories, services)
}

// Execute performs the read staff attribute operation
func (uc *ReadStaffAttributeUseCase) Execute(ctx context.Context, req *staffattributepb.ReadStaffAttributeRequest) (*staffattributepb.ReadStaffAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.ReadStaffAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadStaffAttributeUseCase) validateInput(ctx context.Context, req *staffattributepb.ReadStaffAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.id_required", ""))
	}
	return nil
}
