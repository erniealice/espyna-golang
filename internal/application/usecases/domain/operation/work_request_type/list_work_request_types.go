package work_request_type

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// ListWorkRequestTypesRepositories groups all repository dependencies
type ListWorkRequestTypesRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// ListWorkRequestTypesServices groups all business service dependencies
type ListWorkRequestTypesServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListWorkRequestTypesUseCase handles the business logic for listing work request types
type ListWorkRequestTypesUseCase struct {
	repositories ListWorkRequestTypesRepositories
	services     ListWorkRequestTypesServices
}

// NewListWorkRequestTypesUseCase creates use case with grouped dependencies
func NewListWorkRequestTypesUseCase(
	repositories ListWorkRequestTypesRepositories,
	services ListWorkRequestTypesServices,
) *ListWorkRequestTypesUseCase {
	return &ListWorkRequestTypesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list work request types operation with optional status filter
func (uc *ListWorkRequestTypesUseCase) Execute(ctx context.Context, req *work_request_typepb.ListWorkRequestTypesRequest, status string) (*work_request_typepb.ListWorkRequestTypesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequestType,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Inject status filter server-side if provided (never filter client-side after pagination)
	if status != "" {
		if req.Filters == nil {
			req.Filters = &commonpb.FilterRequest{}
		}
		req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
			Field: "wrt.status",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:    status,
					Operator: commonpb.StringOperator_STRING_EQUALS,
				},
			},
		})
	}

	// Call repository
	resp, err := uc.repositories.WorkRequestType.ListWorkRequestTypes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.list_failed", "[ERR-DEFAULT] Failed to list work request types")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListWorkRequestTypesUseCase) validateInput(ctx context.Context, req *work_request_typepb.ListWorkRequestTypesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}
