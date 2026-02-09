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

// CreateWorkspaceUserRepositories groups all repository dependencies
type CreateWorkspaceUserRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// CreateWorkspaceUserServices groups all business service dependencies
type CreateWorkspaceUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateWorkspaceUserUseCase handles the business logic for creating workspace users
type CreateWorkspaceUserUseCase struct {
	repositories CreateWorkspaceUserRepositories
	services     CreateWorkspaceUserServices
}

// NewCreateWorkspaceUserUseCase creates use case with grouped dependencies
func NewCreateWorkspaceUserUseCase(
	repositories CreateWorkspaceUserRepositories,
	services CreateWorkspaceUserServices,
) *CreateWorkspaceUserUseCase {
	return &CreateWorkspaceUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateWorkspaceUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateWorkspaceUserUseCase with grouped parameters instead
func NewCreateWorkspaceUserUseCaseUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
	workspaceRepo workspacepb.WorkspaceDomainServiceServer,
	userRepo userpb.UserDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *CreateWorkspaceUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateWorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     workspaceRepo,
		User:          userRepo,
	}

	services := CreateWorkspaceUserServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateWorkspaceUserUseCase(repositories, services)
}

// NewCreateWorkspaceUserUseCaseWithTransaction creates a new CreateWorkspaceUserUseCase with transaction support
// Deprecated: Use NewCreateWorkspaceUserUseCase with grouped parameters instead

// Execute performs the create workspace user operation
func (uc *CreateWorkspaceUserUseCase) Execute(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
	// Input validation - must come first to prevent nil pointer dereferences
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspaceUser, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichWorkspaceUserData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUser.CreateWorkspaceUser(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.creation_failed", "Workspace-User creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateWorkspaceUserUseCase) validateInput(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.request_required", "Request is required for workspace users [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.data_required", "Workspace-User data is required [DEFAULT]"))
	}
	if req.Data.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.workspace_id_required", "Workspace ID is required [DEFAULT]"))
	}
	if req.Data.UserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.user_id_required", "User ID is required [DEFAULT]"))
	}
	return nil
}

// enrichWorkspaceUserData adds generated fields and audit information
func (uc *CreateWorkspaceUserUseCase) enrichWorkspaceUserData(workspaceUser *workspaceuserpb.WorkspaceUser) error {
	now := time.Now()

	// Generate WorkspaceUser ID if not provided
	if workspaceUser.Id == "" {
		workspaceUser.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	workspaceUser.DateCreated = &[]int64{now.Unix()}[0]
	workspaceUser.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	workspaceUser.DateModified = &[]int64{now.Unix()}[0]
	workspaceUser.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	workspaceUser.Active = true

	// Initialize workspace_user_roles slice if nil
	if workspaceUser.WorkspaceUserRoles == nil {
		workspaceUser.WorkspaceUserRoles = []*workspaceuserrolepb.WorkspaceUserRole{}
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkspaceUserUseCase) validateBusinessRules(ctx context.Context, workspaceUser *workspaceuserpb.WorkspaceUser) error {
	// Validate workspace and user relationship
	if workspaceUser.WorkspaceId == workspaceUser.UserId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.same_id", "Workspace ID and user ID cannot be the same [DEFAULT]"))
	}

	// Validate workspace_user_roles if provided
	if len(workspaceUser.WorkspaceUserRoles) > 0 {
		for _, userRole := range workspaceUser.WorkspaceUserRoles {
			if userRole == nil {
				return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.null_role", "Workspace user roles cannot contain null entries [DEFAULT]"))
			}
			// Validate the role within the userRole object
			if userRole.Role != nil && strings.TrimSpace(userRole.Role.Name) == "" {
				return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.empty_role", "Roles cannot contain empty strings [DEFAULT]"))
			}
		}

		// Check for duplicate roles
		roleSet := make(map[string]bool)
		for _, userRole := range workspaceUser.WorkspaceUserRoles {
			if userRole != nil && userRole.Role != nil {
				roleName := userRole.Role.Name
				if roleSet[roleName] {
					return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.validation.duplicate_roles", "Duplicate roles are not allowed [DEFAULT]"))
				}
				roleSet[roleName] = true
			}
		}
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateWorkspaceUserUseCase) validateEntityReferences(ctx context.Context, workspaceUser *workspaceuserpb.WorkspaceUser) error {
	// Validate Workspace entity reference
	if workspaceUser.WorkspaceId != "" {
		workspace, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{
			Data: &workspacepb.Workspace{Id: workspaceUser.WorkspaceId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.workspace_reference_validation_failed", "Failed to validate workspace entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if workspace == nil || workspace.Data == nil || len(workspace.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.workspace_not_found", "Referenced workspace with ID '{workspaceId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceId}", workspaceUser.WorkspaceId)
			return errors.New(translatedError)
		}
		if !workspace.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.workspace_not_active", "Referenced workspace with ID '{workspaceId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceId}", workspaceUser.WorkspaceId)
			return errors.New(translatedError)
		}
	}

	// Validate User entity reference
	if workspaceUser.UserId != "" {
		user, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{
			Data: &userpb.User{Id: workspaceUser.UserId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.user_reference_validation_failed", "Failed to validate user entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if user == nil || user.Data == nil || len(user.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.user_not_found", "Referenced user with ID '{userId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{userId}", workspaceUser.UserId)
			return errors.New(translatedError)
		}
		if !user.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user.errors.user_not_active", "Referenced user with ID '{userId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{userId}", workspaceUser.UserId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions

// Additional validation methods can be added here as needed
