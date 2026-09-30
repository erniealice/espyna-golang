//go:build postgresql

package ledger

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

// C4/C9 sentinel (F7): a missing or foreign-workspace id makes the charge_policy_version locker wrap
// domainports.ErrLockedRowNotFound, so use cases classify it without matching message text.
func TestChargePolicyVersionLockerReportsSentinelForMissingRow(t *testing.T) {
	h := scopetest.New(t, "charge_policy_version")
	repo := NewPostgresChargePolicyVersionRepository(h.Ops, entityid.ChargePolicyVersion).(*PostgresChargePolicyVersionRepository)
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.LockChargePolicyVersionForUpdate(ctxA, "f7-missing-charge_policy_version"); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Fatalf("missing row: want ErrLockedRowNotFound, got %v", err)
		}
	})
}
