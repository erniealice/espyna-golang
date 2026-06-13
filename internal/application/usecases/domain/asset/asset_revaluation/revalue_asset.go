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
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// RevalueAssetRequest is the internal input to the use case.
// Kept for internal helpers that still use Go-struct fields.
// The public boundary uses *revaluation_pb.RevalueAssetUseCaseRequest.
type RevalueAssetRequest struct {
	AssetID         string
	NewFairValue    int64 // centavos
	AppraiserName   string
	ValuationMethod string
	Notes           string
}

// RevalueAssetResult is the internal output.
// Kept for internal helpers; the public boundary returns *revaluation_pb.RevalueAssetUseCaseResponse.
type RevalueAssetResult struct {
	Revaluation *revaluation_pb.AssetRevaluation
	Transaction *assettxpb.AssetTransaction
}

// RevalueAssetUseCase executes the IAS 16 revaluation flow:
//  1. SELECT asset FOR UPDATE (acquire row lock — implicit via PostgreSQL MVCC
//     when ExecuteInTransaction wraps the read; see Followup note in
//     docs/plan/20260510-asset-depreciation-defect-fix/progress.md for the
//     plan to add an explicit FOR UPDATE variant).
//  2. Compute revaluation_amount, is_increase
//  3. Walk immutable AssetRevaluation history (oldest first) to derive surplus
//     and prior-loss balances (Option A).
//  4. Apply IAS 16.39-40 four-case PnL/OCI split.
//  5. INSERT asset_revaluation (full IFRS row).
//  6. INSERT asset_transaction (REVALUATION_UP|DOWN) with asset_revaluation_id back-ref.
//  7. UPDATE asset.book_value = asset.fair_value = new_fair_value.
//
// All seven steps run inside a single Transactor.ExecuteInTransaction
// closure so the history read + writes are atomic. A concurrent revaluation
// on the same asset cannot misallocate the split.
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

