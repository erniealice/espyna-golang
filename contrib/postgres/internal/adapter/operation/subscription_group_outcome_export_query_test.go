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
	bindingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
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

	// Policy (owner decision 2026-09-21, narrowStaffReportsToReachableJobs=false):
	// a non-wide STAFF caller is NOT narrowed to its reachable-job graph; the
	// section-assignment (sgwu) EXISTS gate in group_context is the row gate, so
	// an assigned STAFF principal reads the whole assigned section, read-only.
	t.Run("staff non-wide sees the whole assigned section", func(t *testing.T) {
		if narrowStaffReportsToReachableJobs {
			t.Skip("policy switched back to reachable-job narrowing")
		}
		ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "staff-9")
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")
		if built.staffScoped {
			t.Fatal("staffScoped = true, want false under the whole-assigned-section policy")
		}
		if got := len(built.args); got != 8 {
			t.Fatalf("arg count = %d, want 8 (no staff-clause args)", got)
		}
		if strings.Contains(built.statement, "j.id IN (SELECT jp.job_id") {
			t.Fatal("policy off: statement must omit StaffReachableJobClause")
		}
		// The section assignment stays the fail-closed row gate.
		if !strings.Contains(built.statement, "subscription_group_workspace_user") || !strings.Contains(built.statement, "$4::boolean") {
			t.Fatal("section-assignment (sgwu) gate must remain in group_context")
		}
		if got := built.args[2].(string); got == "" {
			t.Error("acting workspace_user id must still be bound for the sgwu gate")
		}
	})

	t.Run("malformed staff without a workspace_user still fails closed at the sgwu gate", func(t *testing.T) {
		ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "")
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{}, "options", "")
		if built.staffScoped || len(built.args) != 8 {
			t.Fatalf("staffScoped=%v args=%d, want false/8", built.staffScoped, len(built.args))
		}
		if !strings.Contains(built.statement, "wu.id = $3") {
			t.Fatal("the sgwu gate must key on the acting workspace_user ($3) so an empty binding matches no assignment")
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

func TestBuildSubscriptionGroupOutcomeDocumentResolverSQL_StaffWholeSection(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "staff-9")
	req := &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
		SubscriptionGroupId: "sg-1",
		JobCategoryId:       "cat-1",
		RenderProfile:       bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
	}
	built := buildSubscriptionGroupOutcomeDocumentResolverSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{})

	if built.staffScoped {
		t.Fatal("staffScoped = true, want false under the whole-assigned-section policy")
	}
	if strings.Contains(built.statement, "j.id IN (SELECT jp.job_id") || strings.Contains(built.statement, "1=0") {
		t.Fatal("policy off: document resolver must omit StaffReachableJobClause and its fail-closed branch")
	}
	if !strings.Contains(built.statement, "subscription_group_workspace_user") || !strings.Contains(built.statement, "sgwu.active = true") {
		t.Fatal("section-assignment (sgwu) gate must remain in the document resolver")
	}
	if !strings.Contains(built.statement, "wu.id = $3") {
		t.Fatal("document resolver must bind the acting workspace_user id at $3")
	}
	if got := built.args[2].(string); got != "workspace-user-1" {
		t.Fatalf("args[2] = %q, want acting workspace_user id", got)
	}
	if !strings.Contains(built.statement, "b.job_category_id = $5") {
		t.Fatal("matrix profile must keep exact category binding resolution")
	}
}

func TestBuildSubscriptionGroupOutcomeDocumentResolverSQL_ClientPhaseUsesNullCategory(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", principalscope.PrincipalTypeStaff, "staff-9")
	req := &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
		SubscriptionGroupId: "sg-1",
		RenderProfile:       bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_CLIENT_PHASE_OUTCOME_REPORT_V1,
	}
	built := buildSubscriptionGroupOutcomeDocumentResolverSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{})
	if !strings.Contains(built.statement, "b.job_category_id IS NULL") {
		t.Fatal("client phase profile must resolve a whole-report NULL category binding")
	}
	if !strings.Contains(built.statement, "EXISTS (SELECT 1 FROM outcome_capable_jobs)") {
		t.Fatal("whole-report resolver must require at least one outcome-capable job across categories")
	}
	if strings.Contains(built.statement, "b.job_category_id = $5") {
		t.Fatal("client phase profile must not resolve a category-bound matrix binding")
	}
	if !strings.Contains(built.statement, "g.plan_id IS NOT DISTINCT FROM") || !strings.Contains(built.statement, "g.price_schedule_id IS NOT DISTINCT FROM") {
		t.Fatal("whole-report binding must retain exact group plan/schedule scope")
	}
}

