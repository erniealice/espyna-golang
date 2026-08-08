//go:build postgresql

package procurement

import (
	"context"
	"reflect"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
)

func TestSearchSupplierPlansByNameRejectsOversizeLimitBeforeDatabaseAccess(t *testing.T) {
	limit := int32(101)
	_, err := (&PostgresSupplierPlanRepository{}).SearchSupplierPlansByName(context.Background(), &supplierplanpb.SearchSupplierPlansByNameRequest{Limit: &limit})
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("SearchSupplierPlansByName() error = %v, want bounded limit rejection", err)
	}
}

func TestSearchSupplierPlansByNameRequiresSelectedWorkspace(t *testing.T) {
	repo := &PostgresSupplierPlanRepository{dbOps: postgresCore.NewWorkspaceAwareOperations(nil)}
	_, err := repo.SearchSupplierPlansByName(context.Background(), &supplierplanpb.SearchSupplierPlansByNameRequest{})
	if err == nil || !strings.Contains(err.Error(), "workspace selection is required") {
		t.Fatalf("SearchSupplierPlansByName() error = %v, want missing workspace rejection", err)
	}
}

func TestSupplierPlanPageDataRequiresSelectedWorkspace(t *testing.T) {
	repo := &PostgresSupplierPlanRepository{dbOps: postgresCore.NewWorkspaceAwareOperations(nil)}

	_, listErr := repo.GetSupplierPlanListPageData(context.Background(), &supplierplanpb.GetSupplierPlanListPageDataRequest{})
	if listErr == nil || !strings.Contains(listErr.Error(), "workspace selection is required") {
		t.Fatalf("GetSupplierPlanListPageData() error = %v, want missing workspace rejection", listErr)
	}

	_, itemErr := repo.GetSupplierPlanItemPageData(context.Background(), &supplierplanpb.GetSupplierPlanItemPageDataRequest{SupplierPlanId: "supplier-plan-1"})
	if itemErr == nil || !strings.Contains(itemErr.Error(), "workspace selection is required") {
		t.Fatalf("GetSupplierPlanItemPageData() error = %v, want missing workspace rejection", itemErr)
	}
}

func TestSupplierPlanPageQueriesUseStrictWorkspacePredicates(t *testing.T) {
	listSQL := supplierPlanListPageSQL("ORDER BY date_created DESC")
	if !strings.Contains(listSQL, "workspace_id = $4") {
		t.Fatalf("list SQL missing strict workspace predicate:\n%s", listSQL)
	}
	if strings.Contains(listSQL, "OR workspace_id") || strings.Contains(listSQL, "$4::text = ''") {
		t.Fatalf("list SQL contains nullable workspace bypass:\n%s", listSQL)
	}
	if got := strings.Count(listSQL, `ESCAPE '\'`); got != 2 {
		t.Fatalf("list SQL ESCAPE clause count = %d, want 2:\n%s", got, listSQL)
	}

	if !strings.Contains(supplierPlanItemPageSQL, "workspace_id = $2") {
		t.Fatalf("item SQL missing strict workspace predicate:\n%s", supplierPlanItemPageSQL)
	}
	if strings.Contains(supplierPlanItemPageSQL, "OR workspace_id") || strings.Contains(supplierPlanItemPageSQL, "$2::text = ''") {
		t.Fatalf("item SQL contains nullable workspace bypass:\n%s", supplierPlanItemPageSQL)
	}
}

func TestSupplierPlanRepositoryHasNoRawDatabaseField(t *testing.T) {
	if _, ok := reflect.TypeOf(PostgresSupplierPlanRepository{}).FieldByName("db"); ok {
		t.Fatal("PostgresSupplierPlanRepository must route raw SQL through its transaction-aware capability, not a cached *sql.DB")
	}
}
