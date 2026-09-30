//go:build postgresql

package revenue

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

// C4/C9 sentinel (F7): a missing or foreign-workspace id makes the document_series locker wrap
// domainports.ErrLockedRowNotFound, so use cases classify it without matching message text.
func TestDocumentSeriesLockerReportsSentinelForMissingRow(t *testing.T) {
	h := scopetest.New(t, "document_series")
	repo := NewPostgresDocumentSeriesRepository(h.Ops, entityid.DocumentSeries).(*PostgresDocumentSeriesRepository)
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.LockDocumentSeriesForUpdate(ctxA, "f7-missing-document_series"); !errors.Is(err, domainports.ErrLockedRowNotFound) {
			t.Fatalf("missing row: want ErrLockedRowNotFound, got %v", err)
		}
	})
}
