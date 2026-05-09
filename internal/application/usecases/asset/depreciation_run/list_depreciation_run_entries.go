package depreciation_run

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListDepreciationRunEntriesRepositories groups repository dependencies.
type ListDepreciationRunEntriesRepositories struct {
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
}

// ListDepreciationRunEntriesServices groups service dependencies.
type ListDepreciationRunEntriesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	runID string,
	pagination *commonpb.PaginationRequest,
) (*depschpb.ListDepreciationSchedulesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetDepreciationRun, ports.ActionRead); err != nil {
		return nil, err
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
