package asset_revaluation_test

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revaluationuc "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_revaluation"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
)

// =============================================================================
// ComputePnLOCISplit unit tests (Phase 0 — kept here unchanged so the 4 IAS
// 16.39-40 cases remain individually documented in code).
// =============================================================================

// TestComputePnLOCISplit_UpFromCost — IAS 16.39: increase with no prior surplus.
// Case 1 (of 4): up-from-cost, no prior history → fully OCI.
func TestComputePnLOCISplit_UpFromCost(t *testing.T) {
	// New asset, first revaluation, cost model → no prior surplus or PnL loss.
	// Increase of 500_000 centavos.
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(500_000, true, 0, 0)
	if pnl != 0 {
		t.Errorf("up-from-cost: expected pnl=0, got %d", pnl)
	}
	if oci != 500_000 {
		t.Errorf("up-from-cost: expected oci=500_000, got %d", oci)
	}
	if surplus != 500_000 {
		t.Errorf("up-from-cost: expected newSurplus=500_000, got %d", surplus)
	}
}

// TestComputePnLOCISplit_UpReversingPriorDown — IAS 16.40: prior PnL loss exists.
// Case 2: increase that reverses a prior down-revaluation that was expensed to PnL.
//
// Setup: prior PnL loss = 200_000 (expensed).
// Increase = 300_000 centavos.
// Expected: PnL gain (reversal) = 200_000; OCI = 100_000; surplus = 100_000.
func TestComputePnLOCISplit_UpReversingPriorDown(t *testing.T) {
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(300_000, true, 0, 200_000)
	if pnl != 200_000 {
		t.Errorf("up-reversing-prior-down: expected pnl=200_000 (reversal), got %d", pnl)
	}
	if oci != 100_000 {
		t.Errorf("up-reversing-prior-down: expected oci=100_000, got %d", oci)
	}
	if surplus != 100_000 {
		t.Errorf("up-reversing-prior-down: expected newSurplus=100_000, got %d", surplus)
	}
}

// TestComputePnLOCISplit_DownFromCost — IAS 16.39: decrease with no prior surplus.
// Case 3: first revaluation is a decrease, no prior surplus → fully PnL.
func TestComputePnLOCISplit_DownFromCost(t *testing.T) {
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(400_000, false, 0, 0)
	if pnl != -400_000 {
		t.Errorf("down-from-cost: expected pnl=-400_000, got %d", pnl)
	}
	if oci != 0 {
		t.Errorf("down-from-cost: expected oci=0, got %d", oci)
	}
	if surplus != 0 {
		t.Errorf("down-from-cost: expected newSurplus=0, got %d", surplus)
	}
}

// TestComputePnLOCISplit_DownExceedingSurplus — IAS 16.40: decrease exceeds prior surplus.
// Case 4: prior surplus = 200_000; decrease = 350_000.
// Expected: OCI = -200_000 (surplus absorbed); PnL = -150_000 (remainder); newSurplus = 0.
func TestComputePnLOCISplit_DownExceedingSurplus(t *testing.T) {
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(350_000, false, 200_000, 0)
	if oci != -200_000 {
		t.Errorf("down-exceeding-surplus: expected oci=-200_000, got %d", oci)
	}
	if pnl != -150_000 {
		t.Errorf("down-exceeding-surplus: expected pnl=-150_000, got %d", pnl)
	}
	if surplus != 0 {
		t.Errorf("down-exceeding-surplus: expected newSurplus=0, got %d", surplus)
	}
}

// TestComputePnLOCISplit_DownWithinSurplus — decrease fully absorbed by surplus.
// prior surplus = 500_000; decrease = 200_000.
// Expected: OCI = -200_000; PnL = 0; newSurplus = 300_000.
func TestComputePnLOCISplit_DownWithinSurplus(t *testing.T) {
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(200_000, false, 500_000, 0)
	if oci != -200_000 {
		t.Errorf("down-within-surplus: expected oci=-200_000, got %d", oci)
	}
	if pnl != 0 {
		t.Errorf("down-within-surplus: expected pnl=0, got %d", pnl)
	}
	if surplus != 300_000 {
		t.Errorf("down-within-surplus: expected newSurplus=300_000, got %d", surplus)
	}
}

