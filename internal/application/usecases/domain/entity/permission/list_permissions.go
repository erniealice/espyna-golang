package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// ListPermissionsRepositories groups all repository dependencies
type ListPermissionsRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// ListPermissionsServices groups all business service dependencies
type ListPermissionsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListPermissionsUseCase handles the business logic for listing permissions
type ListPermissionsUseCase struct {
	repositories ListPermissionsRepositories
	services     ListPermissionsServices
}

// NewListPermissionsUseCase creates use case with grouped dependencies
func NewListPermissionsUseCase(
	repositories ListPermissionsRepositories,
	services ListPermissionsServices,
) *ListPermissionsUseCase {
	return &ListPermissionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListPermissionsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListPermissionsUseCase with grouped parameters instead
func NewListPermissionsUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *ListPermissionsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListPermissionsRepositories{
		Permission: permissionRepo,
	}

	services := ListPermissionsServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListPermissionsUseCase(repositories, services)
}

func (uc *ListPermissionsUseCase) Execute(ctx context.Context, req *permissionpb.ListPermissionsRequest) (*permissionpb.ListPermissionsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Permission,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Permission.ListPermissions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.errors.list_failed", "Failed to retrieve permissions [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListPermissionsUseCase) validateInput(ctx context.Context, req *permissionpb.ListPermissionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListPermissionsUseCase) validateBusinessRules(ctx context.Context, req *permissionpb.ListPermissionsRequest) error {
	// No additional business rules for listing permissions
	// Pagination is not supported in current protobuf definition
	return nil
}
