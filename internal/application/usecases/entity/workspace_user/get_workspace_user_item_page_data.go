package workspace_user

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

type GetWorkspaceUserItemPageDataRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer
}

type GetWorkspaceUserItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetWorkspaceUserItemPageDataUseCase handles the business logic for getting workspace user item page data
type GetWorkspaceUserItemPageDataUseCase struct {
	repositories GetWorkspaceUserItemPageDataRepositories
	services     GetWorkspaceUserItemPageDataServices
}

// NewGetWorkspaceUserItemPageDataUseCase creates a new GetWorkspaceUserItemPageDataUseCase
func NewGetWorkspaceUserItemPageDataUseCase(
	repositories GetWorkspaceUserItemPageDataRepositories,
	services GetWorkspaceUserItemPageDataServices,
) *GetWorkspaceUserItemPageDataUseCase {
	return &GetWorkspaceUserItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get workspace user item page data operation
func (uc *GetWorkspaceUserItemPageDataUseCase) Execute(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityWorkspaceUser, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.WorkspaceUserId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workspace user item page data retrieval within a transaction
func (uc *GetWorkspaceUserItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	var result *workspaceuserpb.GetWorkspaceUserItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workspace_user.errors.item_page_data_failed",
				"workspace user item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting workspace user item page data.
// Delegates to the repository's GetWorkspaceUserItemPageData which uses a CTE query
// to load the workspace_user with its workspace_user_roles and nested role data.
func (uc *GetWorkspaceUserItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	return uc.repositories.WorkspaceUser.GetWorkspaceUserItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetWorkspaceUserItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user.validation.request_required",
			"request is required",
		))
	}

	if req.WorkspaceUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user.validation.id_required",
			"workspace user ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading workspace user item page data
func (uc *GetWorkspaceUserItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	workspaceUserId string,
) error {
	// Validate workspace user ID format
	if len(workspaceUserId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workspace_user.validation.id_too_short",
			"workspace user ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this workspace user
	// - Validate workspace user belongs to the current user's organization
	// - Check if workspace user is in a state that allows viewing
	// - Rate limiting for workspace user access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like workspace and user details
// This would be called from executeCore if needed
func (uc *GetWorkspaceUserItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	workspaceUser *workspaceuserpb.WorkspaceUser,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to workspace and user repositories
	// to populate the nested workspace and user objects if they're not already loaded

	// Example implementation would be:
	// if workspaceUser.Workspace == nil && workspaceUser.WorkspaceId != "" {
	//     // Load workspace data
	// }
	// if workspaceUser.User == nil && workspaceUser.UserId != "" {
	//     // Load user data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetWorkspaceUserItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	workspaceUser *workspaceuserpb.WorkspaceUser,
) *workspaceuserpb.WorkspaceUser {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return workspaceUser
}

// checkAccessPermissions validates user has permission to access this workspace user
func (uc *GetWorkspaceUserItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	workspaceUserId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating workspace user belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
