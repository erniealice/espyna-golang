package depreciation_run

// Tests for GenerateDepreciationRunUseCase.
//
// Phase 2 (2026-05-10) — codex C1, C3, H1, H3 fix coverage:
//   - C1: per-period running balance — schedule rows, asset state, and the
//     declining-balance engine inputs all see the post-prior-period state, not
//     the original asset snapshot.
//   - C3: drop schedule-based pre-check — reruns surface as SKIPPED via the DB
//     unique index, not pre-filtered.
//   - H1: per-period writes are atomic — schedule failure rolls back the
//     transaction insert; the run continues to subsequent periods with the
//     prior period's running balance preserved.
//   - H3: empty period_start_dates for a selected asset means "all elapsed
//     periods", not "no periods" — Surface C/F drawer contract.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// ---------------------------------------------------------------------------
// In-memory mock repositories
// ---------------------------------------------------------------------------

// fakeAssetRepo — minimal AssetDomainServiceServer for use case tests.
// Only implements the methods the generate use case actually calls
// (ReadAsset, UpdateAsset, ListAssets). Other methods panic.
type fakeAssetRepo struct {
	assetpb.UnimplementedAssetDomainServiceServer
	byID    map[string]*assetpb.Asset
	updates []*assetpb.Asset
	listFn  func(ctx context.Context, req *assetpb.ListAssetsRequest) ([]*assetpb.Asset, error)
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
	// Apply the partial update to byID so subsequent reads reflect the new state.
	if a, ok := r.byID[req.GetData().GetId()]; ok {
		updated := proto.Clone(a).(*assetpb.Asset)
		updated.AccumulatedDepreciation = req.GetData().GetAccumulatedDepreciation()
		updated.BookValue = req.GetData().GetBookValue()
		r.byID[updated.GetId()] = updated
	}
	return &assetpb.UpdateAssetResponse{Data: []*assetpb.Asset{req.GetData()}}, nil
}

func (r *fakeAssetRepo) ListAssets(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
	if r.listFn != nil {
		assets, err := r.listFn(ctx, req)
		if err != nil {
			return nil, err
		}
		return &assetpb.ListAssetsResponse{Data: assets}, nil
	}
	// Default: return everything (callers filter via filterInService).
	var out []*assetpb.Asset
	for _, a := range r.byID {
		out = append(out, a)
	}
	return &assetpb.ListAssetsResponse{Data: out}, nil
}

// fakeAssetTransactionRepo records inserts and optionally returns errors per call.
type fakeAssetTransactionRepo struct {
	assettxpb.UnimplementedAssetTransactionDomainServiceServer
	created []*assettxpb.AssetTransaction
	// Per-asset, per-period unique-key set for idempotency simulation.
	seenKeys map[string]bool
	// errOnIndex — return this error on the Nth Create call (1-based). Allows
	// simulating mid-run failures.
	errOnIndex int
	errToYield error
	// nextIdx tracks the call count for errOnIndex matching.
	nextIdx int
}

func (r *fakeAssetTransactionRepo) CreateAssetTransaction(ctx context.Context, req *assettxpb.CreateAssetTransactionRequest) (*assettxpb.CreateAssetTransactionResponse, error) {
	r.nextIdx++
	if r.errOnIndex > 0 && r.nextIdx == r.errOnIndex && r.errToYield != nil {
		return nil, r.errToYield
	}
	tx := req.GetData()
	// Simulate the partial unique index on (asset_id, period_marker) WHERE
	// transaction_type='DEPRECIATION'. period_marker = period_start_date.
	if tx.GetTransactionType() == assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_DEPRECIATION {
		key := tx.GetAssetId() + "|" + tx.GetDepreciationPeriodStartDate()
		if r.seenKeys == nil {
			r.seenKeys = map[string]bool{}
		}
		if r.seenKeys[key] {
			return nil, errors.New("duplicate key value violates unique constraint \"idx_asset_transaction_depreciation_period\" (SQLSTATE 23505)")
		}
		r.seenKeys[key] = true
	}
	r.created = append(r.created, tx)
	return &assettxpb.CreateAssetTransactionResponse{Data: []*assettxpb.AssetTransaction{tx}}, nil
}