// TestComputePnLOCISplit_UpExactlyReversesPriorLoss — increase exactly covers prior loss.
// prior PnL loss = 100_000; increase = 100_000.
// Expected: PnL = 100_000 (full reversal); OCI = 0; surplus = 0.
func TestComputePnLOCISplit_UpExactlyReversesPriorLoss(t *testing.T) {
	pnl, oci, surplus := revaluationuc.ComputePnLOCISplit(100_000, true, 0, 100_000)
	if pnl != 100_000 {
		t.Errorf("up-exact-reversal: expected pnl=100_000, got %d", pnl)
	}
	if oci != 0 {
		t.Errorf("up-exact-reversal: expected oci=0, got %d", oci)
	}
	if surplus != 0 {
		t.Errorf("up-exact-reversal: expected newSurplus=0, got %d", surplus)
	}
}

// =============================================================================
// In-memory mock repositories for Execute() integration tests (Phase 3).
//
// Each mock is intentionally minimal — implements only the methods the use
// case calls. Other methods would panic via the embedded
// Unimplemented*Server, but the use case never invokes them.
// =============================================================================

// fakeAssetRepo — minimal AssetDomainServiceServer.
type fakeAssetRepo struct {
	assetpb.UnimplementedAssetDomainServiceServer
	byID    map[string]*assetpb.Asset
	updates []*assetpb.Asset
}

func (r *fakeAssetRepo) ReadAsset(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
	id := req.GetData().GetId()
	a, ok := r.byID[id]
	if !ok {
		return &assetpb.ReadAssetResponse{}, nil
	}
	return &assetpb.ReadAssetResponse{Data: []*assetpb.Asset{a}}, nil
}

func (r *fakeAssetRepo) UpdateAsset(ctx context.Context, req *assetpb.UpdateAssetRequest) (*assetpb.UpdateAssetResponse, error) {
	r.updates = append(r.updates, req.GetData())
	if a, ok := r.byID[req.GetData().GetId()]; ok {
		updated := *a
		updated.BookValue = req.GetData().GetBookValue()
		if req.GetData().FairValue != nil {
			fv := req.GetData().GetFairValue()
			updated.FairValue = &fv
		}
		r.byID[updated.GetId()] = &updated
	}
	return &assetpb.UpdateAssetResponse{Data: []*assetpb.Asset{req.GetData()}}, nil
}

// fakeAssetTransactionRepo records inserts.
type fakeAssetTransactionRepo struct {
	assettxpb.UnimplementedAssetTransactionDomainServiceServer
	created []*assettxpb.AssetTransaction
}

func (r *fakeAssetTransactionRepo) CreateAssetTransaction(ctx context.Context, req *assettxpb.CreateAssetTransactionRequest) (*assettxpb.CreateAssetTransactionResponse, error) {
	r.created = append(r.created, req.GetData())
	return &assettxpb.CreateAssetTransactionResponse{Data: []*assettxpb.AssetTransaction{req.GetData()}}, nil
}

// fakeRevaluationRepo records inserts and serves history reads.
// Insert order is preserved as the chronological order (each subsequent
// insert is "later" than prior ones for sort purposes — RevaluationDate
// is filled in from a deterministic per-insert counter so the in-memory
// stable-sort in deriveSurplusStateFromHistory is deterministic).
type fakeRevaluationRepo struct {
	revaluation_pb.UnimplementedAssetRevaluationDomainServiceServer
	created []*revaluation_pb.AssetRevaluation
	// dayCounter advances RevaluationDate by 1 per insert so the in-memory
	// chronological sort is stable + deterministic.
	dayCounter int
}

func (r *fakeRevaluationRepo) CreateAssetRevaluation(ctx context.Context, req *revaluation_pb.CreateAssetRevaluationRequest) (*revaluation_pb.CreateAssetRevaluationResponse, error) {
	row := req.GetData()
	// If the use case did not stamp RevaluationDate (it does, but defensive),
	// or if we want strictly increasing dates regardless of clock granularity,
	// override with a counter-based date so the test order is deterministic.
	r.dayCounter++
	row.RevaluationDate = isoDate(2025, 1, r.dayCounter)
	r.created = append(r.created, row)
	return &revaluation_pb.CreateAssetRevaluationResponse{Data: []*revaluation_pb.AssetRevaluation{row}}, nil
}

