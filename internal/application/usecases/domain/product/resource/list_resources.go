package resource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// ListResourcesUseCase handles the business logic for listing resources
// ListResourcesRepositories groups all repository dependencies
type ListResourcesRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
}

// ListResourcesServices groups all business service dependencies
type ListResourcesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListResourcesUseCase handles the business logic for listing resources
type ListResourcesUseCase struct {
	repositories ListResourcesRepositories
	services     ListResourcesServices
}

// NewListResourcesUseCase creates a new ListResourcesUseCase
func NewListResourcesUseCase(
	repositories ListResourcesRepositories,
	services ListResourcesServices,
) *ListResourcesUseCase {
	return &ListResourcesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list resources operation
func (uc *ListResourcesUseCase) Execute(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Resource,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Resource.ListResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListResourcesUseCase) validateInput(ctx context.Context, req *resourcepb.ListResourcesRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
