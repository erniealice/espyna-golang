package depreciation_run

// Tests for ListDepreciationCandidatesUseCase.
//
// Phase 1.6 (2026-05-10) — codex C1.5 mismatch rejection:
//   - A caller authenticated as ws-A must not be able to submit
//     req.workspace_id=ws-B and receive candidates scoped to ws-B.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// fakeCandidateAssetRepo is a minimal AssetDomainServiceServer for candidate tests.
type fakeCandidateAssetRepo struct {
	assetpb.UnimplementedAssetDomainServiceServer
}

func (r *fakeCandidateAssetRepo) ReadAsset(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
	return &assetpb.ReadAssetResponse{}, nil
}

func (r *fakeCandidateAssetRepo) ListAssets(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
	return &assetpb.ListAssetsResponse{}, nil
}

// fakeCandidateAssetCategoryRepo is a minimal AssetCategoryDomainServiceServer stub.
type fakeCandidateAssetCategoryRepo struct {
	assetcategorypb.UnimplementedAssetCategoryDomainServiceServer
}

// fakeCandidateDepreciationScheduleRepo is a minimal DepreciationDomainServiceServer stub.
type fakeCandidateDepreciationScheduleRepo struct {
	depschpb.UnimplementedDepreciationDomainServiceServer
}

func newListCandidatesUseCase() *ListDepreciationCandidatesUseCase {
	return NewListDepreciationCandidatesUseCase(
		ListDepreciationCandidatesRepositories{
			Asset:                &fakeCandidateAssetRepo{},
			AssetCategory:        &fakeCandidateAssetCategoryRepo{},
			DepreciationSchedule: &fakeCandidateDepreciationScheduleRepo{},
		},
		ListDepreciationCandidatesServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TranslationService:   ports.NewNoOpTranslationService(),
		},
	)
}

// ---------------------------------------------------------------------------
// Phase 1.6 — mismatch rejection test (codex C1.5)
// ---------------------------------------------------------------------------

// TestListCandidates_TenancyMismatch_RejectsBeforeRead verifies that when the
// authenticated context carries workspace "ws-A" and the request claims
// workspace "ws-B", Execute returns a tenancy-mismatch error and performs
// no asset reads for the mismatched workspace.
func TestListCandidates_TenancyMismatch_RejectsBeforeRead(t *testing.T) {
	uc := newListCandidatesUseCase()

	// Context is authenticated as ws-A; request claims ws-B.
	ctx := contextutil.WithWorkspaceID(context.Background(), "ws-A")
	req := &deprunpb.ListDepreciationCandidatesRequest{
		WorkspaceId: "ws-B",
		ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE,
		AsOfDate:    "2026-01-15",
	}

	res, err := uc.Execute(ctx, req)
	if err == nil {
		t.Fatalf("expected tenancy-mismatch error, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result on tenancy-mismatch reject, got %+v", res)
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected error to mention workspace, got: %q", err.Error())
	}
}
