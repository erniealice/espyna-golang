//go:build sqlserver

package sqlserver_test

import (
	"testing"

	_ "github.com/erniealice/espyna-golang/contrib/sqlserver"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

func TestExistingDomainFactoriesAreReachable(t *testing.T) {
	for _, entity := range []string{entityid.Account, entityid.ProductPlan, entityid.SupplierContract, entityid.Subscription} {
		if _, ok := registry.GetRepositoryFactory("sqlserver", entity); !ok {
			t.Errorf("sqlserver factory for %s was not imported", entity)
		}
	}
}
