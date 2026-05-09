package asset_revaluation_test

import (
	"testing"

	revaluationuc "github.com/erniealice/espyna-golang/internal/application/usecases/asset/asset_revaluation"
)

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
