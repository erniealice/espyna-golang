//go:build postgresql

package core

import (
	"context"
	"errors"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/model"
)

func requireDatabaseErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected database error %q, got nil", want)
	}
	dbErr, ok := err.(*model.DatabaseError)
	if !ok {
		t.Fatalf("expected *model.DatabaseError, got %T (%v)", err, err)
	}
	if dbErr.Code != want {
		t.Fatalf("database error code = %q, want %q (%v)", dbErr.Code, want, err)
	}
}

func TestWorkspaceScopePolicyRegistryCoversLockedSchoolTables(t *testing.T) {
	direct := []string{
		entityid.JobPhase,
		entityid.JobTask,
		entityid.JobTemplatePhase,
		entityid.JobTemplateTask,
		entityid.JobTemplateRelation,
		entityid.CriteriaOption,
		entityid.CriteriaThreshold,
		entityid.TemplateTaskCriteria,
		entityid.TaskOutcome,
		entityid.TaskOutcomeCheck,
		entityid.PhaseOutcomeSummary,
		entityid.ScoringComponent,
		entityid.SubscriptionGroupDocumentTemplate,
	}
	for _, tableName := range direct {
		if got := workspaceScopePolicies[tableName]; got != workspaceScopeDirectRequired {
			t.Errorf("policy[%q] = %v, want direct-required", tableName, got)
		}
	}

	for _, tableName := range []string{entityid.TreasuryCollection, entityid.TreasuryDisbursement} {
		if got := workspaceScopePolicies[tableName]; got != workspaceScopeDirectRequired {
			t.Errorf("policy[%q] = %v, want direct-required", tableName, got)
		}
	}
}

func TestDirectRequiredTableRejectsMissingIdentityAndSchema(t *testing.T) {
	inner := &stubInner{}
	w := newStubWorkspaceOps(inner, false)
	w.columnCache[entityid.JobTask] = map[string]bool{}

	_, err := w.List(context.Background(), entityid.JobTask, nil)
	requireDatabaseErrorCode(t, err, "WORKSPACE_REQUIRED")

	_, err = w.List(newCtxWithWorkspace("ws-1"), entityid.JobTask, nil)
	requireDatabaseErrorCode(t, err, "TENANT_SCOPE_UNAVAILABLE")
}

func TestDirectRequiredTableScopesListOnceColumnExists(t *testing.T) {
	inner := &capturingInner{stubInner: &stubInner{}}
	w := newInventoryItemOps(inner, false)
	w.columnCache[entityid.JobTask] = map[string]bool{"workspace_id": true}

	if _, err := w.List(newCtxWithWorkspace("ws-1"), entityid.JobTask, nil); err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if inner.lastListParams == nil || inner.lastListParams.Filters == nil {
		t.Fatal("direct-required list reached inner operation without workspace filter")
	}
	filters := inner.lastListParams.Filters.Filters
	if len(filters) != 1 || filters[0].GetField() != "workspace_id" ||
		filters[0].GetStringFilter().GetValue() != "ws-1" {
		t.Fatalf("unexpected direct-required filters: %+v", filters)
	}
}

func TestWorkspaceCatalogErrorFailsClosed(t *testing.T) {
	w := newStubWorkspaceOps(&stubInner{}, false)
	delete(w.columnCache, "legacy_table")
	w.columnProbe = func(context.Context, string) (map[string]bool, error) {
		return nil, errors.New("catalog unavailable")
	}

	_, err := w.List(newCtxWithWorkspace("ws-1"), "legacy_table", nil)
	requireDatabaseErrorCode(t, err, "TENANT_SCOPE_UNAVAILABLE")
}

func TestTreasuryPolicyRejectsGenericQueryAndMissingSchema(t *testing.T) {
	w := newStubWorkspaceOps(&stubInner{}, false)
	w.columnCache[entityid.TreasuryCollection] = map[string]bool{}
	ctx := newCtxWithWorkspace("ws-1")

	_, err := w.Create(ctx, entityid.TreasuryCollection, map[string]any{"id": "collection-1"})
	requireDatabaseErrorCode(t, err, "TENANT_SCOPE_UNAVAILABLE")

	w.columnCache[entityid.TreasuryCollection] = map[string]bool{"workspace_id": true}
	_, err = w.Query(ctx, entityid.TreasuryCollection, interfaces.NewQueryBuilder())
	requireDatabaseErrorCode(t, err, "TENANT_SCOPE_UNAVAILABLE")
}

func TestRequireDirectWorkspaceGatesRawSQLBeforeMigration(t *testing.T) {
	w := newStubWorkspaceOps(&stubInner{}, false)
	w.columnCache[entityid.TreasuryDisbursement] = map[string]bool{}

	_, err := w.RequireDirectWorkspace(context.Background(), entityid.TreasuryDisbursement)
	requireDatabaseErrorCode(t, err, "WORKSPACE_REQUIRED")

	_, err = w.RequireDirectWorkspace(newCtxWithWorkspace("ws-1"), entityid.TreasuryDisbursement)
	requireDatabaseErrorCode(t, err, "TENANT_SCOPE_UNAVAILABLE")

	w.columnCache[entityid.TreasuryDisbursement] = map[string]bool{"workspace_id": true}
	workspaceID, err := w.RequireDirectWorkspace(newCtxWithWorkspace("ws-1"), entityid.TreasuryDisbursement)
	if err != nil {
		t.Fatalf("RequireDirectWorkspace() unexpected error: %v", err)
	}
	if workspaceID != "ws-1" {
		t.Fatalf("workspace ID = %q, want ws-1", workspaceID)
	}
}