func (r *fakeRevaluationRepo) ListAssetRevaluations(ctx context.Context, req *revaluation_pb.ListAssetRevaluationsRequest) (*revaluation_pb.ListAssetRevaluationsResponse, error) {
	target := req.GetAssetId()
	out := make([]*revaluation_pb.AssetRevaluation, 0, len(r.created))
	for _, row := range r.created {
		if row.GetAssetId() == target {
			out = append(out, row)
		}
	}
	// We deliberately return in INSERT order (chronological). The use case
	// re-sorts in-memory, so this represents adapter-side guarantees.
	return &revaluation_pb.ListAssetRevaluationsResponse{Data: out}, nil
}

// isoDate produces a YYYY-MM-DD string. Day rolls over month/year if the
// counter exceeds 28 — kept simple since tests only insert a few revaluations.
func isoDate(year, month, day int) string {
	for day > 28 {
		day -= 28
		month++
		if month > 12 {
			month = 1
			year++
		}
	}
	// Manual int→2-digit string to avoid pulling in fmt for tests.
	twoDigit := func(n int) string {
		if n < 10 {
			return "0" + itoa(n)
		}
		return itoa(n)
	}
	return itoa(year) + "-" + twoDigit(month) + "-" + twoDigit(day)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if negative {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// =============================================================================
// Execute() helper — wires the use case with NoOp services and ws-bound ctx.
// =============================================================================

func newRevalueUseCaseWithRepos(
	asset *fakeAssetRepo,
	tx *fakeAssetTransactionRepo,
	rev *fakeRevaluationRepo,
) *revaluationuc.RevalueAssetUseCase {
	return revaluationuc.NewRevalueAssetUseCase(
		revaluationuc.RevalueAssetRepositories{
			Asset:            asset,
			AssetTransaction: tx,
			AssetRevaluation: rev,
		},
		revaluationuc.RevalueAssetServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   ports.NewNoOpTransactionService(),
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            ports.NewNoOpIDService(),
		},
	)
}

func ctxForWorkspace(workspaceID string) context.Context {
	return contextutil.WithWorkspaceID(context.Background(), workspaceID)
}

// newRevaluationAsset builds a REVALUATION-model asset with a starting book
// value. workspaceID is used so the use case's tenancy gate passes.
func newRevaluationAsset(id, workspaceID string, bookValue int64) *assetpb.Asset {
	ws := workspaceID
	return &assetpb.Asset{
		Id:               id,
		WorkspaceId:      &ws,
		Name:             "test-asset-" + id,
		BookValue:        bookValue,
		MeasurementModel: assetpb.MeasurementModel_MEASUREMENT_MODEL_REVALUATION,
		Status:           assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE,
		Active:           true,
	}
}

// newCostAsset builds a COST-model asset (used by the H4 gate test).
func newCostAsset(id, workspaceID string, bookValue int64) *assetpb.Asset {
	ws := workspaceID
	return &assetpb.Asset{
		Id:               id,
		WorkspaceId:      &ws,
		Name:             "test-cost-asset-" + id,
		BookValue:        bookValue,
		MeasurementModel: assetpb.MeasurementModel_MEASUREMENT_MODEL_COST,
		Status:           assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE,
		Active:           true,
	}
}

// =============================================================================
// Phase 3 — H4 measurement-model gate test
// =============================================================================

// TestRevalueAsset_RejectsCostModelAsset verifies the H4 gate: calling
// RevalueAsset on a COST-model asset returns the gate error and writes
// nothing to the database (no AssetRevaluation, no AssetTransaction, no
// asset update).
func TestRevalueAsset_RejectsCostModelAsset(t *testing.T) {
	asset := newCostAsset("asset-cost-1", "ws-1", 100_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)

	res, err := uc.Execute(ctxForWorkspace("ws-1"), &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId:      "asset-cost-1",
		NewFairValue: 150_000,
	})
	if err == nil {
		t.Fatalf("expected error rejecting COST-model asset, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result on reject, got %+v", res)
	}

	// No DB writes should have occurred.
	if len(revRepo.created) != 0 {
		t.Errorf("expected zero asset_revaluation inserts on COST-model reject, got %d", len(revRepo.created))
	}
	if len(txRepo.created) != 0 {
		t.Errorf("expected zero asset_transaction inserts on COST-model reject, got %d", len(txRepo.created))
	}
	if len(assetRepo.updates) != 0 {
		t.Errorf("expected zero asset updates on COST-model reject, got %d", len(assetRepo.updates))
	}

	// Sanity: rejecting message matches the H4 gate (substring check —
	// raw error today; will be translated via lyngua in Phase 7.3).
	if want := "REVALUATION"; !contains(err.Error(), want) {
		t.Errorf("expected error to mention %q (the required model), got %q", want, err.Error())
	}
}

