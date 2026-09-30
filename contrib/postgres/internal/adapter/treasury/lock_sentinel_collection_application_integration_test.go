//go:build postgresql

package treasury

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

// C4/C9 sentinel (F7): a missing or foreign-workspace id makes the collection_application locker wrap
// domainports.ErrLockedRowNotFound, so use cases classify it without matching message text.
func TestCollectionApplicationLockerReportsSentinelForMissingRow(t *testing.T) {
	h := scopetest.New(t, "collection_application")
	repo := NewPostgresCollectionApplicationRepository(h.Ops, entityid.CollectionApplication).(*PostgresCollectionApplicationRepository)
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.LockCollectionApplicationForUpdate(ctxA, "f7-missing-collection_application"); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Fatalf("missing row: want ErrLockedRowNotFound, got %v", err)
		}
	})
}