func TestNarrowStaffReportsPolicyDocumentedOff(t *testing.T) {
	if narrowStaffReportsToReachableJobs {
		t.Fatal("narrowStaffReportsToReachableJobs must stay false; flipping it requires a conscious review in docs/plan/20260921-section-manager-report-card-access/plan.md")
	}
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

func TestBuildSubscriptionGroupClientReportCardSQL_IsClientAnchoredTenantScopedAndAllowlisted(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "non-staff-1")
	id := requestIdentityFromContext(t, ctx)
	req := &exportpb.GetSubscriptionGroupClientReportCardRequest{
		SubscriptionGroupId:  "sg-1",
		ClientId:             "client-9",
		ClientAttributeCodes: []string{"student_code", "display_name'); SELECT pg_sleep(9); --"},
	}
	built := buildSubscriptionGroupClientReportCardSQL(id, req, ports.SubscriptionGroupOutcomeExportScope{}, `[
		"student_code", "display_name'); SELECT pg_sleep(9); --"
	]`)
	statement := built.statement

	if got, want := len(built.args), 7; got != want {
		t.Fatalf("arg count = %d, want %d", got, want)
	}
	for i, want := range []string{"ws-1", "sg-1", "workspace-user-1", "client-9"} {
		if got := built.args[i].(string); got != want {
			t.Errorf("arg[%d] = %q, want %q", i, got, want)
		}
	}
	if got := built.args[4].(string); !strings.Contains(got, "display_name'); SELECT pg_sleep(9); --") {
		t.Fatalf("attribute-code JSON binding lost requested value: %q", got)
	}
	if got := built.args[6].(string); got != "sg-1" {
		t.Fatalf("gate group bind = %q, want exact requested group sg-1", got)
	}
	if strings.Contains(statement, "display_name'); SELECT pg_sleep(9); --") || !strings.Contains(statement, "jsonb_array_elements_text($5::jsonb)") {
		t.Fatal("attribute code values must be bound as data, never interpolated into SQL")
	}

	anchorAt := strings.Index(statement, "member_anchor AS MATERIALIZED")
	jobsAt := strings.Index(statement, "job_rows AS MATERIALIZED")
	if anchorAt < 0 || jobsAt < 0 || anchorAt >= jobsAt {
		t.Fatalf("exact client membership must be materialized before job selection (anchor=%d jobs=%d)", anchorAt, jobsAt)
	}
	for _, fragment := range []string{
		"m.client_id = $4",
		"s.id = m.subscription_id AND s.workspace_id = $1 AND s.client_id = m.client_id",
		"g.historical OR c.active = true",
		"j.origin_id = a.subscription_id",
		"j.client_id = a.client_id",
		"j.workspace_id = $1",
		`LEFT JOIN "user" u ON u.id = s.user_id AND u.active = true`,
		"sg.workspace_id = $1",
		"ps.workspace_id = sg.workspace_id",
		"pl.workspace_id = sg.workspace_id",
		"sgwu.workspace_id = wu.workspace_id",
		"sgwu.subscription_group_id = sg.id",
		"wu.id = $3",
		"m.workspace_id = $1",
		"ca.client_id = a.client_id",
		"attr.code IN (SELECT jsonb_array_elements_text($5::jsonb))",
		"local_jt.id = j.job_template_id AND local_jt.workspace_id = $1",
		"SELECT DISTINCT jt.id, jt.workspace_id, jt.name, jt.template_code, jt.job_category_id, jt.active",
		"'template_code', jt.template_code",
		"jc.workspace_id = $1",
		"jp.job_id = j.id AND jp.workspace_id = $1",
		"DISTINCT ON (a.client_id, j.job_template_id)",
		"ORDER BY a.client_id, j.job_template_id, j.id ASC",
		"jtp.id = jp.template_phase_id AND jtp.workspace_id = $1",
		"tp.historical OR jtt.active = true",
		"jtt.job_template_phase_id = tp.id AND jtt.workspace_id = $1",
		"jp.historical OR jt.active = true",
		"jt.job_phase_id = jp.id AND jt.workspace_id = $1",
		"tt.historical OR ttc.active = true",
		"ttc.job_template_task_id = tt.id AND ttc.workspace_id = $1",
		"ttc.historical OR oc.active = true",
		"oc.id = ttc.outcome_criteria_id AND oc.workspace_id = $1",
		"o.workspace_id = $1 AND (jt.historical OR o.active = true)",
		"td.template_task_criteria_id = ttc.id",
		"td.workspace_id = $1 AND (ttc.historical OR td.active = true)",
		"jp.template_phase_id IS NOT NULL",
		"sgm_g.subscription_group_id = $7",
		"sgm_g.workspace_id = $1",
		"BOOL_OR(",
		"BOOL_AND(jp.approval_status = 'PHASE_APPROVAL_STATUS_PUBLISHED')",
		"gate_outcome.workspace_id = $1 AND gate_outcome.active = true",
		"'summary_score', jos.summary_score",
		"'summary_score', pos.summary_score",
		"'approval_status', jp.approval_status",
		"pos.workspace_id = $1 AND pos.active = true",
		"jos.job_id = j.id AND jos.workspace_id = $1 AND jos.active = true",
		"sgpps.workspace_id = $1",
		"s.workspace_id = $1",
		"jol.workspace_id = $1",
		"'client_id', jos.client_id",
		"'client_id', jol.client_id",
		"pp.product_id = j.output_product_id",
		"sgpps.subscription_group_id = $2",
		"sgpps.job_template_phase_id IS NULL OR sgpps.job_template_phase_id = jp.template_phase_id",
		"WHERE NOT EXISTS (\n     SELECT 1 FROM job_tasks direct_task",
		"render_gate_job_id",
	} {
		if !strings.Contains(statement, fragment) {
			t.Errorf("missing scoped client projection predicate/row: %q", fragment)
		}
	}

	if strings.Contains(statement, "matrix_members") {
		t.Fatal("client projection must not scan or construct a group roster")
	}
	if !strings.Contains(statement, "m.subscription_group_id = g.id AND m.workspace_id = $1\n     AND m.client_id = $4") {
		t.Fatal("membership scan must target exactly the requested client")
	}
	if !strings.Contains(statement, "SELECT DISTINCT 2, 'client_subscription'") || !strings.Contains(statement, "SELECT DISTINCT 20, 'render_gate_job_id'") {
		t.Fatal("all exact client subscriptions and render-gate jobs must be returned once each")
	}
	if !strings.Contains(statement, "'render_gate_group_id'") || !strings.Contains(statement, "'render_gate_sheet'") {
		t.Fatal("typed render gate must include an applied-group echo and sheet rollups")
	}
	if strings.Contains(statement, "{{render_gate_group_narrow}}") {
		t.Fatal("render-gate narrow marker was not replaced")
	}
	if strings.Contains(statement, "j.id IN (SELECT jp.job_id") || strings.Contains(statement, "StaffReachableJobClause") {
		t.Fatal("client projection must not intersect a broad staff reachable-job set")
	}
	if strings.Contains(statement, "to_jsonb(j)") || strings.Contains(statement, "to_jsonb(c)") || strings.Contains(statement, "to_jsonb(s)") {
		t.Fatal("projection must use explicit safe fields, never serialize whole sensitive rows")
	}
	if strings.Contains(statement, "{{") || strings.Contains(statement, "}}") {
		t.Fatal("client report card SQL contains an unrendered table placeholder")
	}
}

