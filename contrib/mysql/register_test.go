//go:build mysql

package mysql_test

import (
	"testing"

	_ "github.com/erniealice/espyna-golang/contrib/mysql"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

func TestExistingDomainFactoriesAreReachable(t *testing.T) {
	for _, entity := range []string{entityid.Account, entityid.Expenditure, entityid.RevenueRun, entityid.TaxRate} {
		if _, ok := registry.GetRepositoryFactory("mysql", entity); !ok {
			t.Errorf("mysql factory for %s was not imported", entity)
		}
	}
}
