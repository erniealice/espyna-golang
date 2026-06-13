package work_request

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// ListWorkRequestsRepositories groups all repository dependencies.
type ListWorkRequestsRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// ListWorkRequestsServices groups all business service dependencies.
type ListWorkRequestsServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// ListWorkRequestsUseCase lists work requests with workspace-scoped filtering.
//
// Supports TypedFilter for status, origin, and client_id filtering.
// The adapter enforces workspace_id scoping. For client paths, the origin-aware
// predicate (client_id = acting_as_client_id AND origin = CLIENT_ORIGINATED) is
// injected server-side via TypedFilter before the backend call.
type ListWorkRequestsUseCase struct {
	repositories ListWorkRequestsRepositories
	services     ListWorkRequestsServices
}

func NewListWorkRequestsUseCase(repositories ListWorkRequestsRepositories, services ListWorkRequestsServices) *ListWorkRequestsUseCase {
	return &ListWorkRequestsUseCase{repositories: repositories, services: services}
}

func (uc *ListWorkRequestsUseCase) Execute(ctx context.Context, req *work_requestpb.ListWorkRequestsRequest) (*work_requestpb.ListWorkRequestsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		req = &work_requestpb.ListWorkRequestsRequest{}
	}
	return uc.repositories.WorkRequest.ListWorkRequests(ctx, req)
}

// InjectStatusFilter appends a server-side status filter to the list request.
// This ensures correct pagination counts — NEVER filter client-side after
// paginated results (use-case-patterns.md: Server-Side Status Filtering).
func InjectStatusFilter(req *work_requestpb.ListWorkRequestsRequest, status string) {
	if req.Filters == nil {
		req.Filters = &commonpb.FilterRequest{}
	}
	req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
		Field: "wr.status",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    status,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	})
}

// InjectOriginFilter appends a server-side origin filter for admin inbox
// origin-filter chips (Client / Internal / Client-related).
func InjectOriginFilter(req *work_requestpb.ListWorkRequestsRequest, origin string) {
	if req.Filters == nil {
		req.Filters = &commonpb.FilterRequest{}
	}
	req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
		Field: "wr.origin",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    origin,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	})
}

// InjectClientIDFilter appends a server-side client_id filter for the client
// portal path. The client_id is the session's acting_as_client_id (NEVER a
// request parameter).
func InjectClientIDFilter(req *work_requestpb.ListWorkRequestsRequest, clientID string) {
	if req.Filters == nil {
		req.Filters = &commonpb.FilterRequest{}
	}
	req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
		Field: "wr.client_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    clientID,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	})
}