func TestAppendClientReportCardPayloadDecodesTypedNarrowRows(t *testing.T) {
	t.Parallel()

	projection := &exportpb.ClientReportCardProjection{}
	rows := []struct{ kind, payload string }{
		{"client", `{"client_id":"c-1","name":"Ada","first_name":"Ada","last_name":"Lovelace"}`},
		{"job_template", `{"id":"template-1","workspace_id":"ws-1","name":"Attendance","template_code":"attendance-template-code","job_category_id":"category-1","active":true}`},
		{"client_subscription", `"sub-1"`},
		{"client_subscription", `"sub-2"`},
		{"render_gate_job_id", `"job-1"`},
		{"render_gate_job_id", `"job-2"`},
		{"render_gate_group_id", `"group-1"`},
		{"render_gate_sheet", `{"job_template_phase_id":"phase-template-1","applied_subscription_group_id":"group-1","target_count":2,"any_workflow_entered":true,"all_published":false,"has_data":true}`},
		{"attribute", `{"code":"student_code","value":"S-1"}`},
	}
	for _, row := range rows {
		if err := appendClientReportCardPayload(projection, row.kind, []byte(row.payload)); err != nil {
			t.Fatalf("append %s: %v", row.kind, err)
		}
	}
	if projection.GetClient().GetClientId() != "c-1" || len(projection.GetClientSubscriptionIds()) != 2 || projection.GetClientSubscriptionIds()[0] != "sub-1" || projection.GetClientSubscriptionIds()[1] != "sub-2" {
		t.Fatalf("narrow DTO decode failed: client=%v subscriptions=%v", projection.GetClient(), projection.GetClientSubscriptionIds())
	}
	if len(projection.GetJobTemplates()) != 1 || projection.GetJobTemplates()[0].GetTemplateCode() != "attendance-template-code" {
		t.Fatalf("job template code decode = %v, want attendance-template-code", projection.GetJobTemplates())
	}
	if len(projection.GetRenderGateJobIds()) != 2 || projection.GetRenderGateJobIds()[0] != "job-1" || projection.GetRenderGateJobIds()[1] != "job-2" {
		t.Fatalf("render gate ids = %v, want [job-1 job-2]", projection.GetRenderGateJobIds())
	}
	if projection.GetRenderGateAppliedSubscriptionGroupId() != "group-1" || len(projection.GetRenderGateSheets()) != 1 || projection.GetRenderGateSheets()[0].GetTargetCount() != 2 {
		t.Fatalf("render gate sheet decode = group %q, sheets %v", projection.GetRenderGateAppliedSubscriptionGroupId(), projection.GetRenderGateSheets())
	}
	if len(projection.GetAttributes()) != 1 || projection.GetAttributes()[0].GetCode() != "student_code" {
		t.Fatalf("attribute decode = %v", projection.GetAttributes())
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