// =============================================================================
// Phase 3 — Multi-revaluation history scenarios via Execute()
//
// These scenarios exercise deriveSurplusStateFromHistory + ComputePnLOCISplit
// end-to-end through the use case. The COMPUTE function unit tests above
// validate the math; these tests prove the HISTORY WALK is correct (signs,
// ordering, clamping at zero).
// =============================================================================

// TestRevalueAsset_UpThenDown — up creates surplus; down consumes it then
// spills to PnL when surplus exhausted.
//
// Setup:
//
//	Asset starts at BV=100_000 (REVALUATION model).
//	Step 1: revalue to 300_000 (up by 200_000) → OCI=200_000, PnL=0, surplus=200_000.
//	Step 2: revalue to 50_000 (down by 250_000 from 300_000 — overshoots surplus).
//	  Expected: OCI=-200_000 (full surplus consumed), PnL=-50_000 (remainder), surplus=0.
func TestRevalueAsset_UpThenDown(t *testing.T) {
	asset := newRevaluationAsset("asset-up-down", "ws-1", 100_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)
	ctx := ctxForWorkspace("ws-1")

	// Step 1: up to 300_000.
	if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId: "asset-up-down", NewFairValue: 300_000,
	}); err != nil {
		t.Fatalf("step1 (up) error: %v", err)
	}

	// Step 2: down to 50_000 (book value now 300_000, so down by 250_000).
	if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId: "asset-up-down", NewFairValue: 50_000,
	}); err != nil {
		t.Fatalf("step2 (down) error: %v", err)
	}

	if len(revRepo.created) != 2 {
		t.Fatalf("expected 2 revaluation rows, got %d", len(revRepo.created))
	}
	step1 := revRepo.created[0]
	step2 := revRepo.created[1]

	if step1.GetRecognizedInOci() != 200_000 || step1.GetRecognizedInPnl() != 0 {
		t.Errorf("step1 expected oci=200_000,pnl=0; got oci=%d,pnl=%d",
			step1.GetRecognizedInOci(), step1.GetRecognizedInPnl())
	}
	if step1.GetRevaluationSurplusBalance() != 200_000 {
		t.Errorf("step1 expected surplus=200_000, got %d", step1.GetRevaluationSurplusBalance())
	}

	if step2.GetRecognizedInOci() != -200_000 {
		t.Errorf("step2 expected oci=-200_000 (full surplus consumed), got %d", step2.GetRecognizedInOci())
	}
	if step2.GetRecognizedInPnl() != -50_000 {
		t.Errorf("step2 expected pnl=-50_000 (remainder loss), got %d", step2.GetRecognizedInPnl())
	}
	if step2.GetRevaluationSurplusBalance() != 0 {
		t.Errorf("step2 expected surplus=0 (fully exhausted), got %d", step2.GetRevaluationSurplusBalance())
	}
}

// TestRevalueAsset_DownThenUp — down creates a PnL loss; up reverses it,
// remainder lands in OCI.
//
// Setup:
//
//	Asset starts at BV=300_000.
//	Step 1: revalue to 100_000 (down by 200_000) → PnL=-200_000, OCI=0, prior_loss=200_000.
//	Step 2: revalue to 350_000 (up by 250_000 from 100_000).
//	  Expected: PnL=+200_000 (full reversal), OCI=+50_000 (remainder), surplus=50_000.
func TestRevalueAsset_DownThenUp(t *testing.T) {
	asset := newRevaluationAsset("asset-down-up", "ws-1", 300_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)
	ctx := ctxForWorkspace("ws-1")

	if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId: "asset-down-up", NewFairValue: 100_000,
	}); err != nil {
		t.Fatalf("step1 (down) error: %v", err)
	}
	if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId: "asset-down-up", NewFairValue: 350_000,
	}); err != nil {
		t.Fatalf("step2 (up) error: %v", err)
	}

	if len(revRepo.created) != 2 {
		t.Fatalf("expected 2 revaluation rows, got %d", len(revRepo.created))
	}
	step1 := revRepo.created[0]
	step2 := revRepo.created[1]

	if step1.GetRecognizedInPnl() != -200_000 || step1.GetRecognizedInOci() != 0 {
		t.Errorf("step1 expected pnl=-200_000,oci=0; got pnl=%d,oci=%d",
			step1.GetRecognizedInPnl(), step1.GetRecognizedInOci())
	}

	if step2.GetRecognizedInPnl() != 200_000 {
		t.Errorf("step2 expected pnl=+200_000 (full reversal), got %d", step2.GetRecognizedInPnl())
	}
	if step2.GetRecognizedInOci() != 50_000 {
		t.Errorf("step2 expected oci=+50_000 (remainder to surplus), got %d", step2.GetRecognizedInOci())
	}
	if step2.GetRevaluationSurplusBalance() != 50_000 {
		t.Errorf("step2 expected surplus=50_000, got %d", step2.GetRevaluationSurplusBalance())
	}
}

