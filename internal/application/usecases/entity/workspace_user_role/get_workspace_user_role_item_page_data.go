package workspace_user_role

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

type GetWorkspaceUserRoleItemPageDataRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
}

type GetWorkspaceUserRoleItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetWorkspaceUserRoleItemPageDataUseCase handles the business logic for getting workspace user role item page data
type GetWorkspaceUserRoleItemPageDataUseCase struct {
	repositories GetWorkspaceUserRoleItemPageDataRepositories
	services     GetWorkspaceUserRoleItemPageDataServices
}

// NewGetWorkspaceUserRoleItemPageDataUseCase creates a new GetWorkspaceUserRoleItemPageDataUseCase
func NewGetWorkspaceUserRoleItemPageDataUseCase(
	repositories GetWorkspaceUserRoleItemPageDataRepositories,
	services GetWorkspaceUserRoleItemPageDataServices,
) *GetWorkspaceUserRoleItemPageDataUseCase {
	return &GetWorkspaceUserRoleItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get workspace user role item page data operation
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) Execute(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest,
) (*workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspaceUserRole, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.WorkspaceUserRoleId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workspace user role item page data retrieval within a transaction
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest,
) (*workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse, error) {
	var result *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workspace_user_role.errors.item_page_data_failed",
				"workspace user role item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting workspace user role item page data
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest,
) (*workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse, error) {
	// Create read request for the workspace user role
	readReq := &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{
			Id: req.WorkspaceUserRoleId,
		},
	}

	// Retrieve the workspace user role
	readResp, err := uc.repositories.WorkspaceUserRole.ReadWorkspaceUserRole(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.errors.read_failed",
			"failed to retrieve workspace user role: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.errors.not_found",
			"workspace user role not found",
		))
	}

	// Get the workspace user role (should be only one)
	workspaceUserRole := readResp.Data[0]

	// Validate that we got the expected workspace user role
	if workspaceUserRole.Id != req.WorkspaceUserRoleId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.errors.id_mismatch",
			"retrieved workspace user role ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (workspace user details, role details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the workspace user role as-is
	return &workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse{
		WorkspaceUserRole: workspaceUserRole,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.validation.request_required",
			"request is required",
		))
	}

	if req.WorkspaceUserRoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.validation.id_required",
			"workspace user role ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading workspace user role item page data
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	workspaceUserRoleId string,
) error {
	// Validate workspace user role ID format
	if len(workspaceUserRoleId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user_role.validation.id_too_short",
			"workspace user role ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this workspace user role
	// - Validate workspace user role belongs to the current user's organization
	// - Check if workspace user role is in a state that allows viewing
	// - Rate limiting for workspace user role access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like workspace user and role details
// This would be called from executeCore if needed
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to workspace user and role repositories
	// to populate the nested workspace user and role objects if they're not already loaded

	// Example implementation would be:
	// if workspaceUserRole.WorkspaceUser == nil && workspaceUserRole.WorkspaceUserId != "" {
	//     // Load workspace user data
	// }
	// if workspaceUserRole.Role == nil && workspaceUserRole.RoleId != "" {
	//     // Load role data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole,
) *workspaceuserrolepb.WorkspaceUserRole {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return workspaceUserRole
}

// checkAccessPermissions validates user has permission to access this workspace user role
func (uc *GetWorkspaceUserRoleItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	workspaceUserRoleId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating workspace user role belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
