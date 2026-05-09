package depreciation_run

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// ListDepreciationRunsRepositories groups repository dependencies.
type ListDepreciationRunsRepositories struct {
	DepreciationRun deprunpb.DepreciationRunDomainServiceServer
}

// ListDepreciationRunsServices groups service dependencies.
type ListDepreciationRunsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListDepreciationRunsUseCase lists depreciation run history rows.
type ListDepreciationRunsUseCase struct {
	repositories ListDepreciationRunsRepositories
	services     ListDepreciationRunsServices
}

// NewListDepreciationRunsUseCase wires the use case.
func NewListDepreciationRunsUseCase(
	repositories ListDepreciationRunsRepositories,
	services ListDepreciationRunsServices,
) *ListDepreciationRunsUseCase {
	return &ListDepreciationRunsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute returns paginated depreciation runs for the workspace.
func (uc *ListDepreciationRunsUseCase) Execute(
	ctx context.Context,
	req *deprunpb.ListDepreciationRunsRequest,
) (*deprunpb.ListDepreciationRunsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetDepreciationRun, ports.ActionRead); err != nil {
		return nil, err
	}
	if uc.repositories.DepreciationRun == nil {
		return &deprunpb.ListDepreciationRunsResponse{Success: true}, nil
	}
	return uc.repositories.DepreciationRun.ListDepreciationRuns(ctx, req)
}