// Execute performs the revaluation within a single transaction. The history
// read + insert + asset update all happen inside one ExecuteInTransaction so
// concurrent revaluations on the same asset cannot misallocate the split.
func (uc *RevalueAssetUseCase) Execute(
	ctx context.Context,
	pbReq *revaluation_pb.RevalueAssetUseCaseRequest,
) (*revaluation_pb.RevalueAssetUseCaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAssetRevaluation,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Translate proto → internal Go struct at the boundary.
	req := RevalueAssetRequest{}
	if pbReq != nil {
		req.AssetID = pbReq.GetAssetId()
		req.NewFairValue = pbReq.GetNewFairValue()
		if pbReq.AppraiserName != nil {
			req.AppraiserName = pbReq.GetAppraiserName()
		}
		if pbReq.ValuationMethod != nil {
			req.ValuationMethod = pbReq.GetValuationMethod()
		}
		if pbReq.Notes != nil {
			req.Notes = pbReq.GetNotes()
		}
	}

	if req.AssetID == "" {
		return nil, errors.New("revalue_asset: asset_id is required")
	}
	if req.NewFairValue <= 0 {
		return nil, errors.New("revalue_asset: new_fair_value must be > 0")
	}

	// Workspace tenancy check (Phase 1 — codex C2): require a workspace_id from
	// context. The full asset-row tenant verification happens INSIDE the tx
	// below (after the lock is held) so a concurrent ownership change cannot
	// race the check.
	workspaceID := strings.TrimSpace(contextutil.ExtractWorkspaceIDFromContext(ctx))
	if workspaceID == "" {
		// TODO: translate via Translator (Fix #4 deferred — codex L1).
		// Suggested key: asset.assetDetail.depreciationRun.errors.workspaceRequired
		return nil, errors.New("revalue_asset: Workspace context required")
	}

	var result *RevalueAssetResult // internal — translated to proto on return

	// All steps run inside a single tx so the history read is consistent with
	// the write. Note: the row lock on the asset is implicit via PostgreSQL
	// MVCC + tx isolation — there is no explicit `SELECT … FOR UPDATE` variant
	// available on the repo today. A future phase can swap in an explicit
	// SelectForUpdate adapter call without changing this code's structure.
	executeCore := func(txCtx context.Context) error {
		// Step 1: READ asset (workspace-scoped). The decorator's workspace
		// auto-filter rejects mismatched ownership.
		asset, err := uc.readAsset(txCtx, req.AssetID)
		if err != nil || asset == nil {
			return fmt.Errorf("revalue_asset: asset %q not found: %w", req.AssetID, err)
		}
		if got := strings.TrimSpace(asset.GetWorkspaceId()); got != workspaceID {
			return fmt.Errorf("revalue_asset: asset %q does not belong to workspace %q", req.AssetID, workspaceID)
		}

		// Step 1b: H4 measurement-model gate. Only REVALUATION-model assets
		// can be revalued. COST-model assets are rejected at the boundary so
		// no asset_revaluation or asset_transaction row is ever written for
		// them.
		if asset.GetMeasurementModel() != assetpb.MeasurementModel_MEASUREMENT_MODEL_REVALUATION {
			// TODO: translate via Translator (Fix #4 deferred — codex L1).
			// Suggested key: asset.assetRevaluation.errors.wrongMeasurementModel
			return errors.New("revalue_asset: asset measurement_model must be REVALUATION to be revalued (codex H4)")
		}

		// Step 2: Compute revaluation_amount, is_increase
		currentBookValue := asset.GetBookValue()
		revaluationAmount := req.NewFairValue - currentBookValue
		if revaluationAmount == 0 {
			return errors.New("revalue_asset: new_fair_value equals current book value — no revaluation needed")
		}
		isIncrease := revaluationAmount > 0
		absAmount := revaluationAmount
		if absAmount < 0 {
			absAmount = -absAmount
		}

		// Step 3: Option A — derive surplus state from immutable
		// AssetRevaluation history. Walked oldest-first so reversing/consuming
		// events apply in chronological order.
		priorSurplusBalance, priorPnLLossBalance, err := deriveSurplusStateFromHistory(
			txCtx, uc.repositories.AssetRevaluation, req.AssetID,
		)
		if err != nil {
			return fmt.Errorf("revalue_asset: surplus state derivation failed: %w", err)
		}

		// Step 4: Apply IAS 16.39-40 PnL/OCI split.
		recognizedInPnL, recognizedInOCI, newSurplusBalance := ComputePnLOCISplit(
			absAmount, isIncrease, priorSurplusBalance, priorPnLLossBalance,
		)

		// Step 5: INSERT asset_revaluation
		revDate := time.Now().UTC().Format("2006-01-02")
		revID := uc.services.IDGenerator.GenerateID()
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
			Id:                        revID,
			AssetId:                   req.AssetID,
			RevaluationDate:           revDate,
			PreviousCarryingAmount:    currentBookValue,
			NewFairValue:              req.NewFairValue,
			RevaluationAmount:         revaluationAmount,
			IsIncrease:                isIncrease,
			RecognizedInPnl:           recognizedInPnL,
			RecognizedInOci:           recognizedInOCI,
			RevaluationSurplusBalance: newSurplusBalance,
			AppraiserName:             appraiserPtr,
			ValuationMethod:           valMethodPtr,
			Notes:                     notesPtr,
			DateCreated:               &now,
			DateCreatedString:         &nowStr,
			Active:                    true,
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

		// Step 6: INSERT asset_transaction (REVALUATION_UP or REVALUATION_DOWN)
		txType := assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_REVALUATION_UP
		if !isIncrease {
			txType = assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_REVALUATION_DOWN
		}
		txID := uc.services.IDGenerator.GenerateID()
		txAmount := absAmount // always positive; type discriminates direction
		revIDStr := rev.GetId()
		initiator := contextutil.ExtractWorkspaceUserIDFromContext(txCtx)

		assetTx := &assettxpb.AssetTransaction{
			Id:                    txID,
			AssetId:               req.AssetID,
			TransactionType:       txType,
			TransactionDate:       now,
			TransactionDateString: time.Now().UTC().Format("2006-01-02"),
			Amount:                txAmount,
			PerformedBy:           &initiator,
			AssetRevaluationId:    &revIDStr,
			Active:                true,
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

		// Step 7: UPDATE asset.book_value = asset.fair_value = new_fair_value
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

	// Execute within a transaction if supported. The NoOp service runs inline
	// (used by unit tests); production wires a PostgreSQL Transactor
	// that opens BEGIN/COMMIT around the closure.
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, executeCore); err != nil {
			return nil, err
		}
	} else {
		if err := executeCore(ctx); err != nil {
			return nil, err
		}
	}

	// Translate internal result to proto response.
	resp := &revaluation_pb.RevalueAssetUseCaseResponse{Success: true}
	if result != nil {
		resp.Revaluation = result.Revaluation
		if result.Transaction != nil {
			txID := result.Transaction.GetId()
			resp.AssetTransactionId = &txID
		}
	}
	return resp, nil
}

