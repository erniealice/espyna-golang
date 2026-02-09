package workspace_user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// UpdateWorkspaceUserRepositories groups all repository dependencies
type UpdateWorkspaceUserRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// UpdateWorkspaceUserServices groups all business service dependencies
type UpdateWorkspaceUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateWorkspaceUserUseCase handles the business logic for updating a workspace user
type UpdateWorkspaceUserUseCase struct {
	repositories UpdateWorkspaceUserRepositories
	services     UpdateWorkspaceUserServices
}

// NewUpdateWorkspaceUserUseCase creates use case with grouped dependencies
func NewUpdateWorkspaceUserUseCase(
	repositories UpdateWorkspaceUserRepositories,
	services UpdateWorkspaceUserServices,
) *UpdateWorkspaceUserUseCase {
	return &UpdateWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateWorkspaceUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateWorkspaceUserUseCase with grouped parameters instead
func NewUpdateWorkspaceUserUseCaseUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
) *UpdateWorkspaceUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateWorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
	}

	services := UpdateWorkspaceUserServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateWorkspaceUserUseCase(repositories, services)
}

// NewUpdateWorkspaceUserUseCaseWithTransaction creates a new UpdateWorkspaceUserUseCase with transaction support
// Deprecated: Use NewUpdateWorkspaceUserUseCase with grouped parameters instead

// Execute performs the update workspace user operation
func (uc *UpdateWorkspaceUserUseCase) Execute(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
	// Input validation - must come first to prevent nil pointer dereferences
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspaceUser, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichWorkspaceUserData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUser.UpdateWorkspaceUser(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return successful response
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateWorkspaceUserUseCase) validateInput(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) error {
	if req == nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.request_required", "Request is required for workspace users")
		return errors.New(translatedError)
	}
	if req.Data == nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.data_required", "Workspace user data is required")
		return errors.New(translatedError)
	}
	if req.Data.Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.id_required", "Workspace-User ID is required")
		return errors.New(translatedError)
	}
	if req.Data.WorkspaceId == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.workspace_id_required", "Workspace ID is required")
		return errors.New(translatedError)
	}
	if req.Data.UserId == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.user_id_required", "User ID is required")
		return errors.New(translatedError)
	}
	return nil
}

// enrichWorkspaceUserData adds generated fields and audit information
func (uc *UpdateWorkspaceUserUseCase) enrichWorkspaceUserData(workspaceUser *workspaceuserpb.WorkspaceUser) error {
	now := time.Now()

	// Set audit fields for update
	workspaceUser.DateModified = &[]int64{now.Unix()}[0]
	workspaceUser.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Initialize workspace_user_roles slice if nil
	if workspaceUser.WorkspaceUserRoles == nil {
		workspaceUser.WorkspaceUserRoles = []*workspaceuserrolepb.WorkspaceUserRole{}
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkspaceUserUseCase) validateBusinessRules(workspaceUser *workspaceuserpb.WorkspaceUser) error {
	// Validate workspace and user relationship
	if workspaceUser.WorkspaceId == workspaceUser.UserId {
		return fmt.Errorf("workspace ID and user ID cannot be the same")
	}

	// Validate workspace_user_roles if provided
	if len(workspaceUser.WorkspaceUserRoles) > 0 {
		for _, userRole := range workspaceUser.WorkspaceUserRoles {
			if userRole == nil {
				return fmt.Errorf("workspace user roles cannot contain null entries")
			}
			// Validate the role within the userRole object
			if userRole.Role != nil && strings.TrimSpace(userRole.Role.Name) == "" {
				return fmt.Errorf("roles cannot contain empty strings")
			}
		}

		// Check for duplicate roles
		roleSet := make(map[string]bool)
		for _, userRole := range workspaceUser.WorkspaceUserRoles {
			if userRole != nil && userRole.Role != nil {
				roleName := userRole.Role.Name
				if roleSet[roleName] {
					return fmt.Errorf("duplicate roles are not allowed")
				}
				roleSet[roleName] = true
			}
		}
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateWorkspaceUserUseCase) validateEntityReferences(ctx context.Context, workspaceUser *workspaceuserpb.WorkspaceUser) error {
	// Validate Workspace entity reference
	if workspaceUser.WorkspaceId != "" {
		workspace, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
			Data: &workspacepb.Workspace{Id: workspaceUser.WorkspaceId},
		})
		if err != nil {
			return fmt.Errorf("failed to validate workspace entity reference: %w", err)
		}
		if workspace == nil || workspace.Data == nil || len(workspace.Data) == 0 {
			return fmt.Errorf("referenced workspace with ID '%s' does not exist", workspaceUser.WorkspaceId)
		}
		if !workspace.Data[0].Active {
			return fmt.Errorf("referenced workspace with ID '%s' is not active", workspaceUser.WorkspaceId)
		}
	}

	// Validate User entity reference
	if workspaceUser.UserId != "" {
		user, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
			Data: &userpb.User{Id: workspaceUser.UserId},
		})
		if err != nil {
			return fmt.Errorf("failed to validate user entity reference: %w", err)
		}
		if user == nil || user.Data == nil || len(user.Data) == 0 {
			return fmt.Errorf("referenced user with ID '%s' does not exist", workspaceUser.UserId)
		}
		if !user.Data[0].Active {
			return fmt.Errorf("referenced user with ID '%s' is not active", workspaceUser.UserId)
		}
	}

	return nil
}
