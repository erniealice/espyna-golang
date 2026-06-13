package work_request

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// GetWorkRequestItemPageDataUseCase wraps the item page data for the work
// request detail view.
//
// Returns the entity joined with its type and assignee info for the detail page.
// The IDOR workspace_id + origin-aware client scoping is enforced in the
// adapter's query predicate.
type GetWorkRequestItemPageDataUseCase struct {
	repositories ListWorkRequestsRepositories
	services     ListWorkRequestsServices
}

func NewGetWorkRequestItemPageDataUseCase(repositories ListWorkRequestsRepositories, services ListWorkRequestsServices) *GetWorkRequestItemPageDataUseCase {
	return &GetWorkRequestItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetWorkRequestItemPageDataUseCase) Execute(ctx context.Context, req *work_requestpb.GetWorkRequestItemPageDataRequest) (*work_requestpb.GetWorkRequestItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.WorkRequest.GetWorkRequestItemPageData(ctx, req)
}
