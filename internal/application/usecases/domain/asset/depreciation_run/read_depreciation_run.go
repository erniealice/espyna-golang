package depreciation_run

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"

	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// ReadDepreciationRunRepositories groups repository dependencies.
type ReadDepreciationRunRepositories struct {
	DepreciationRun deprunpb.DepreciationRunDomainServiceServer
}

// ReadDepreciationRunServices groups service dependencies.
type ReadDepreciationRunServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadDepreciationRunUseCase reads a single depreciation run by ID.
type ReadDepreciationRunUseCase struct {
	repositories ReadDepreciationRunRepositories
	services     ReadDepreciationRunServices
}

// NewReadDepreciationRunUseCase wires the use case.
func NewReadDepreciationRunUseCase(
	repositories ReadDepreciationRunRepositories,
	services ReadDepreciationRunServices,
) *ReadDepreciationRunUseCase {
	return &ReadDepreciationRunUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute reads a depreciation run by ID.
func (uc *ReadDepreciationRunUseCase) Execute(
	ctx context.Context,
	req *deprunpb.ReadDepreciationRunRequest,
) (*deprunpb.ReadDepreciationRunResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAssetDepreciationRun, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repositories.DepreciationRun == nil {
		return &deprunpb.ReadDepreciationRunResponse{}, nil
	}
	return uc.repositories.DepreciationRun.ReadDepreciationRun(ctx, req)
}
