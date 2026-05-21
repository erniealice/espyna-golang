// Package asset_revaluation — PreviewRevaluation use case (read-only).
//
// Returns the IAS 16.39-40 PnL/OCI split that would apply if the supplied
// new_fair_value were posted right now, without writing to the database.
//
// The preview shares the surplus-state derivation algorithm with
// RevalueAsset (deriveSurplusState + ComputePnLOCISplit) so the preview is
// faithful to the actual posting logic. NOTE: the preview is non-locking —
// a concurrent revaluation that lands between preview and submit can shift
// the actual split slightly. This is a documented, accepted race; the
// drawer is informational, and the authoritative split is computed inside
// RevalueAsset's tx with row-locked history.
package asset_revaluation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
)

// PreviewRevaluationRequest is the internal input to PreviewRevaluation.
// Kept for internal helpers. The public boundary uses *revaluation_pb.PreviewRevaluationUseCaseRequest.
type PreviewRevaluationRequest struct {
	AssetID      string
	NewFairValue int64 // centavos
}

// PreviewRevaluationResult is the internal predicted IAS 16.39-40 split.
// Kept for internal helpers. The public boundary returns *revaluation_pb.PreviewRevaluationUseCaseResponse.
type PreviewRevaluationResult struct {
	PreviousCarryingAmount int64
	NewFairValue           int64
	RevaluationAmount      int64 // signed: positive = up, negative = down
	IsIncrease             bool
	RecognizedInPnL        int64 // signed (positive=gain reversal, negative=loss)
	RecognizedInOCI        int64 // signed (positive=surplus credit, negative=surplus debit)
	NewSurplusBalance      int64 // running surplus after the predicted entry
	PriorSurplusBalance    int64
	PriorPnLLossBalance    int64
}

// PreviewRevaluationUseCase computes the predicted PnL/OCI split for a pending
// revaluation without persisting any state.
type PreviewRevaluationUseCase struct {
	repositories RevalueAssetRepositories
	services     RevalueAssetServices
}

// NewPreviewRevaluationUseCase wires the use case.
func NewPreviewRevaluationUseCase(
	repositories RevalueAssetRepositories,
	services RevalueAssetServices,
) *PreviewRevaluationUseCase {
	return &PreviewRevaluationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute returns the predicted IAS 16.39-40 split for new_fair_value against
// the asset's current book value and the immutable AssetRevaluation history.
// Read-only; no transaction needed.
func (uc *PreviewRevaluationUseCase) Execute(
	ctx context.Context,
	pbReq *revaluation_pb.PreviewRevaluationUseCaseRequest,
) (*revaluation_pb.PreviewRevaluationUseCaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetRevaluation, ports.ActionRead); err != nil {
		return nil, err
	}

	// Translate proto → internal Go struct at the boundary.
	req := PreviewRevaluationRequest{}
	if pbReq != nil {
		req.AssetID = pbReq.GetAssetId()
		req.NewFairValue = pbReq.GetNewFairValue()
	}

	if req.AssetID == "" {
		return nil, errors.New("preview_revaluation: asset_id is required")
	}
	if req.NewFairValue <= 0 {
		return nil, errors.New("preview_revaluation: new_fair_value must be > 0")
	}

	// Workspace tenancy check (mirrors RevalueAsset.Execute, codex C2).
	workspaceID := strings.TrimSpace(contextutil.ExtractWorkspaceIDFromContext(ctx))
	if workspaceID == "" {
		// TODO: translate via TranslationService (Fix #4 deferred — codex L1).
		// Suggested key: asset.assetDetail.depreciationRun.errors.workspaceRequired
		return nil, errors.New("preview_revaluation: workspace_id is required (codex C2 — workspace tenancy)")
	}

	// Read the asset (snapshot read; no row lock for preview).
	asset, err := uc.readAsset(ctx, req.AssetID)
	if err != nil || asset == nil {
		return nil, fmt.Errorf("preview_revaluation: asset %q not found: %w", req.AssetID, err)
	}
	if got := strings.TrimSpace(asset.GetWorkspaceId()); got != workspaceID {
		return nil, fmt.Errorf("preview_revaluation: asset %q does not belong to workspace %q", req.AssetID, workspaceID)
	}

	// Measurement-model gate (codex H4): COST-model assets cannot be revalued.
	if asset.GetMeasurementModel() != assetpb.MeasurementModel_MEASUREMENT_MODEL_REVALUATION {
		// TODO: translate via TranslationService.
		// Suggested key: asset.assetRevaluation.errors.wrongMeasurementModel
		return nil, errors.New("preview_revaluation: asset measurement_model must be REVALUATION to be revalued")
	}

	currentBookValue := asset.GetBookValue()
	revaluationAmount := req.NewFairValue - currentBookValue
	if revaluationAmount == 0 {
		// Caller can render this as a "no change" preview.
		return &revaluation_pb.PreviewRevaluationUseCaseResponse{
			Success:                true,
			PreviousCarryingAmount: currentBookValue,
			NewFairValue:           req.NewFairValue,
			RevaluationAmount:      0,
			IsIncrease:             false,
		}, nil
	}
	isIncrease := revaluationAmount > 0
	absAmount := revaluationAmount
	if absAmount < 0 {
		absAmount = -absAmount
	}

	// Walk the immutable AssetRevaluation history (Option A — derive surplus
	// state from history, no per-asset balance fields on Asset proto).
	priorSurplus, priorPnLLoss, err := deriveSurplusStateFromHistory(ctx, uc.repositories.AssetRevaluation, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("preview_revaluation: surplus state derivation failed: %w", err)
	}

	pnl, oci, newSurplus := ComputePnLOCISplit(absAmount, isIncrease, priorSurplus, priorPnLLoss)

	return &revaluation_pb.PreviewRevaluationUseCaseResponse{
		Success:                true,
		PreviousCarryingAmount: currentBookValue,
		NewFairValue:           req.NewFairValue,
		RevaluationAmount:      revaluationAmount,
		IsIncrease:             isIncrease,
		RecognizedInPnl:        pnl,
		RecognizedInOci:        oci,
		NewSurplusBalance:      newSurplus,
		PriorSurplusBalance:    priorSurplus,
		PriorPnlLossBalance:    priorPnLLoss,
	}, nil
}

// readAsset fetches a single asset (snapshot read).
func (uc *PreviewRevaluationUseCase) readAsset(ctx context.Context, assetID string) (*assetpb.Asset, error) {
	if uc.repositories.Asset == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Asset.ReadAsset(ctx, &assetpb.ReadAssetRequest{
		Data: &assetpb.Asset{Id: assetID},
	})
	if err != nil || resp == nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

// AssetRevaluationRepo exposes the underlying AssetRevaluation repository so
// the consumer-layer pass-through can drill in. Mirrors RevalueAssetUseCase
// for symmetry — preview and revalue share a repo.
func (uc *PreviewRevaluationUseCase) AssetRevaluationRepo() revaluation_pb.AssetRevaluationDomainServiceServer {
	if uc == nil {
		return nil
	}
	return uc.repositories.AssetRevaluation
}
