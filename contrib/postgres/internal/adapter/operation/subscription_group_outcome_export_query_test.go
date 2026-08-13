//go:build postgresql

package operation

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

func identityContext(workspaceID, userID string, principalType int32, principalID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:     workspaceID,
		WorkspaceUserID: "workspace-user-1",
		UserID:          userID,
		PrincipalType:   principalType,
		PrincipalID:     principalID,
	})
}

func requestIdentityFromContext(t *testing.T, ctx context.Context) *identity.RequestIdentity {
	t.Helper()
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil {
		t.Fatalf("context is missing identity")
	}
	return id
}

func stringPtr(value string) *string {
	return &value
}

func TestBuildSubscriptionGroupOutcomeExportSQL_SelectorDefaultsAndBinds(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: "sg-123",
	}
	built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")

	if built.staffScoped {
		t.Fatalf("staffScoped = true, want false for non-staff caller")
	}
	if got, want := len(built.args), 8; got != want {
		t.Fatalf("arg count = %d, want %d", got, want)
	}
	if got := built.args[0].(string); got != "ws-1" {
		t.Errorf("arg[0]=%q, want ws-1", got)
	}
	if got := built.args[1].(string); got != "sg-123" {
		t.Errorf("arg[1]=%q, want sg-123", got)
	}
	if got := built.args[2].(string); got != "workspace-user-1" {
		t.Errorf("arg[2]=%q, want workspace-user-1", got)
	}
	if got := built.args[3].(bool); got != false {
		t.Errorf("arg[4]=%v, want false", got)
	}
	if got := built.args[4].(string); got != "" {
		t.Errorf("arg[4]=%q, want empty category", got)
	}
	if got := built.args[5].(string); got != "options" {
		t.Errorf("arg[5]=%q, want options", got)
	}
	if got := built.args[6].(string); got != "" {
		t.Errorf("arg[6]=%q, want empty phase code", got)
	}
	if got := built.args[7].(bool); got != false {
		t.Errorf("arg[7]=%v, want staffScoped false bool arg", got)
	}

	if strings.Count(built.statement, "WITH ") != 1 {
		t.Errorf("expected one CTE block, statement has %d 'WITH' fragments", strings.Count(built.statement, "WITH "))
	}
	if strings.Contains(built.statement, "{{") || strings.Contains(built.statement, "}}") {
		t.Fatalf("statement still contains table placeholders")
	}
	if !strings.Contains(built.statement, "ORDER BY kind_order, sort_key_1, sort_key_2") {
		t.Fatalf("statement should preserve single stable outer sort")
	}
}

func TestExportSelectorAndRequestVariants(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	cases := []struct {
		name         string
		req          *exportpb.GetSubscriptionGroupOutcomeExportRequest
		wantKind     string
		wantPhase    string
		wantCategory string
	}{
		{
			name:         "no explicit selector defaults options",
			req:          &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-1"},
			wantKind:     "options",
			wantPhase:    "",
			wantCategory: "",
		},
		{
			name:         "explicit final selector",
			req:          &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-1", OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome{FinalOutcome: true}},
			wantKind:     "final",
			wantPhase:    "",
			wantCategory: "",
		},
		{
			name:         "explicit phase selector",
			req:          &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-1", JobCategoryId: stringPtr("cat-77"), OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{JobTemplatePhaseCode: "P-9"}},
			wantKind:     "phase",
			wantPhase:    "P-9",
			wantCategory: "cat-77",
		},
		{
			name:         "malformed request selector value ignored",
			req:          &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-1", OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome{FinalOutcome: false}},
			wantKind:     "options",
			wantPhase:    "",
			wantCategory: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, phaseCode := exportSelector(c.req)
			built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), c.req, ports.SubscriptionGroupOutcomeExportScope{}, kind, phaseCode)
			if got := built.args[5].(string); got != c.wantKind {
				t.Fatalf("selector kind = %q, want %q", got, c.wantKind)
			}
			if got := built.args[6].(string); got != c.wantPhase {
				t.Fatalf("phase code = %q, want %q", got, c.wantPhase)
			}
			if got := built.args[4].(string); got != c.wantCategory {
				t.Fatalf("job category = %q, want %q", got, c.wantCategory)
			}
			if strings.Contains(built.statement, "attribute_id") || strings.Contains(built.statement, "attribute_value") || strings.Contains(built.statement, "request_id") {
				t.Fatalf("request/attribute/value selector terms unexpectedly present in SQL")
			}
		})
	}
}

