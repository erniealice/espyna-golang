//go:build mock_db && mock_auth

package role_permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// GetRolePermissionItemPageDataRepositories groups all repository dependencies
type GetRolePermissionItemPageDataRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
}

// GetRolePermissionItemPageDataServices groups all business service dependencies
type GetRolePermissionItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRolePermissionItemPageDataUseCase handles the business logic for retrieving role permission item page data
type GetRolePermissionItemPageDataUseCase struct {
	repositories GetRolePermissionItemPageDataRepositories
	services     GetRolePermissionItemPageDataServices
}

// NewGetRolePermissionItemPageDataUseCase creates use case with grouped dependencies
func NewGetRolePermissionItemPageDataUseCase(
	repositories GetRolePermissionItemPageDataRepositories,
	services GetRolePermissionItemPageDataServices,
) *GetRolePermissionItemPageDataUseCase {
	return &GetRolePermissionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetRolePermissionItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetRolePermissionItemPageDataUseCase with grouped parameters instead
func NewGetRolePermissionItemPageDataUseCaseUngrouped(rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer) *GetRolePermissionItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetRolePermissionItemPageDataRepositories{
		RolePermission: rolePermissionRepo,
	}

	services := GetRolePermissionItemPageDataServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewGetRolePermissionItemPageDataUseCase(repositories, services)
}

// Execute performs the get role permission item page data operation
func (uc *GetRolePermissionItemPageDataUseCase) Execute(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.RolePermission,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role permission item page data retrieval within a transaction
func (uc *GetRolePermissionItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	var result *rolepermissionpb.GetRolePermissionItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "role_permission.errors.item_page_data_retrieval_failed", "Role permission item page data retrieval failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *GetRolePermissionItemPageDataUseCase) executeCore(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.RolePermission.GetRolePermissionItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetRolePermissionItemPageDataUseCase) validateInput(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.request_required", "Request is required for role permission item page data [DEFAULT]"))
	}

	if req.RolePermissionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.role_permission_id_required", "Role permission ID is required [DEFAULT]"))
	}

	return nil
}
