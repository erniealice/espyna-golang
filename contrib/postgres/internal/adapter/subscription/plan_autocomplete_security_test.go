//go:build postgresql

package subscription

import (
	"context"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

func TestSearchPlansByNameRejectsOversizeLimitBeforeDatabaseAccess(t *testing.T) {
	limit := int32(101)
	_, err := (&PostgresPlanRepository{}).SearchPlansByName(context.Background(), &planpb.SearchPlansByNameRequest{Limit: &limit})
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("SearchPlansByName() error = %v, want bounded limit rejection", err)
	}
}

func TestSearchPlansByNameRequiresSelectedWorkspace(t *testing.T) {
	repo := &PostgresPlanRepository{dbOps: postgresCore.NewWorkspaceAwareOperations(nil)}
	_, err := repo.SearchPlansByName(context.Background(), &planpb.SearchPlansByNameRequest{})
	if err == nil || !strings.Contains(err.Error(), "workspace selection is required") {
		t.Fatalf("SearchPlansByName() error = %v, want missing workspace rejection", err)
	}
}