// TestRevalueAsset_UpDownUp — three transitions verifying that surplus is
// rebuilt correctly after partial consumption.
//
// Setup:
//
//	Asset starts at BV=100_000.
//	Step 1: up to 300_000 (+200_000) → OCI=200_000, surplus=200_000.
//	Step 2: down to 250_000 (-50_000 from 300_000) → OCI=-50_000, PnL=0, surplus=150_000.
//	Step 3: up to 400_000 (+150_000 from 250_000) → OCI=+150_000, PnL=0, surplus=300_000.
func TestRevalueAsset_UpDownUp(t *testing.T) {
	asset := newRevaluationAsset("asset-udu", "ws-1", 100_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)
	ctx := ctxForWorkspace("ws-1")

	for i, fv := range []int64{300_000, 250_000, 400_000} {
		if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
			AssetId: "asset-udu", NewFairValue: fv,
		}); err != nil {
			t.Fatalf("step%d error: %v", i+1, err)
		}
	}

	if len(revRepo.created) != 3 {
		t.Fatalf("expected 3 revaluation rows, got %d", len(revRepo.created))
	}
	step2 := revRepo.created[1]
	step3 := revRepo.created[2]

	if step2.GetRecognizedInOci() != -50_000 || step2.GetRecognizedInPnl() != 0 {
		t.Errorf("step2 expected oci=-50_000,pnl=0; got oci=%d,pnl=%d",
			step2.GetRecognizedInOci(), step2.GetRecognizedInPnl())
	}
	if step2.GetRevaluationSurplusBalance() != 150_000 {
		t.Errorf("step2 expected surplus=150_000 (200k-50k), got %d", step2.GetRevaluationSurplusBalance())
	}
	if step3.GetRecognizedInOci() != 150_000 || step3.GetRecognizedInPnl() != 0 {
		t.Errorf("step3 expected oci=+150_000,pnl=0 (no prior loss); got oci=%d,pnl=%d",
			step3.GetRecognizedInOci(), step3.GetRecognizedInPnl())
	}
	if step3.GetRevaluationSurplusBalance() != 300_000 {
		t.Errorf("step3 expected surplus=300_000 (150k+150k), got %d", step3.GetRevaluationSurplusBalance())
	}
}

