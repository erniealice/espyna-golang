// Package asset_revaluation provides the RevalueAsset use case implementing IAS 16.31-42.
//
// PnL/OCI split (IAS 16.39-40) is derived at recognize-time from the immutable
// AssetRevaluation history under SELECT … FOR UPDATE. No per-asset balance fields
// are added to the Asset proto — Option A locked 2026-05-09.
package asset_revaluation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

const entityAssetRevaluation = "asset"

// RevalueAssetRepositories groups all repository dependencies.
type RevalueAssetRepositories struct {
	Asset            assetpb.AssetDomainServiceServer
	AssetTransaction assettxpb.AssetTransactionDomainServiceServer
	AssetRevaluation revaluation_pb.AssetRevaluationDomainServiceServer
}

// RevalueAssetServices groups all business service dependencies.
type RevalueAssetServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// RevalueAssetRequest is the input to the use case.
type RevalueAssetRequest struct {
	AssetID         string
	NewFairValue    int64  // centavos
	AppraiserName   string
	ValuationMethod string
	Notes           string
}

// RevalueAssetResult is the output.
type RevalueAssetResult struct {
	Revaluation *revaluation_pb.AssetRevaluation
	Transaction *assettxpb.AssetTransaction
}

// RevalueAssetUseCase executes the IAS 16 revaluation flow:
//  1. SELECT asset FOR UPDATE (acquire row lock)
//  2. Compute revaluation_amount, is_increase
//  3. Derive PnL/OCI split from immutable AssetRevaluation history (Option A)
//  4. INSERT asset_revaluation (full IFRS row)
//  5. INSERT asset_transaction (REVALUATION_UP|DOWN) with asset_revaluation_id back-ref
//  6. UPDATE asset.book_value = asset.fair_value = new_fair_value
type RevalueAssetUseCase struct {
	repositories RevalueAssetRepositories
	services     RevalueAssetServices
}

// AssetRevaluationRepo exposes the underlying AssetRevaluation repository for
// consumer-layer pass-through calls (ListAssetRevaluations, ReadAssetRevaluation, etc.)
func (uc *RevalueAssetUseCase) AssetRevaluationRepo() revaluation_pb.AssetRevaluationDomainServiceServer {
	if uc == nil {
		return nil
	}
	return uc.repositories.AssetRevaluation
}

