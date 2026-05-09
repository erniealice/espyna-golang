package consumer

import (
	"context"

	assetrevuc "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_revaluation"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
)

// Re-export types so view packages can use them without importing espyna internals.

// RevalueAssetRequest is the public input type for RevalueAsset.
type RevalueAssetRequest = assetrevuc.RevalueAssetRequest

// RevalueAssetResult is the public output type for RevalueAsset.
type RevalueAssetResult = assetrevuc.RevalueAssetResult

// RevalueAsset executes the IAS 16 revaluation for a single asset.
// Inserts an AssetRevaluation record + AssetTransaction back-ref + updates Asset.
func RevalueAsset(
	useCases *UseCases,
	ctx context.Context,
	req RevalueAssetRequest,
) (*RevalueAssetResult, error) {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.AssetRevaluation == nil {
		return nil, nil
	}
	uc := useCases.Asset.AssetRevaluation.RevalueAsset
	if uc == nil {
		return nil, nil
	}
	return uc.Execute(ctx, req)
}

// ListAssetRevaluations returns the revaluation history for an asset.
// Pass-through to the AssetRevaluationDomainService repo.
func ListAssetRevaluations(
	useCases *UseCases,
	ctx context.Context,
	req *revaluation_pb.ListAssetRevaluationsRequest,
) (*revaluation_pb.ListAssetRevaluationsResponse, error) {
	repo := assetRevaluationRepo(useCases)
	if repo == nil {
		return &revaluation_pb.ListAssetRevaluationsResponse{Success: true}, nil
	}
	return repo.ListAssetRevaluations(ctx, req)
}

// ReadAssetRevaluation reads a single AssetRevaluation row by ID.
func ReadAssetRevaluation(
	useCases *UseCases,
	ctx context.Context,
	req *revaluation_pb.ReadAssetRevaluationRequest,
) (*revaluation_pb.ReadAssetRevaluationResponse, error) {
	repo := assetRevaluationRepo(useCases)
	if repo == nil {
		return &revaluation_pb.ReadAssetRevaluationResponse{}, nil
	}
	return repo.ReadAssetRevaluation(ctx, req)
}

// assetRevaluationRepo is a nil-safe helper that drills into the use-case aggregate
// to retrieve the AssetRevaluation repository for proto pass-through calls.
func assetRevaluationRepo(useCases *UseCases) revaluation_pb.AssetRevaluationDomainServiceServer {
	if useCases == nil || useCases.Asset == nil || useCases.Asset.AssetRevaluation == nil {
		return nil
	}
	uc := useCases.Asset.AssetRevaluation.RevalueAsset
	if uc == nil {
		return nil
	}
	return uc.AssetRevaluationRepo()
}
