package depreciation_run

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"

	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListDepreciationRunEntriesRepositories groups repository dependencies.
type ListDepreciationRunEntriesRepositories struct {
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
}

// ListDepreciationRunEntriesServices groups service dependencies.
type ListDepreciationRunEntriesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListDepreciationRunEntriesUseCase lists DepreciationSchedule rows by depreciation_run_id.
// This is the "entries" sub-list on the Surface D run detail page.
type ListDepreciationRunEntriesUseCase struct {
	repositories ListDepreciationRunEntriesRepositories
	services     ListDepreciationRunEntriesServices
}

// NewListDepreciationRunEntriesUseCase wires the use case.
func NewListDepreciationRunEntriesUseCase(
	repositories ListDepreciationRunEntriesRepositories,
	services ListDepreciationRunEntriesServices,
) *ListDepreciationRunEntriesUseCase {
	return &ListDepreciationRunEntriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute returns all DepreciationSchedule rows for a given depreciation_run_id.
func (uc *ListDepreciationRunEntriesUseCase) Execute(
	ctx context.Context,
	req *deprunpb.ListDepreciationRunEntriesRequest,
) (*depschpb.ListDepreciationSchedulesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAssetDepreciationRun,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	runID := ""
	var pagination *commonpb.PaginationRequest
	if req != nil {
		runID = req.GetRunId()
		pagination = req.GetPagination()
	}
	if runID == "" {
		return nil, errors.New("list_depreciation_run_entries: run_id is required")
	}
	if uc.repositories.DepreciationSchedule == nil {
		return &depschpb.ListDepreciationSchedulesResponse{Success: true}, nil
	}
	return uc.repositories.DepreciationSchedule.ListDepreciationSchedules(ctx, &depschpb.ListDepreciationSchedulesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "depreciation_run_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    runID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
		Pagination: pagination,
	})
}