// fakeDepreciationScheduleRepo records inserts. Optionally fails on a specific
// CREATED-outcome insert (used by the rollback test) so we can prove that a
// mid-period schedule failure rolls back the transaction insert.
type fakeDepreciationScheduleRepo struct {
	depschpb.UnimplementedDepreciationDomainServiceServer
	created []*depschpb.DepreciationSchedule
	// failCreatedIndex — the Nth CREATED-outcome insert (1-based) returns errToYield.
	// SKIPPED/ERRORED audit inserts always succeed so the audit trail is preserved.
	failCreatedIndex   int
	errToYield         error
	createdInsertsSeen int
}

func (r *fakeDepreciationScheduleRepo) CreateDepreciationSchedule(ctx context.Context, req *depschpb.CreateDepreciationScheduleRequest) (*depschpb.CreateDepreciationScheduleResponse, error) {
	sch := req.GetData()
	outcome := ""
	if sch.Outcome != nil {
		outcome = *sch.Outcome
	}
	if outcome == deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED.String() {
		r.createdInsertsSeen++
		if r.failCreatedIndex > 0 && r.createdInsertsSeen == r.failCreatedIndex && r.errToYield != nil {
			return nil, r.errToYield
		}
	}
	r.created = append(r.created, sch)
	return &depschpb.CreateDepreciationScheduleResponse{Data: []*depschpb.DepreciationSchedule{sch}}, nil
}

func (r *fakeDepreciationScheduleRepo) ListDepreciationSchedules(ctx context.Context, req *depschpb.ListDepreciationSchedulesRequest) (*depschpb.ListDepreciationSchedulesResponse, error) {
	return &depschpb.ListDepreciationSchedulesResponse{Data: r.created}, nil
}

// fakeDepreciationRunRepo records the parent run lifecycle.
type fakeDepreciationRunRepo struct {
	deprunpb.UnimplementedDepreciationRunDomainServiceServer
	created []*deprunpb.DepreciationRun
	updates []*deprunpb.DepreciationRun
}

func (r *fakeDepreciationRunRepo) CreateDepreciationRun(ctx context.Context, req *deprunpb.CreateDepreciationRunRequest) (*deprunpb.CreateDepreciationRunResponse, error) {
	r.created = append(r.created, req.GetData())
	return &deprunpb.CreateDepreciationRunResponse{Data: []*deprunpb.DepreciationRun{req.GetData()}}, nil
}

func (r *fakeDepreciationRunRepo) UpdateDepreciationRun(ctx context.Context, req *deprunpb.UpdateDepreciationRunRequest) (*deprunpb.UpdateDepreciationRunResponse, error) {
	r.updates = append(r.updates, req.GetData())
	return &deprunpb.UpdateDepreciationRunResponse{Data: []*deprunpb.DepreciationRun{req.GetData()}}, nil
}

// fakeAssetCategoryRepo — empty stub; only needed because the use case struct
// names it in repositories. Not exercised by these tests.
type fakeAssetCategoryRepo struct {
	assetcategorypb.UnimplementedAssetCategoryDomainServiceServer
}

// We use ports.NewNoOpIDService() for tests — the IDs are not asserted on
// (assertions target money values + outcomes, which are deterministic).

