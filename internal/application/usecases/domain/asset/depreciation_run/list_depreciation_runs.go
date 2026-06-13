package depreciation_run

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"

	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// ListDepreciationRunsRepositories groups repository dependencies.
type ListDepreciationRunsRepositories struct {
	DepreciationRun deprunpb.DepreciationRunDomainServiceServer
}

// ListDepreciationRunsServices groups service dependencies.
type ListDepreciationRunsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAssetDepreciationRun,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if uc.repositories.DepreciationRun == nil {
		return &deprunpb.ListDepreciationRunsResponse{Success: true}, nil
	}
	return uc.repositories.DepreciationRun.ListDepreciationRuns(ctx, req)
}