func TestBuildSubscriptionGroupOutcomeExportSQL_ResolverAndTenantPredicates(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: "sg-1",
		JobCategoryId:       stringPtr("cat-2"),
	}
	kind, phaseCode := exportSelector(req)
	built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, kind, phaseCode)
	statement := built.statement

	workspacePredicates := []string{
		"sg.workspace_id = $1",
		"m.workspace_id = $1",
		"s.workspace_id = $1",
		"c.workspace_id = $1",
		"j.workspace_id = $1",
		"jt.workspace_id = $1",
		"local_category.workspace_id = $1",
		"jtp.workspace_id = $1",
		"pos.workspace_id = $1",
		"jos.workspace_id = $1",
		"task.workspace_id = $1",
		"outcome.workspace_id = $1",
	}
	for _, predicate := range workspacePredicates {
		if !strings.Contains(statement, predicate) {
			t.Errorf("missing workspace predicate: %s", predicate)
		}
	}

	requiredFragments := []string{
		"outcome_capable_jobs AS (",
		"phase_facts AS (",
		"selected_jobs AS (",
		"g.historical OR (m.active = true AND s.active = true)",
		"m.historical = false AND j.active = true AND jt.id IS NOT NULL AND jt.active = true",
		"m.historical = true",
		"CASE WHEN jt.id IS NOT NULL THEN jt.job_category_id ELSE j.job_category_id END AS effective_category_id",
		"COALESCE(local_category.id, 'uncategorized') AS category_bucket_id",
		"final_summary.workspace_id = $1",
		"NOT EXISTS (\n           SELECT 1 FROM",
		"CROSS JOIN matrix_columns col",
		"ORDER BY j.id ASC",
		"ORDER BY jp.id ASC",
		"ORDER BY pos.date_created DESC NULLS LAST, pos.id DESC",
		"ORDER BY jos.date_created DESC NULLS LAST, jos.id DESC",
		"selected_phase.id",
		"LEFT JOIN LATERAL (",
		"SELECT pos.scaled_label, pos.scaled_score",
		"SELECT jos.scaled_label, jos.scaled_score",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("missing SQL fragment: %s", fragment)
		}
	}

	if !strings.Contains(statement, "WHERE $6 = 'phase'") {
		t.Fatalf("phase selector branch guard missing")
	}
	if !strings.Contains(statement, "WHERE $6 = 'final'") {
		t.Fatalf("final selector branch guard missing")
	}
	if strings.Contains(statement, "selected_phase.id") && !strings.Contains(statement, "job_phase_id = selected_phase.id") {
		t.Fatalf("phase evidence mismatch: phase summary should join through selected_phase.id")
	}
	if !strings.Contains(statement, "phase_summary") || !strings.Contains(statement, "final_summary") {
		t.Fatalf("both phase and final evidence blocks should be present in SQL")
	}
	if strings.Contains(statement, "true AS final_outcome_available") {
		t.Fatal("final availability must be derived per category, never hard-coded")
	}
	if strings.Contains(statement, "j.job_template_id != pl.job_template_id") {
		t.Fatal("global root-template exclusion is not allowed")
	}
}

func TestBuildSubscriptionGroupOutcomeExportSQL_PhaseSelectorRequiresExactTemplateMatch(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: "sg-1",
		JobCategoryId:       stringPtr("cat-2"),
		OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
			JobTemplatePhaseCode: "P1",
		},
	}
	kind, phaseCode := exportSelector(req)
	built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, kind, phaseCode)
	statement := built.statement

	requirements := []string{
		"FROM outcome_capable_jobs oj",
		"FROM outcome_capable_jobs final_job",
		"phase_facts AS (",
		"pf.job_template_id = oj.job_template_id",
	}
	for _, fragment := range requirements {
		if !strings.Contains(statement, fragment) {
			t.Fatalf("missing fragment in phase selector SQL: %s", fragment)
		}
	}
	if !regexp.MustCompile(`FROM phase_facts pf`).MatchString(statement) {
		t.Fatal("missing phase selector from phase_facts alias")
	}
	if !regexp.MustCompile(`pf\.code = \$\d+`).MatchString(statement) {
		t.Fatal("missing phase selector code placeholder")
	}
	if !regexp.MustCompile(`po\.code = \$\d+ AND po\.ambiguous = false`).MatchString(statement) {
		t.Fatal("missing final selector code placeholder")
	}
}

func TestBuildSubscriptionGroupOutcomeExportSQL_StaffScopeVariants(t *testing.T) {
	t.Parallel()

	req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: "sg-1",
	}

	t.Run("staff non-wide", func(t *testing.T) {
		ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "staff-9")
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")
		if !built.staffScoped {
			t.Fatal("staffScoped = false, want true")
		}
		if got := len(built.args); got != 10 {
			t.Fatalf("arg count = %d, want 10", got)
		}
		if got := built.args[8].(string); got != "staff-9" {
			t.Errorf("staff arg = %q, want staff-9", got)
		}
		if got := built.args[9].(string); got != "ws-1" {
			t.Errorf("workspace arg = %q, want ws-1", got)
		}
		if !strings.Contains(built.statement, " AND j.id IN (SELECT jp.job_id FROM") {
			t.Fatalf("staff clause should include StaffReachableJobClause")
		}
	})

	t.Run("malformed staff", func(t *testing.T) {
		ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "")
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")
		if !built.staffScoped {
			t.Fatal("malformed staff should set staffScoped=true")
		}
		if len(built.args) != 8 {
			t.Fatalf("arg count = %d, want 8", len(built.args))
		}
		if !strings.Contains(built.statement, "AND 1=0") {
			t.Fatal("malformed staff should emit fail-closed 1=0 clause")
		}
	})

	t.Run("workspace-wide staff", func(t *testing.T) {
		ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "staff-9")
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true}, "options", "")
		if built.staffScoped {
			t.Fatal("workspace-wide should disable staff scoping")
		}
		if len(built.args) != 8 {
			t.Fatalf("arg count = %d, want 8", len(built.args))
		}
		if strings.Contains(built.statement, "j.id IN (SELECT jp.job_id") {
			t.Fatal("workspace-wide call must omit StaffReachableJobClause")
		}
	})
}

