package work_request

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// GetWorkRequestListPageDataUseCase wraps the paginated list page data
// for the work request inbox.
//
// Returns count-by-status for status tabs and KPI summary row.
// The IDOR workspace_id scoping is enforced in the adapter's query predicate.
type GetWorkRequestListPageDataUseCase struct {
	repositories ListWorkRequestsRepositories
	services     ListWorkRequestsServices
}

func NewGetWorkRequestListPageDataUseCase(repositories ListWorkRequestsRepositories, services ListWorkRequestsServices) *GetWorkRequestListPageDataUseCase {
	return &GetWorkRequestListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetWorkRequestListPageDataUseCase) Execute(ctx context.Context, req *work_requestpb.GetWorkRequestListPageDataRequest) (*work_requestpb.GetWorkRequestListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.WorkRequest.GetWorkRequestListPageData(ctx, req)
}
