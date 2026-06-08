package staff_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// DeleteStaffAttributeUseCase handles the business logic for deleting staff attributes
// DeleteStaffAttributeRepositories groups all repository dependencies
type DeleteStaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
}

// DeleteStaffAttributeServices groups all business service dependencies
type DeleteStaffAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteStaffAttributeUseCase handles the business logic for deleting staff attributes
type DeleteStaffAttributeUseCase struct {
	repositories DeleteStaffAttributeRepositories
	services     DeleteStaffAttributeServices
}

// NewDeleteStaffAttributeUseCase creates use case with grouped dependencies
func NewDeleteStaffAttributeUseCase(
	repositories DeleteStaffAttributeRepositories,
	services DeleteStaffAttributeServices,
) *DeleteStaffAttributeUseCase {
	return &DeleteStaffAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteStaffAttributeUseCaseUngrouped creates a new DeleteStaffAttributeUseCase
// Deprecated: Use NewDeleteStaffAttributeUseCase with grouped parameters instead
func NewDeleteStaffAttributeUseCaseUngrouped(staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer) *DeleteStaffAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteStaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
	}

	services := DeleteStaffAttributeServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewDeleteStaffAttributeUseCase(repositories, services)
}

// Execute performs the delete staff attribute operation
func (uc *DeleteStaffAttributeUseCase) Execute(ctx context.Context, req *staffattributepb.DeleteStaffAttributeRequest) (*staffattributepb.DeleteStaffAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.StaffAttribute, entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.DeleteStaffAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff_attribute.errors.deletion_failed", "Staff attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteStaffAttributeUseCase) validateInput(ctx context.Context, req *staffattributepb.DeleteStaffAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff_attribute.validation.request_required", "Request is required for staff attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff_attribute.validation.data_required", "Staff attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "staff_attribute.validation.id_required", "Staff attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteStaffAttributeUseCase) validateBusinessRules(ctx context.Context, req *staffattributepb.DeleteStaffAttributeRequest) error {
	// TODO: Additional business rules
	// Example: Check if attribute is required and cannot be deleted
	// Example: Check permissions for deleting this attribute
	// Example: Validate cascading effects
	// For now, allow all deletions

	return nil
}