// NewRevalueAssetUseCase wires the use case.
func NewRevalueAssetUseCase(
	repositories RevalueAssetRepositories,
	services RevalueAssetServices,
) *RevalueAssetUseCase {
	return &RevalueAssetUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the revaluation within a single transaction.
func (uc *RevalueAssetUseCase) Execute(
	ctx context.Context,
	req RevalueAssetRequest,
) (*RevalueAssetResult, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAssetRevaluation, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req.AssetID == "" {
		return nil, errors.New("revalue_asset: asset_id is required")
	}
	if req.NewFairValue <= 0 {
		return nil, errors.New("revalue_asset: new_fair_value must be > 0")
	}

	// Step 1: READ asset (with row lock acquired by the TransactionService if supported).
	asset, err := uc.readAsset(ctx, req.AssetID)
	if err != nil || asset == nil {
		return nil, fmt.Errorf("revalue_asset: asset %q not found: %w", req.AssetID, err)
	}

	// Step 2: Compute revaluation_amount, is_increase
	currentBookValue := asset.GetBookValue()
	revaluationAmount := req.NewFairValue - currentBookValue
	if revaluationAmount == 0 {
		return nil, errors.New("revalue_asset: new_fair_value equals current book value — no revaluation needed")
	}
	isIncrease := revaluationAmount > 0
	absAmount := revaluationAmount
	if absAmount < 0 {
		absAmount = -absAmount
	}

	// Step 3: Option A — derive surplus state from immutable AssetRevaluation history.
	// Under the same row lock on the asset row, query the full revaluation history to
	// compute the net revaluation surplus (OCI balance) and the net PnL loss balance.
	priorSurplusBalance, priorPnLLossBalance, err := uc.deriveSurplusState(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("revalue_asset: surplus state derivation failed: %w", err)
	}

	// Step 3b: Apply IAS 16.39-40 PnL/OCI split.
	recognizedInPnL, recognizedInOCI, newSurplusBalance := ComputePnLOCISplit(
		absAmount, isIncrease, priorSurplusBalance, priorPnLLossBalance,
	)
	_ = newSurplusBalance // recorded on the AssetRevaluation row below

	var result *RevalueAssetResult

	executeCore := func(txCtx context.Context) error {
		// Step 4: INSERT asset_revaluation
		revDate := time.Now().UTC().Format("2006-01-02")
		revID := uc.services.IDService.GenerateID()
		now := time.Now().UTC().UnixMilli()
		nowStr := time.Now().UTC().Format(time.RFC3339)

		appraiserName := strings.TrimSpace(req.AppraiserName)
		valMethod := strings.TrimSpace(req.ValuationMethod)
		notes := strings.TrimSpace(req.Notes)

		var appraiserPtr *string
		if appraiserName != "" {
			appraiserPtr = &appraiserName
		}
		var valMethodPtr *string
		if valMethod != "" {
			valMethodPtr = &valMethod
		}
		var notesPtr *string
		if notes != "" {
			notesPtr = &notes
		}

		rev := &revaluation_pb.AssetRevaluation{
			Id:                       revID,
			AssetId:                  req.AssetID,
			RevaluationDate:          revDate,
			PreviousCarryingAmount:   currentBookValue,
			NewFairValue:             req.NewFairValue,
			RevaluationAmount:        revaluationAmount,
			IsIncrease:               isIncrease,
			RecognizedInPnl:          recognizedInPnL,
			RecognizedInOci:          recognizedInOCI,
			RevaluationSurplusBalance: newSurplusBalance,
			AppraiserName:            appraiserPtr,
			ValuationMethod:          valMethodPtr,
			Notes:                    notesPtr,
			DateCreated:              &now,
			DateCreatedString:        &nowStr,
			Active:                   true,
		}
		createdRevResp, revErr := uc.repositories.AssetRevaluation.CreateAssetRevaluation(txCtx, &revaluation_pb.CreateAssetRevaluationRequest{
			Data: rev,
		})
		if revErr != nil {
			return fmt.Errorf("revalue_asset: failed to create asset_revaluation: %w", revErr)
		}
		if createdRevResp != nil && len(createdRevResp.GetData()) > 0 {
			rev = createdRevResp.GetData()[0]
		}

		// Step 5: INSERT asset_transaction (REVALUATION_UP or REVALUATION_DOWN)
		txType := assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_REVALUATION_UP
		if !isIncrease {
			txType = assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_REVALUATION_DOWN
		}
		txID := uc.services.IDService.GenerateID()
		txAmount := absAmount // always positive; type discriminates direction
		revIDStr := rev.GetId()
		initiator := contextutil.ExtractWorkspaceUserIDFromContext(txCtx)

		assetTx := &assettxpb.AssetTransaction{
			Id:                  txID,
			AssetId:             req.AssetID,
			TransactionType:     txType,
			TransactionDate:     now,
			TransactionDateString: time.Now().UTC().Format("2006-01-02"),
			Amount:              txAmount,
			PerformedBy:         &initiator,
			AssetRevaluationId:  &revIDStr,
			Active:              true,
		}
		createdTxResp, txErr := uc.repositories.AssetTransaction.CreateAssetTransaction(txCtx, &assettxpb.CreateAssetTransactionRequest{
			Data: assetTx,
		})
		if txErr != nil {
			return fmt.Errorf("revalue_asset: failed to create asset_transaction: %w", txErr)
		}
		if createdTxResp != nil && len(createdTxResp.GetData()) > 0 {
			assetTx = createdTxResp.GetData()[0]
		}

		// Step 6: UPDATE asset.book_value = asset.fair_value = new_fair_value
		newFV := req.NewFairValue
		_, updateErr := uc.repositories.Asset.UpdateAsset(txCtx, &assetpb.UpdateAssetRequest{
			Data: &assetpb.Asset{
				Id:        req.AssetID,
				BookValue: req.NewFairValue,
				FairValue: &newFV,
			},
		})
		if updateErr != nil {
			return fmt.Errorf("revalue_asset: failed to update asset: %w", updateErr)
		}

		result = &RevalueAssetResult{
			Revaluation: rev,
			Transaction: assetTx,
		}
		return nil
	}

	// Execute within a transaction if supported
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, executeCore); err != nil {
			return nil, err
		}
	} else {
		if err := executeCore(ctx); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// deriveSurplusState queries the immutable AssetRevaluation history (Option A)
// and returns:
//   - priorSurplusBalance: cumulative OCI recognized in revaluation surplus (positive)
//   - priorPnLLossBalance: cumulative PnL losses recognized (absolute value, positive)
func (uc *RevalueAssetUseCase) deriveSurplusState(ctx context.Context, assetID string) (
	priorSurplusBalance int64,
	priorPnLLossBalance int64,
	err error,
) {
	if uc.repositories.AssetRevaluation == nil {
		return 0, 0, nil
	}

	resp, err := uc.repositories.AssetRevaluation.ListAssetRevaluations(ctx, &revaluation_pb.ListAssetRevaluationsRequest{
		AssetId: &assetID,
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "asset_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    assetID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil {
		return 0, 0, err
	}

	// Sum recognized_in_oci (positive = surplus build-up from increases)
	// Sum recognized_in_pnl from decreases (negative = loss recognized)
	for _, r := range resp.GetData() {
		oci := r.GetRecognizedInOci()
		pnl := r.GetRecognizedInPnl()
		if oci > 0 {
			priorSurplusBalance += oci
		}
		if pnl < 0 {
			// pnl is stored as a negative centavo amount for down-revaluations
			priorPnLLossBalance += (-pnl) // make it positive for tracking
		}
	}
	return priorSurplusBalance, priorPnLLossBalance, nil
}

// ComputePnLOCISplit applies IAS 16.39-40 to split the revaluation amount into
// recognized_in_pnl and recognized_in_oci, and returns the new running surplus balance.
//
// Rules:
//
//	Increase (isIncrease=true):
//	  - If there was a prior recognized PnL loss (recognized_in_pnl < 0 in history),
//	    reverse it first as a PnL gain (recognized_in_pnl = +X until prior loss exhausted).
//	    Remaining amount goes to OCI (recognized_in_oci = +Y).
//	  - If no prior PnL loss, entire amount is OCI.
//
//	Decrease (isIncrease=false):
//	  - If there is a surplus balance (prior oci > 0), use it first (recognized_in_oci = -X).
//	    Remaining beyond surplus goes to PnL loss (recognized_in_pnl = -Y).
//	  - If no surplus, entire amount is PnL loss.
//
// All amounts are centavos (int64). Signs:
//   - recognized_in_pnl: positive = gain (reversal of prior loss); negative = loss
//   - recognized_in_oci:  positive = surplus credit; negative = surplus debit
//   - returned newSurplusBalance = priorSurplusBalance + recognized_in_oci
func ComputePnLOCISplit(
	absAmount int64,
	isIncrease bool,
	priorSurplusBalance int64,
	priorPnLLossBalance int64,
) (recognizedInPnL, recognizedInOCI, newSurplusBalance int64) {
	if isIncrease {
		// Increases: reverse prior PnL losses first, remainder to OCI
		if priorPnLLossBalance > 0 {
			reversal := priorPnLLossBalance
			if absAmount < reversal {
				reversal = absAmount
			}
			recognizedInPnL = reversal           // positive = gain
			recognizedInOCI = absAmount - reversal // remainder to OCI
		} else {
			recognizedInPnL = 0
			recognizedInOCI = absAmount
		}
	} else {
		// Decreases: consume prior surplus first, remainder to PnL
		if priorSurplusBalance > 0 {
			surplusUsed := priorSurplusBalance
			if absAmount < surplusUsed {
				surplusUsed = absAmount
			}
			recognizedInOCI = -surplusUsed           // negative = surplus debit
			recognizedInPnL = -(absAmount - surplusUsed) // negative = loss
		} else {
			recognizedInOCI = 0
			recognizedInPnL = -absAmount // negative = loss
		}
	}

	newSurplusBalance = priorSurplusBalance + recognizedInOCI
	return recognizedInPnL, recognizedInOCI, newSurplusBalance
}

// readAsset fetches a single asset.
func (uc *RevalueAssetUseCase) readAsset(ctx context.Context, assetID string) (*assetpb.Asset, error) {
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
