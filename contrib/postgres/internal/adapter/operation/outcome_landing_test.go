//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

func TestBuildSubscriptionGroupOutcomeLandingSQL_StaffScopeVariants(t *testing.T) {
	t.Parallel()

	active := true
	req := &exportpb.ListSubscriptionGroupOutcomeLandingRequest{PriceScheduleActive: &active}
	cases := []struct {
		name          string
		principalType int32
		workspaceWide bool
	}{
		{name: "non-staff", principalType: 1},
		{name: "staff non-wide", principalType: principalscope.PrincipalTypeStaff},
		{name: "staff workspace-wide", principalType: principalscope.PrincipalTypeStaff, workspaceWide: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := identityContext("ws-1", "user-1", tc.principalType, "principal-1")
			built := buildSubscriptionGroupOutcomeLandingSQL(ctx, requestIdentityFromContext(t, ctx), req, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: tc.workspaceWide})

			if built.staffScoped {
				t.Fatal("staffScoped = true, want false under the whole-assigned-section policy")
			}
			if got, want := len(built.args), 5; got != want {
				t.Fatalf("arg count = %d, want %d (ws, active, workspaceUserID, userID, wide)", got, want)
			}
			if strings.Contains(built.statement, "j.id IN (SELECT jp.job_id") || strings.Contains(built.statement, "1=0") {
				t.Fatal("policy off: landing must omit StaffReachableJobClause and its fail-closed branch")
			}
			if !strings.Contains(built.statement, "subscription_group_workspace_user") || !strings.Contains(built.statement, "sgwu.active = true") {
				t.Fatal("section-assignment (sgwu) gate must remain in the landing query")
			}
		})
	}
}

// The landing counts members and job templates PER GROUP before joining. Joining
// eligible_members to eligible_jobs on subscription_group_id and counting
// DISTINCT afterwards materialises members x jobs per group (measured on
// education2: ~225k rows, 17 s, past the app's 30 s statement timeout once the
// reach clause was added) and blanks the report-card landing. Results are
// identical without the fan-out (verified old-vs-new on real data).
func TestBuildSubscriptionGroupOutcomeLandingSQL_NoMembersByJobsFanOut(t *testing.T) {
	t.Parallel()

	ctx := identityContext("ws-1", "user-1", 1, "principal-1")
	built := buildSubscriptionGroupOutcomeLandingSQL(ctx, requestIdentityFromContext(t, ctx), &exportpb.ListSubscriptionGroupOutcomeLandingRequest{}, ports.SubscriptionGroupOutcomeExportScope{})

	for _, want := range []string{"member_counts AS (", "job_template_counts AS (", "COALESCE(mc.member_count, 0)", "COALESCE(jc.job_template_count, 0)"} {
		if !strings.Contains(built.statement, want) {
			t.Errorf("landing statement must aggregate per group before joining; missing %q", want)
		}
	}
	for _, banned := range []string{"LEFT JOIN eligible_members em", "LEFT JOIN eligible_jobs ej", "COUNT(DISTINCT em.member_id)", "COUNT(DISTINCT ej.job_template_id)"} {
		if strings.Contains(built.statement, banned) {
			t.Errorf("landing statement reintroduces the members x jobs fan-out via %q", banned)
		}
	}
}