func itoa(n int) string {
	// Manual int→string to avoid pulling in strconv/fmt only for tests.
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestUseCase(
	asset *fakeAssetRepo,
	tx *fakeAssetTransactionRepo,
	sch *fakeDepreciationScheduleRepo,
	run *fakeDepreciationRunRepo,
) *GenerateDepreciationRunUseCase {
	return NewGenerateDepreciationRunUseCase(
		GenerateDepreciationRunRepositories{
			Asset:                asset,
			AssetCategory:        &fakeAssetCategoryRepo{},
			AssetTransaction:     tx,
			DepreciationSchedule: sch,
			DepreciationRun:      run,
		},
		GenerateDepreciationRunServices{
			AuthorizationService: ports.NewNoOpAuthorizationService(),
			TransactionService:   ports.NewNoOpTransactionService(),
			TranslationService:   ports.NewNoOpTranslationService(),
			IDService:            ports.NewNoOpIDService(),
		},
	)
}

// newSLAsset creates a STRAIGHT_LINE asset with `cost=12000, salvage=0,
// useful_life=12` and depreciation_start_date=2025-01-01. Per-period SL amount
// is 1000 centavos.
func newSLAsset(id, workspaceID string) *assetpb.Asset {
	ws := workspaceID
	startDate := "2025-01-01"
	return &assetpb.Asset{
		Id:                      id,
		WorkspaceId:             &ws,
		Name:                    "TEST-SL-" + id,
		AcquisitionCost:         12000,
		SalvageValue:            0,
		UsefulLifeMonths:        12,
		DepreciationMethod:      assetpb.DepreciationMethod_DEPRECIATION_METHOD_STRAIGHT_LINE,
		DepreciationStartDate:   &startDate,
		BookValue:               12000,
		AccumulatedDepreciation: 0,
		Status:                  assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE,
		Active:                  true,
	}
}

// newDDBAsset creates a DOUBLE_DECLINING_BALANCE asset for the DDB running-
// balance test. Per the engine, monthly rate = 2/12 / 12 ≈ 0.0139.
// Period 1: BV=12000, amount = round(12000 * 2/144) = round(166.66) = 167.
// Period 2: BV=11833, amount = round(11833 * 2/144) = round(164.35) = 164.
// (We assert period 2's amount uses BV after period 1, not the original 12000.)
func newDDBAsset(id, workspaceID string) *assetpb.Asset {
	ws := workspaceID
	startDate := "2025-01-01"
	return &assetpb.Asset{
		Id:                      id,
		WorkspaceId:             &ws,
		Name:                    "TEST-DDB-" + id,
		AcquisitionCost:         12000,
		SalvageValue:            0,
		UsefulLifeMonths:        12,
		DepreciationMethod:      assetpb.DepreciationMethod_DEPRECIATION_METHOD_DOUBLE_DECLINING_BALANCE,
		DepreciationStartDate:   &startDate,
		BookValue:               12000,
		AccumulatedDepreciation: 0,
		Status:                  assetpb.AssetStatus_ASSET_STATUS_IN_SERVICE,
		Active:                  true,
	}
}

func ctxWithWorkspace(workspaceID string) context.Context {
	return context.Background()
}

// ---------------------------------------------------------------------------
// Test 1 — C1: 12-period SL run posts 12 transactions and zeroes book_value
// ---------------------------------------------------------------------------

func TestGenerate_StraightLine_TwelvePeriods_RunningBalance(t *testing.T) {
	asset := newSLAsset("asset-sl", "ws-1")
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	schRepo := &fakeDepreciationScheduleRepo{}
	runRepo := &fakeDepreciationRunRepo{}

	uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

	scopeID := asset.GetId()
	res, err := uc.Execute(ctxWithWorkspace("ws-1"), &deprunpb.GenerateDepreciationRunRequest{
		WorkspaceId: "ws-1",
		ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET,
		ScopeId:     &scopeID,
		AsOfDate:    "2026-01-15", // through Dec 2025 — 12 elapsed full months
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// 12 transactions, each amount = 1000 centavos.
	if len(txRepo.created) != 12 {
		t.Fatalf("expected 12 asset_transaction inserts, got %d", len(txRepo.created))
	}
	for i, tx := range txRepo.created {
		if tx.GetAmount() != 1000 {
			t.Errorf("period %d: expected amount=1000, got %d", i+1, tx.GetAmount())
		}
		if tx.GetTransactionType() != assettxpb.AssetTransactionType_ASSET_TRANSACTION_TYPE_DEPRECIATION {
			t.Errorf("period %d: expected DEPRECIATION type, got %v", i+1, tx.GetTransactionType())
		}
	}

	// Final asset state: BV=0, accumulated=12000.
	if len(assetRepo.updates) != 1 {
		t.Fatalf("expected exactly 1 asset update (batched at end of asset), got %d", len(assetRepo.updates))
	}
	finalUpdate := assetRepo.updates[0]
	if finalUpdate.GetBookValue() != 0 {
		t.Errorf("expected final book_value=0, got %d", finalUpdate.GetBookValue())
	}
	if finalUpdate.GetAccumulatedDepreciation() != 12000 {
		t.Errorf("expected final accumulated=12000, got %d", finalUpdate.GetAccumulatedDepreciation())
	}

	// 12 schedule rows; closing values are 11000, 10000, ..., 0.
	creatdSchRows := filterCreatedSchedules(schRepo.created)
	if len(creatdSchRows) != 12 {
		t.Fatalf("expected 12 CREATED schedule rows, got %d", len(creatdSchRows))
	}
	expected := []int64{11000, 10000, 9000, 8000, 7000, 6000, 5000, 4000, 3000, 2000, 1000, 0}
	for i, sch := range creatdSchRows {
		if sch.GetClosingBookValue() != expected[i] {
			t.Errorf("period %d: expected closing_book_value=%d, got %d",
				i+1, expected[i], sch.GetClosingBookValue())
		}
		// Opening for period N is the closing for period N-1 (= 12000 - i*1000).
		expectedOpening := int64(12000 - int64(i)*1000)
		if sch.GetOpeningBookValue() != expectedOpening {
			t.Errorf("period %d: expected opening_book_value=%d, got %d",
				i+1, expectedOpening, sch.GetOpeningBookValue())
		}
		// Accumulated after period N = (i+1)*1000.
		if sch.GetAccumulatedDepreciation() != int64(i+1)*1000 {
			t.Errorf("period %d: expected accumulated=%d, got %d",
				i+1, int64(i+1)*1000, sch.GetAccumulatedDepreciation())
		}
	}

	// Run counts.
	if res.CreatedCount != 12 || res.SkippedCount != 0 || res.ErroredCount != 0 {
		t.Errorf("expected created=12,skipped=0,errored=0; got created=%d,skipped=%d,errored=%d",
			res.CreatedCount, res.SkippedCount, res.ErroredCount)
	}
}

// ---------------------------------------------------------------------------
// Test 2 — C1: DDB run uses prior period's closing BV, not original cost
// ---------------------------------------------------------------------------

func TestGenerate_DoubleDecliningBalance_RunningBalanceFedToEngine(t *testing.T) {
	asset := newDDBAsset("asset-ddb", "ws-1")
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	schRepo := &fakeDepreciationScheduleRepo{}
	runRepo := &fakeDepreciationRunRepo{}

	uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

	scopeID := asset.GetId()
	res, err := uc.Execute(ctxWithWorkspace("ws-1"), &deprunpb.GenerateDepreciationRunRequest{
		WorkspaceId: "ws-1",
		ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET,
		ScopeId:     &scopeID,
		AsOfDate:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.CreatedCount == 0 {
		t.Fatalf("expected non-zero CreatedCount, got 0")
	}

	// We do not pin exact engine output (rounding can shift a centavo) — we assert
	// the structural property: each period's amount is computed against the
	// PRIOR period's closing BV. Concretely, amounts are monotone non-increasing
	// (declining balance), and period 2's amount is STRICTLY less than period 1's
	// amount when both are positive — which can only be true if period 2 sees the
	// reduced book value.
	if len(txRepo.created) < 2 {
		t.Fatalf("expected at least 2 periods to compare DDB amounts, got %d", len(txRepo.created))
	}
	first := txRepo.created[0].GetAmount()
	second := txRepo.created[1].GetAmount()
	if first <= 0 {
		t.Fatalf("period 1 amount must be positive (DDB), got %d", first)
	}
	if second >= first {
		t.Errorf("DDB period 2 amount (%d) must be < period 1 amount (%d) — proves engine sees prior BV, not original cost",
			second, first)
	}

	// Schedule rows must reflect the running balance: opening of period N
	// equals closing of period N-1.
	creatdSchRows := filterCreatedSchedules(schRepo.created)
	for i := 1; i < len(creatdSchRows); i++ {
		prevClose := creatdSchRows[i-1].GetClosingBookValue()
		thisOpen := creatdSchRows[i].GetOpeningBookValue()
		if prevClose != thisOpen {
			t.Errorf("DDB chain break: period %d closing=%d, period %d opening=%d (must match)",
				i, prevClose, i+1, thisOpen)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3 — C3: idempotent rerun yields all-SKIPPED
// ---------------------------------------------------------------------------

func TestGenerate_IdempotentRerun_AllSkipped(t *testing.T) {
	asset := newSLAsset("asset-rerun", "ws-1")
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	schRepo := &fakeDepreciationScheduleRepo{}
	runRepo := &fakeDepreciationRunRepo{}

	uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

	scopeID := asset.GetId()
	req := &deprunpb.GenerateDepreciationRunRequest{
		WorkspaceId: "ws-1",
		ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET,
		ScopeId:     &scopeID,
		AsOfDate:    "2026-01-15",
	}

	first, err := uc.Execute(ctxWithWorkspace("ws-1"), req)
	if err != nil {
		t.Fatalf("first Execute error: %v", err)
	}
	if first.CreatedCount != 12 {
		t.Fatalf("first run: expected created=12, got %d", first.CreatedCount)
	}

	// Re-run with the same as_of_date. The fake AssetTransaction repo simulates
	// the partial unique index and returns a 23505-style error for every period.
	second, err := uc.Execute(ctxWithWorkspace("ws-1"), req)
	if err != nil {
		t.Fatalf("second Execute error: %v", err)
	}
	if second.CreatedCount != 0 {
		t.Errorf("rerun: expected created=0 (all already posted), got %d", second.CreatedCount)
	}
	if second.SkippedCount != 12 {
		t.Errorf("rerun: expected skipped=12 (idempotent unique-index re-runs), got %d",
			second.SkippedCount)
	}
	if second.ErroredCount != 0 {
		t.Errorf("rerun: expected errored=0, got %d", second.ErroredCount)
	}

	// Sanity: no duplicate asset_transaction inserts.
	if len(txRepo.created) != 12 {
		t.Errorf("expected 12 asset_transaction rows total across both runs, got %d",
			len(txRepo.created))
	}
}

// ---------------------------------------------------------------------------
// Test 4 — H1: schedule failure on period 5 rolls back, run continues
// ---------------------------------------------------------------------------

func TestGenerate_SchedulePersistFailureOnPeriod5_RollsBackAndContinues(t *testing.T) {
	asset := newSLAsset("asset-rollback", "ws-1")
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	// schRepo fails the 5th CREATED-outcome insert. The audit row for the
	// resulting ERRORED outcome is ALSO routed through CreateDepreciationSchedule
	// but with outcome != CREATED, so it is not affected by failCreatedIndex.
	txRepo := &fakeAssetTransactionRepo{}
	schRepo := &fakeDepreciationScheduleRepo{
		failCreatedIndex: 5,
		errToYield:       errors.New("simulated schedule write failure on period 5"),
	}
	runRepo := &fakeDepreciationRunRepo{}

	uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

	scopeID := asset.GetId()
	res, err := uc.Execute(ctxWithWorkspace("ws-1"), &deprunpb.GenerateDepreciationRunRequest{
		WorkspaceId: "ws-1",
		ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_ASSET,
		ScopeId:     &scopeID,
		AsOfDate:    "2026-01-15",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Expectation: 11 CREATED (periods 1-4 + 6-12) + 1 ERRORED (period 5) + 0 SKIPPED.
	if res.CreatedCount != 11 {
		t.Errorf("expected created=11 (12 - 1 failed), got %d", res.CreatedCount)
	}
	if res.ErroredCount != 1 {
		t.Errorf("expected errored=1 (period 5 schedule failed), got %d", res.ErroredCount)
	}
	if res.SkippedCount != 0 {
		t.Errorf("expected skipped=0, got %d", res.SkippedCount)
	}

	// Note on tx count: the NoOp TransactionService does NOT actually roll back
	// the asset_transaction insert when the schedule insert fails inside the
	// closure (NoOp executes inline, no real BEGIN/ROLLBACK). With a real
	// PostgreSQL TransactionService the 5th asset_transaction would be rolled
	// back. The test here exercises the use case's CONTROL FLOW: ERRORED
	// outcome, run continues, running balance preserved. A live-DB integration
	// test (Phase 2 acceptance gate) covers the actual rollback.

	// The CRITICAL property to assert: running balance was preserved across the
	// failed period. Period 6's schedule row (the next CREATED after the failure)
	// must have an opening_book_value equal to period 4's closing (8000), NOT
	// period 5's projected closing (7000) — because period 5 did NOT advance
	// the running balance.
	creatdSchRows := filterCreatedSchedules(schRepo.created)
	if len(creatdSchRows) != 11 {
		t.Fatalf("expected 11 CREATED schedule rows, got %d", len(creatdSchRows))
	}
	// Periods 1..4 (indices 0..3) closing should be 11000, 10000, 9000, 8000.
	if creatdSchRows[3].GetClosingBookValue() != 8000 {
		t.Errorf("period 4 closing: expected 8000, got %d", creatdSchRows[3].GetClosingBookValue())
	}
	// Period 6 (index 4 in CREATED list) opening must equal period 4's closing.
	period6Opening := creatdSchRows[4].GetOpeningBookValue()
	if period6Opening != 8000 {
		t.Errorf("period 6 opening: expected 8000 (period 4 closing — period 5 must NOT have advanced running balance), got %d",
			period6Opening)
	}
	// Period 6 amount = 1000 (SL), so closing = 7000.
	if creatdSchRows[4].GetClosingBookValue() != 7000 {
		t.Errorf("period 6 closing: expected 7000, got %d", creatdSchRows[4].GetClosingBookValue())
	}

	// Period 5 must be recorded as ERRORED in the audit table.
	var erroredRows []*depschpb.DepreciationSchedule
	for _, sch := range schRepo.created {
		if sch.Outcome != nil && *sch.Outcome == deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_ERRORED.String() {
			erroredRows = append(erroredRows, sch)
		}
	}
	if len(erroredRows) != 1 {
		t.Fatalf("expected 1 ERRORED schedule row for period 5, got %d", len(erroredRows))
	}
	if erroredRows[0].ErrorMessage == nil || !strings.Contains(*erroredRows[0].ErrorMessage, "simulated schedule write failure") {
		t.Errorf("ERRORED row should carry the simulated error message, got %v", erroredRows[0].ErrorMessage)
	}

	// Final asset state must reflect 11 successful periods (accumulated=11000, BV=1000),
	// NOT 12 (which would imply we kept advancing through the failed period).
	if len(assetRepo.updates) != 1 {
		t.Fatalf("expected 1 asset update at end of asset, got %d", len(assetRepo.updates))
	}
	finalUpdate := assetRepo.updates[0]
	if finalUpdate.GetAccumulatedDepreciation() != 11000 {
		t.Errorf("expected accumulated=11000 (11 successful periods), got %d",
			finalUpdate.GetAccumulatedDepreciation())
	}
	if finalUpdate.GetBookValue() != 1000 {
		t.Errorf("expected book_value=1000 (12000 - 11*1000), got %d", finalUpdate.GetBookValue())
	}
}

// ---------------------------------------------------------------------------
// Test 5 — H3: selection semantics (empty period list = all elapsed)
// ---------------------------------------------------------------------------

func TestGenerate_SelectionSemantics_EmptyPeriodListMeansAll(t *testing.T) {
	// 5 in-service assets in workspace ws-1, all with depreciation_start_date
	// 2025-09-01 → as_of 2026-01-15 → 4 elapsed full months (Sep, Oct, Nov, Dec).
	const elapsedMonths = 4
	assets := make([]*assetpb.Asset, 5)
	byID := map[string]*assetpb.Asset{}
	startDate := "2025-09-01"
	for i := 0; i < 5; i++ {
		a := newSLAsset("asset-"+itoa(i+1), "ws-1")
		a.DepreciationStartDate = &startDate
		assets[i] = a
		byID[a.GetId()] = a
	}

	tcases := []struct {
		name           string
		selections     []*deprunpb.DepreciationRunSelection
		expectCreated  int32
		expectAssetSet map[string]bool
	}{
		{
			name:           "no selections → all 5 assets × 4 periods = 20 entries",
			selections:     nil,
			expectCreated:  20,
			expectAssetSet: map[string]bool{"asset-1": true, "asset-2": true, "asset-3": true, "asset-4": true, "asset-5": true},
		},
		{
			name: "3 selected assets with EMPTY period_start_dates → 3 × 4 = 12 entries",
			selections: []*deprunpb.DepreciationRunSelection{
				{AssetId: "asset-1", PeriodStartDates: nil},
				{AssetId: "asset-2", PeriodStartDates: nil},
				{AssetId: "asset-3", PeriodStartDates: nil},
			},
			expectCreated:  3 * elapsedMonths,
			expectAssetSet: map[string]bool{"asset-1": true, "asset-2": true, "asset-3": true},
		},
		{
			name: "1 selected asset with EXPLICIT 2-period list → 2 entries",
			selections: []*deprunpb.DepreciationRunSelection{
				{AssetId: "asset-4", PeriodStartDates: []string{"2025-09-01", "2025-10-01"}},
			},
			expectCreated:  2,
			expectAssetSet: map[string]bool{"asset-4": true},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			// Fresh repos per subtest (the asset is workspace-listed).
			fresh := map[string]*assetpb.Asset{}
			for k, v := range byID {
				fresh[k] = proto.Clone(v).(*assetpb.Asset)
			}
			assetRepo := &fakeAssetRepo{byID: fresh}
			txRepo := &fakeAssetTransactionRepo{}
			schRepo := &fakeDepreciationScheduleRepo{}
			runRepo := &fakeDepreciationRunRepo{}

			uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

			res, err := uc.Execute(ctxWithWorkspace("ws-1"), &deprunpb.GenerateDepreciationRunRequest{
				WorkspaceId: "ws-1",
				ScopeKind:   deprunpb.DepreciationRunScopeKind_DEPRECIATION_RUN_SCOPE_KIND_WORKSPACE,
				AsOfDate:    "2026-01-15",
				Selections:  tc.selections,
			})
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if res.CreatedCount != tc.expectCreated {
				t.Errorf("expected created=%d, got %d", tc.expectCreated, res.CreatedCount)
			}
			// Verify all created transactions belong to assets in expectAssetSet.
			gotAssets := map[string]int{}
			for _, tx := range txRepo.created {
				gotAssets[tx.GetAssetId()]++
			}
			for assetID := range tc.expectAssetSet {
				if gotAssets[assetID] == 0 {
					t.Errorf("expected transactions for asset %q, got none", assetID)
				}
			}
			for assetID := range gotAssets {
				if !tc.expectAssetSet[assetID] {
					t.Errorf("unexpected transactions for asset %q (not in selection)", assetID)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 1.6 — mismatch rejection test (codex C1.5)
// ---------------------------------------------------------------------------

// TestGenerate_TenancyMismatch_RejectsBeforeDBWrites verifies that when the
// authenticated context carries workspace "ws-A" and the request claims
// workspace "ws-B", Execute returns a tenancy-mismatch error and writes
// nothing to any repository.
func TestGenerate_TenancyMismatch_RejectsBeforeDBWrites(t *testing.T) {
	asset := newSLAsset("asset-1", "ws-A")
	assetRepo := &fakeAssetRepo{byID: map[string]*assetpb.Asset{asset.GetId(): asset}}
	txRepo := &fakeAssetTransactionRepo{}
	schRepo := &fakeDepreciationScheduleRepo{}
	runRepo := &fakeDepreciationRunRepo{}

	uc := newTestUseCase(assetRepo, txRepo, schRepo, runRepo)

	// Context is authenticated as ws-A; request claims ws-B.
	ctx := contextutil.WithWorkspaceID(context.Background(), "ws-A")
	req := &deprunpb.GenerateDepreciationRunRequest{
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
	// No DB writes should have occurred.
	if len(txRepo.created) != 0 {
		t.Errorf("expected 0 asset_transaction inserts, got %d", len(txRepo.created))
	}
	if len(schRepo.created) != 0 {
		t.Errorf("expected 0 depreciation_schedule inserts, got %d", len(schRepo.created))
	}
	if len(runRepo.created) != 0 {
		t.Errorf("expected 0 depreciation_run inserts, got %d", len(runRepo.created))
	}
}

// ---------------------------------------------------------------------------
// Helpers (test-only)
// ---------------------------------------------------------------------------

// filterCreatedSchedules returns only schedule rows whose outcome is CREATED,
// preserving insertion order. Tests use this to walk the running-balance chain
// without mixing in SKIPPED/ERRORED audit rows.
func filterCreatedSchedules(rows []*depschpb.DepreciationSchedule) []*depschpb.DepreciationSchedule {
	var out []*depschpb.DepreciationSchedule
	for _, sch := range rows {
		if sch.Outcome != nil && *sch.Outcome == deprunpb.DepreciationRunOutcome_DEPRECIATION_RUN_OUTCOME_CREATED.String() {
			out = append(out, sch)
		}
	}
	return out
}
