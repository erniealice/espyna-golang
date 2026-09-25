//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// Integration coverage for codex-review-impl3.out.md finding #3 ("the new
// offering picker exposes cross-workspace offering IDs/names") on the fix3
// brief's prescribed lane (education2clone20260925a via TEST_DATABASE_URL,
// rolled back — no residue). Mirrors the rating_description_set_product_plan
// package's W3-lifecycle integration harness (openW3LifecycleIntegrationDB /
// tm.RunInTransaction / seedW3LifecycleOffering), reimplemented locally
// because these are unexported helpers in a different adapter package.

func openProductPlanIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var ok string
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('product_plan')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("product_plan not present")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func ppScopeExecutor(ctx context.Context, ops interfaces.DatabaseOperation) (sqlexec.DBExecutor, error) {
	ex, ok := ops.(interface {
		GetExecutor(context.Context) sqlexec.DBExecutor
	})
	if !ok {
		return nil, fmt.Errorf("ops does not expose GetExecutor")
	}
	return ex.GetExecutor(ctx), nil
}

func seedPPScopeWorkspace(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, id string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, id); err != nil {
		t.Fatalf("seed workspace %s: %v", id, err)
	}
}

// seedPPScopeOffering seeds a product+plan+product_plan chain scoped to the
// given workspace (product.workspace_id AND plan.workspace_id both set —
// product_plan itself carries no workspace column).
func seedPPScopeOffering(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, productID, planID, productPlanID, workspaceID string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO product (id, workspace_id, name) VALUES ($1, $2, 'PPScope Product')`, productID, workspaceID); err != nil {
		t.Fatalf("seed product %s: %v", productID, err)
	}
	if _, err := ex.ExecContext(ctx, `INSERT INTO plan (id, workspace_id, name) VALUES ($1, $2, 'PPScope Plan')`, planID, workspaceID); err != nil {
		t.Fatalf("seed plan %s: %v", planID, err)
	}
	if _, err := ex.ExecContext(ctx, `INSERT INTO product_plan (id, product_id, plan_id, name, active) VALUES ($1, $2, $3, 'PPScope Offering', true)`, productPlanID, productID, planID); err != nil {
		t.Fatalf("seed product_plan %s: %v", productPlanID, err)
	}
}

// TestIntegration_ListWorkspaceScopedProductPlans_ExcludesForeignWorkspace is
// the brief's required cross-workspace picker test: create a foreign-
// workspace product + plan + product_plan inside the tx and assert it is
// NOT returned to the caller's own workspace.
func TestIntegration_ListWorkspaceScopedProductPlans_ExcludesForeignWorkspace(t *testing.T) {
	db := openProductPlanIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo, ok := NewPostgresProductPlanRepository(wsOps, "product_plan").(*PostgresProductPlanRepository)
	if !ok {
		t.Fatal("NewPostgresProductPlanRepository did not return *PostgresProductPlanRepository")
	}

	const wsOwn = "ppscope-own-ws"
	const wsForeign = "ppscope-foreign-ws"
	const ppOwn = "ppscope-own-pp"
	const ppForeign = "ppscope-foreign-pp"

	rollbackMsg := "ppscope-int: intentional rollback"
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := ppScopeExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedPPScopeWorkspace(t, ex, txCtx, wsOwn)
		seedPPScopeWorkspace(t, ex, txCtx, wsForeign)
		seedPPScopeOffering(t, ex, txCtx, "ppscope-own-product", "ppscope-own-plan", ppOwn, wsOwn)
		seedPPScopeOffering(t, ex, txCtx, "ppscope-foreign-product", "ppscope-foreign-plan", ppForeign, wsForeign)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "ppscope-user", WorkspaceID: wsOwn})
		items, listErr := repo.ListWorkspaceScopedProductPlans(ctx)
		if listErr != nil {
			return fmt.Errorf("ListWorkspaceScopedProductPlans: %w", listErr)
		}

		foundOwn, foundForeign := false, false
		for _, pp := range items {
			if pp.GetId() == ppOwn {
				foundOwn = true
			}
			if pp.GetId() == ppForeign {
				foundForeign = true
			}
		}
		if !foundOwn {
			return fmt.Errorf("expected the OWN workspace's product_plan %q to be returned, got %d rows", ppOwn, len(items))
		}
		if foundForeign {
			return fmt.Errorf("SECURITY: foreign workspace's product_plan %q was returned to workspace %q", ppForeign, wsOwn)
		}
		return fmt.Errorf("%s", rollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), rollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// TestIntegration_ListWorkspaceScopedProductPlans_FailsClosedWithoutWorkspace
// proves the port never falls back to an unscoped list when the caller's
// trusted workspace cannot be resolved from context.
func TestIntegration_ListWorkspaceScopedProductPlans_FailsClosedWithoutWorkspace(t *testing.T) {
	db := openProductPlanIntegrationDB(t)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo, ok := NewPostgresProductPlanRepository(wsOps, "product_plan").(*PostgresProductPlanRepository)
	if !ok {
		t.Fatal("NewPostgresProductPlanRepository did not return *PostgresProductPlanRepository")
	}

	if _, err := repo.ListWorkspaceScopedProductPlans(context.Background()); err == nil {
		t.Fatal("expected an error when no trusted workspace is present in context")
	}
}