// deriveSurplusStateFromHistory walks the immutable AssetRevaluation history
// for an asset (oldest first) and returns the running surplus balance and the
// running prior-loss balance at the END of the history, ready for a fresh
// revaluation event to consume.
//
// Two running balances are maintained:
//
//  1. surplus — accumulated `recognized_in_oci`. Each event's recognized_in_oci
//     can be positive (up event creating/adding surplus) or negative (down
//     event consuming prior surplus). Clamps at 0 from below: surplus is never
//     negative; once exhausted, further down-revaluations fall to PnL.
//
//  2. priorLoss — net unreversed prior PnL losses. Each event's
//     recognized_in_pnl can be negative (down event with no surplus to absorb,
//     adding to prior loss) or positive (up event reversing prior loss).
//     We accumulate the NEGATION of recognized_in_pnl so down→positive add,
//     up→positive subtract. Clamps at 0 from below: priorLoss is never
//     negative.
//
// Both balances are returned as positive int64 centavos (the convention the
// IAS-16 ComputePnLOCISplit function expects).
//
// Ordering: events are sorted by (revaluation_date ASC, date_created ASC, id ASC)
// to give a deterministic chronological walk even when the repository does not
// guarantee sorted output.
func deriveSurplusStateFromHistory(
	ctx context.Context,
	repo revaluation_pb.AssetRevaluationDomainServiceServer,
	assetID string,
) (priorSurplusBalance int64, priorPnLLossBalance int64, err error) {
	if repo == nil {
		return 0, 0, nil
	}

	// Request sort by recorded_at ASC. Even if the underlying adapter ignores
	// the SortRequest, we re-sort defensively in-memory below.
	sortAsc := commonpb.SortDirection_ASC
	resp, err := repo.ListAssetRevaluations(ctx, &revaluation_pb.ListAssetRevaluationsRequest{
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
		Sort: &commonpb.SortRequest{
			Fields: []*commonpb.SortField{
				{Field: "revaluation_date", Direction: sortAsc},
				{Field: "date_created", Direction: sortAsc},
				{Field: "id", Direction: sortAsc},
			},
		},
	})
	if err != nil || resp == nil {
		return 0, 0, err
	}

	history := append([]*revaluation_pb.AssetRevaluation(nil), resp.GetData()...)

	// Defensive in-memory sort (oldest first). The repository SortRequest may
	// be a no-op on some adapters; the algorithm REQUIRES chronological order.
	sort.SliceStable(history, func(i, j int) bool {
		a, b := history[i], history[j]
		if a.GetRevaluationDate() != b.GetRevaluationDate() {
			return a.GetRevaluationDate() < b.GetRevaluationDate()
		}
		if a.GetDateCreated() != b.GetDateCreated() {
			return a.GetDateCreated() < b.GetDateCreated()
		}
		return a.GetId() < b.GetId()
	})

	// Walk oldest first, maintaining two running balances.
	surplus := int64(0)
	priorLoss := int64(0)
	for _, h := range history {
		// Apply the OCI delta: positive adds to surplus, negative consumes it.
		surplus += h.GetRecognizedInOci()
		if surplus < 0 {
			surplus = 0
		}
		// Apply the PnL delta to priorLoss balance:
		//   recognized_in_pnl < 0 (down event) → adds to priorLoss
		//   recognized_in_pnl > 0 (up event reversing prior loss) → subtracts
		priorLoss += -h.GetRecognizedInPnl()
		if priorLoss < 0 {
			priorLoss = 0
		}
	}

	return surplus, priorLoss, nil
}