// TestRevalueAsset_UpDownPastSurplus_Up — the most subtle multi-step scenario.
// An up creates surplus; a down EXCEEDS the surplus (so it spills to PnL); a
// follow-up up must reverse the PnL portion FIRST before crediting OCI.
//
// Setup:
//
//	Asset starts at BV=100_000.
//	Step 1: up to 200_000 (+100_000) → OCI=100_000, surplus=100_000.
//	Step 2: down to 50_000 (-150_000 from 200_000) →
//	    OCI=-100_000 (full surplus consumed), PnL=-50_000 (remainder), surplus=0, prior_loss=50_000.
//	Step 3: up to 200_000 (+150_000 from 50_000) →
//	    PnL=+50_000 (full reversal of prior loss), OCI=+100_000 (remainder to surplus), surplus=100_000, prior_loss=0.
//
// This is the "up reverses PnL FIRST, OCI second" invariant — IAS 16.39.
func TestRevalueAsset_UpDownPastSurplus_Up(t *testing.T) {
	asset := newRevaluationAsset("asset-udu-past", "ws-1", 100_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)
	ctx := ctxForWorkspace("ws-1")

	for i, fv := range []int64{200_000, 50_000, 200_000} {
		if _, err := uc.Execute(ctx, &revaluation_pb.RevalueAssetUseCaseRequest{
			AssetId: "asset-udu-past", NewFairValue: fv,
		}); err != nil {
			t.Fatalf("step%d error: %v", i+1, err)
		}
	}

	if len(revRepo.created) != 3 {
		t.Fatalf("expected 3 revaluation rows, got %d", len(revRepo.created))
	}
	step1 := revRepo.created[0]
	step2 := revRepo.created[1]
	step3 := revRepo.created[2]

	// Step 1 verification.
	if step1.GetRecognizedInOci() != 100_000 || step1.GetRecognizedInPnl() != 0 {
		t.Errorf("step1 expected oci=100_000,pnl=0; got oci=%d,pnl=%d",
			step1.GetRecognizedInOci(), step1.GetRecognizedInPnl())
	}

	// Step 2 verification — surplus consumed, remainder to PnL.
	if step2.GetRecognizedInOci() != -100_000 {
		t.Errorf("step2 expected oci=-100_000 (full surplus consumed), got %d", step2.GetRecognizedInOci())
	}
	if step2.GetRecognizedInPnl() != -50_000 {
		t.Errorf("step2 expected pnl=-50_000 (remainder loss after surplus exhausted), got %d", step2.GetRecognizedInPnl())
	}
	if step2.GetRevaluationSurplusBalance() != 0 {
		t.Errorf("step2 expected surplus=0 (exhausted), got %d", step2.GetRevaluationSurplusBalance())
	}

	// Step 3 verification — THIS is the IAS 16.39 invariant: up MUST reverse
	// prior PnL loss FIRST, then credit OCI.
	if step3.GetRecognizedInPnl() != 50_000 {
		t.Errorf("step3 expected pnl=+50_000 (must reverse prior loss FIRST), got %d", step3.GetRecognizedInPnl())
	}
	if step3.GetRecognizedInOci() != 100_000 {
		t.Errorf("step3 expected oci=+100_000 (remainder to surplus AFTER reversal), got %d", step3.GetRecognizedInOci())
	}
	if step3.GetRevaluationSurplusBalance() != 100_000 {
		t.Errorf("step3 expected surplus=100_000 (rebuilt), got %d", step3.GetRevaluationSurplusBalance())
	}
}

// =============================================================================
// Phase 1.6 — workspace-required error string test (codex L1.5 + C1.5)
//
// RevalueAssetRequest has no WorkspaceId field, so there is no ctx-vs-req
// mismatch to check; the only workspace enforcement path is the ctx-empty
// rejection. This test proves:
//  1. The rejection fires (no workspace in ctx → error).
//  2. The error message does not leak the internal column name "workspace_id"
//     (operator-safe text since Phase 1.6).
//  3. No DB writes occur on the rejection path.
// =============================================================================

// TestRevalueAsset_RejectsEmptyWorkspaceInContext verifies that Execute returns
// an error when no workspace is present in the context, and that the error
// message is operator-safe (does not contain the raw DB column name).
func TestRevalueAsset_RejectsEmptyWorkspaceInContext(t *testing.T) {
	asset := newRevaluationAsset("asset-ws-test", "ws-1", 100_000)
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	revRepo := &fakeRevaluationRepo{}

	uc := newRevalueUseCaseWithRepos(assetRepo, txRepo, revRepo)

	// Deliberately omit workspace from context (no WithWorkspaceID call).
	res, err := uc.Execute(context.Background(), &revaluation_pb.RevalueAssetUseCaseRequest{
		AssetId:      "asset-ws-test",
		NewFairValue: 150_000,
	})
	if err == nil {
		t.Fatalf("expected workspace-required error, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result on workspace rejection, got %+v", res)
	}
	// Error must not expose the raw DB column name.
	if contains(err.Error(), "workspace_id") {
		t.Errorf("error message leaks DB column name 'workspace_id': %q", err.Error())
	}
	// Error must mention workspace in some operator-safe form.
	if !contains(err.Error(), "Workspace") && !contains(err.Error(), "workspace") {
		t.Errorf("expected error to mention workspace, got: %q", err.Error())
	}
	// No DB writes should have occurred.
	if len(revRepo.created) != 0 {
		t.Errorf("expected 0 asset_revaluation inserts, got %d", len(revRepo.created))
	}
	if len(txRepo.created) != 0 {
		t.Errorf("expected 0 asset_transaction inserts, got %d", len(txRepo.created))
	}
	if len(assetRepo.updates) != 0 {
		t.Errorf("expected 0 asset updates, got %d", len(assetRepo.updates))
	}
}

// =============================================================================
// Helper utilities
// =============================================================================

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