func TestBuildSubscriptionGroupOutcomeExportSQL_PlaceholdersAndTables(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "sg-1"}
	built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")

	placeholders := []string{
		"{{subscription_group}}", "{{subscription_group_member}}", "{{subscription_group_workspace_user}}",
		"{{workspace_user}}", "{{subscription}}", "{{client}}", "{{job}}", "{{job_template}}",
		"{{job_category}}", "{{job_template_phase}}", "{{job_phase}}", "{{job_task}}",
		"{{task_outcome}}", "{{phase_outcome_summary}}", "{{job_outcome_summary}}",
		"{{price_schedule}}", "{{plan}}",
	}
	for _, placeholder := range placeholders {
		if strings.Contains(built.statement, placeholder) {
			t.Fatalf("placeholder %s was not rendered", placeholder)
		}
	}

	emit := []string{
		entityid.SubscriptionGroup, entityid.SubscriptionGroupMember, entityid.SubscriptionGroupWorkspaceUser,
		entityid.WorkspaceUser, entityid.Subscription, entityid.Client, entityid.Job,
		entityid.JobTemplate, entityid.JobCategory, entityid.JobTemplatePhase, entityid.JobPhase,
		entityid.JobTask, entityid.TaskOutcome, entityid.PhaseOutcomeSummary, entityid.JobOutcomeSummary,
		entityid.PriceSchedule, entityid.Plan,
	}
	for _, table := range emit {
		if !strings.Contains(built.statement, table) {
			t.Fatalf("rendered SQL does not contain table token %s", table)
		}
	}
}

func TestBuildSubscriptionGroupOutcomeExportSQL_AgencyForRenderTables(t *testing.T) {
	t.Parallel()

	rendered := renderOutcomeExportTables("{{subscription_group_document_template}} {{document_template}}")
	if strings.Contains(rendered, "{{") || strings.Contains(rendered, "}}") {
		t.Fatal("document render placeholders were not rendered")
	}
	for _, table := range []string{entityid.SubscriptionGroupDocumentTemplate, entityid.DocumentTemplate} {
		if !strings.Contains(rendered, table) {
			t.Fatalf("rendered document SQL misses %s", table)
		}
	}
}

func TestSubscriptionGroupOutcomeExportColumnComparatorCasefoldAndTieBreak(t *testing.T) {
	t.Parallel()

	if got := columnName(&exportpb.JobTemplateColumn{JobTemplateId: "J-1", DisplayName: " Alpha "}); got != "Alpha" {
		t.Fatalf("columnName = %q, want Alpha", got)
	}
	if got := columnName(&exportpb.JobTemplateColumn{JobTemplateId: "J-2", DisplayName: "   "}); got != "J-2" {
		t.Fatalf("blank display name should fallback to job_template_id, got %q", got)
	}

	got := []*exportpb.JobTemplateColumn{
		{JobTemplateId: "zz-2", DisplayName: "alpha"},
		{JobTemplateId: "aa-1", DisplayName: "Alpha"},
	}
	sort.SliceStable(got, func(i, j int) bool {
		an, bn := strings.ToLower(columnName(got[i])), strings.ToLower(columnName(got[j]))
		if an != bn {
			return an < bn
		}
		return got[i].GetJobTemplateId() < got[j].GetJobTemplateId()
	})
	if got[0].GetJobTemplateId() != "aa-1" {
		t.Fatalf("casefold comparator tie should defer to job_template_id, got %q", got[0].GetJobTemplateId())
	}
}

func TestSubscriptionGroupOutcomeExportPhaseComparatorSequenceThenCode(t *testing.T) {
	t.Parallel()

	a := &exportpb.JobTemplatePhaseOption{SequenceOrder: 2, Code: "a", Name: "Zulu"}
	b := &exportpb.JobTemplatePhaseOption{SequenceOrder: 2, Code: "b", Name: "Alpha"}
	if !phaseOptionLess(a, b) || phaseOptionLess(b, a) {
		t.Fatal("equal-sequence phases must sort by code before display name")
	}
	earlier := &exportpb.JobTemplatePhaseOption{SequenceOrder: 1, Code: "z", Name: "Last by name"}
	if !phaseOptionLess(earlier, a) {
		t.Fatal("sequence order must remain the primary phase comparator")
	}
}