// ComputePnLOCISplit applies IAS 16.39-40 to split the revaluation amount into
// recognized_in_pnl and recognized_in_oci, and returns the new running surplus
// balance after applying the split.
//
// IAS 16.39 (revaluation INCREASE / gain):
//   - Default: the increase is recognised in OCI and accumulated in equity
//     under "revaluation surplus."
//   - Exception: to the extent the increase reverses a prior revaluation
//     decrease that was recognised in PROFIT OR LOSS, recognise the increase
//     in profit or loss (up to the prior loss balance).
//
// IAS 16.40 (revaluation DECREASE / loss):
//   - Default: the decrease is recognised in PROFIT OR LOSS.
//   - Exception: to the extent of any credit balance existing in the
//     revaluation surplus in respect of the same asset, recognise the
//     decrease in OCI (up to the surplus balance).
//
// The four cases:
//
//	(a) Up + no prior loss      → full amount → OCI (creates/adds surplus).
//	(b) Up + prior loss > 0     → PnL = min(amount, prior_loss); OCI = amount − PnL.
//	(c) Down + no prior surplus → full amount → PnL.
//	(d) Down + prior surplus    → OCI = −min(amount, surplus); PnL = OCI − amount (more negative remainder).
//
// Sign conventions (signed centavos, int64):
//
//	recognized_in_pnl: positive = gain (PnL credit, reverses prior loss);
//	                   negative = loss (PnL debit).
//	recognized_in_oci: positive = surplus credit (OCI gain);
//	                   negative = surplus debit (OCI loss, consumes prior surplus).
//
// All inputs are POSITIVE int64 centavos:
//
//	absAmount           — magnitude of |new_fair_value − previous_carrying_amount|
//	priorSurplusBalance — current surplus balance (>= 0)
//	priorPnLLossBalance — current unreversed prior PnL losses (>= 0)
//
// Returned newSurplusBalance is the running surplus AFTER applying this entry
// (used for tracking on the AssetRevaluation row).
func ComputePnLOCISplit(
	absAmount int64,
	isIncrease bool,
	priorSurplusBalance int64,
	priorPnLLossBalance int64,
) (recognizedInPnL, recognizedInOCI, newSurplusBalance int64) {
	if isIncrease {
		// IAS 16.39 — increase. First reverse prior PnL losses, then OCI.
		if priorPnLLossBalance > 0 {
			reversal := priorPnLLossBalance
			if absAmount < reversal {
				reversal = absAmount
			}
			recognizedInPnL = reversal             // positive = gain (reversal)
			recognizedInOCI = absAmount - reversal // remainder to OCI
		} else {
			recognizedInPnL = 0
			recognizedInOCI = absAmount
		}
	} else {
		// IAS 16.40 — decrease. First consume prior surplus, then PnL.
		if priorSurplusBalance > 0 {
			surplusUsed := priorSurplusBalance
			if absAmount < surplusUsed {
				surplusUsed = absAmount
			}
			recognizedInOCI = -surplusUsed               // negative = surplus debit
			recognizedInPnL = -(absAmount - surplusUsed) // negative = loss
		} else {
			recognizedInOCI = 0
			recognizedInPnL = -absAmount // negative = loss
		}
	}

	newSurplusBalance = priorSurplusBalance + recognizedInOCI
	if newSurplusBalance < 0 {
		newSurplusBalance = 0
	}
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
