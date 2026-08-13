//go:build firestore

package consumer

import (
	"testing"

	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry"
)

func TestFirestoreDatabaseTagRegistersLandingWithoutDetailedExport(t *testing.T) {
	if _, ok := registry.GetSubscriptionGroupOutcomeLandingFactory(); !ok {
		t.Fatal("firestore tag did not register the subscription-group outcome landing factory")
	}
	if _, ok := internalregistry.GetSubscriptionGroupOutcomeExportFactory(); ok {
		t.Fatal("firestore tag registered the PostgreSQL-only detailed outcome export factory")
	}
}
