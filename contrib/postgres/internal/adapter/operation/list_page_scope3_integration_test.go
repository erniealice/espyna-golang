//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

func scope3ReadOnlyDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	var actual, readOnly string
	var localHost bool
	if err := db.QueryRow(`SELECT current_database(), current_setting('default_transaction_read_only'),
		inet_server_addr() IS NULL OR inet_server_addr() <<= inet '127.0.0.0/8' OR inet_server_addr() = inet '::1'`).Scan(&actual, &readOnly, &localHost); err != nil {
		t.Fatal(err)
	}
	if !localHost {
		db.Close()
		t.Fatal("scope3 integration tests require a local database")
	}
	if actual != name {
		db.Close()
		t.Skipf("requires %s, got %s", name, actual)
	}
	if readOnly != "on" {
		db.Close()
		t.Fatal("TEST_DATABASE_URL must set default_transaction_read_only=on")
	}
	return db
}

func scope3LegacyIDsAndTotal(t *testing.T, db *sql.DB, query string, args ...any) ([]string, int64) {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	var total int64
	for rows.Next() {
		var id string
		if err := rows.Scan(&id, &total); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids, total
}

func TestScope3CloneParityAndNegative(t *testing.T) {
	db := scope3ReadOnlyDB(t, "education2clone20260925b")
	defer db.Close()
	var workspace string
	if err := db.QueryRow(`SELECT workspace_id FROM job GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	phaseRepo := NewPostgresPhaseOutcomeSummaryRepository(ops, "phase_outcome_summary").(*PostgresPhaseOutcomeSummaryRepository)
	templateRepo := NewPostgresTemplateTaskCriteriaRepository(ops, "template_task_criteria").(*PostgresTemplateTaskCriteriaRepository)
	page := func(n int32) *commonpb.PaginationRequest {
		return &commonpb.PaginationRequest{Limit: 50, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}
	}
	admin := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
	staff := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace, PrincipalType: 7, PrincipalID: "scope3-no-issued-summary"})
	foreign := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "scope3-foreign-workspace"})
	for _, p := range []int32{1, 21} {
		t.Run(fmt.Sprintf("phase/admin/page%d", p), func(t *testing.T) {
			want, total := scope3LegacyIDsAndTotal(t, db, `SELECT e.id, COUNT(*) OVER () FROM phase_outcome_summary e WHERE e.active=true ORDER BY date_created DESC, id ASC LIMIT $1 OFFSET $2`, 50, (p-1)*50)
			got, err := phaseRepo.GetPhaseOutcomeSummaryListPageData(admin, &phasepb.GetPhaseOutcomeSummaryListPageDataRequest{Pagination: page(p)})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, 0, len(got.GetPhaseOutcomeSummaryList()))
			for _, row := range got.GetPhaseOutcomeSummaryList() {
				ids = append(ids, row.GetId())
			}
			if !slices.Equal(ids, want) || int64(got.GetPagination().GetTotalItems()) != total {
				t.Fatalf("phase ids/order/total mismatch: rows=%d/%d total=%d/%d", len(ids), len(want), got.GetPagination().GetTotalItems(), total)
			}
			t.Logf("rows=%d total=%d", len(ids), total)
		})
		t.Run(fmt.Sprintf("phase/staff/page%d", p), func(t *testing.T) {
			want, _ := scope3LegacyIDsAndTotal(t, db, `SELECT e.id, COUNT(*) OVER () FROM phase_outcome_summary e WHERE e.active=true AND e.issued_by=$3 ORDER BY date_created DESC, id ASC LIMIT $1 OFFSET $2`, 50, (p-1)*50, "scope3-no-issued-summary")
			got, err := phaseRepo.GetPhaseOutcomeSummaryListPageData(staff, &phasepb.GetPhaseOutcomeSummaryListPageDataRequest{Pagination: page(p)})
			if err != nil {
				t.Fatal(err)
			}
			if len(want) != 0 || len(got.GetPhaseOutcomeSummaryList()) != 0 || got.GetPagination().GetTotalItems() != 0 {
				t.Fatalf("staff scope mismatch: legacy=%d scoped=%d total=%d", len(want), len(got.GetPhaseOutcomeSummaryList()), got.GetPagination().GetTotalItems())
			}
		})
		t.Run(fmt.Sprintf("template/admin/page%d", p), func(t *testing.T) {
			want, total := scope3LegacyIDsAndTotal(t, db, `SELECT e.id, COUNT(*) OVER () FROM template_task_criteria e WHERE e.active=true ORDER BY sequence_order ASC, id ASC LIMIT $1 OFFSET $2`, 50, (p-1)*50)
			got, err := templateRepo.GetTemplateTaskCriteriaListPageData(admin, &templatepb.GetTemplateTaskCriteriaListPageDataRequest{Pagination: page(p)})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, 0, len(got.GetTemplateTaskCriteriaList()))
			for _, row := range got.GetTemplateTaskCriteriaList() {
				ids = append(ids, row.GetId())
			}
			if !slices.Equal(ids, want) || int64(got.GetPagination().GetTotalItems()) != total {
				t.Fatalf("template ids/order/total mismatch: rows=%d/%d total=%d/%d", len(ids), len(want), got.GetPagination().GetTotalItems(), total)
			}
			t.Logf("rows=%d total=%d", len(ids), total)
		})
	}
	phaseResult, err := phaseRepo.GetPhaseOutcomeSummaryListPageData(foreign, &phasepb.GetPhaseOutcomeSummaryListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	templateResult, err := templateRepo.GetTemplateTaskCriteriaListPageData(foreign, &templatepb.GetTemplateTaskCriteriaListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(phaseResult.GetPhaseOutcomeSummaryList()) != 0 || phaseResult.GetPagination().GetTotalItems() != 0 ||
		len(templateResult.GetTemplateTaskCriteriaList()) != 0 || templateResult.GetPagination().GetTotalItems() != 0 {
		t.Fatal("foreign workspace returned clone rows or count")
	}
}

func TestScope3ProfessionalForeignWorkspaceBind(t *testing.T) {
	db := scope3ReadOnlyDB(t, "professional2")
	defer db.Close()
	rows, err := db.Query(`SELECT id FROM workspace ORDER BY id LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}
	var workspaces []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		workspaces = append(workspaces, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if len(workspaces) != 2 || workspaces[0] == workspaces[1] {
		t.Fatal("professional2 needs two distinct workspaces")
	}
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	phaseRepo := NewPostgresPhaseOutcomeSummaryRepository(ops, "phase_outcome_summary").(*PostgresPhaseOutcomeSummaryRepository)
	templateRepo := NewPostgresTemplateTaskCriteriaRepository(ops, "template_task_criteria").(*PostgresTemplateTaskCriteriaRepository)
	for _, workspace := range workspaces {
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
		phase, err := phaseRepo.GetPhaseOutcomeSummaryListPageData(ctx, &phasepb.GetPhaseOutcomeSummaryListPageDataRequest{})
		if err != nil {
			t.Fatal(err)
		}
		template, err := templateRepo.GetTemplateTaskCriteriaListPageData(ctx, &templatepb.GetTemplateTaskCriteriaListPageDataRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(phase.GetPhaseOutcomeSummaryList()) != 0 || phase.GetPagination().GetTotalItems() != 0 ||
			len(template.GetTemplateTaskCriteriaList()) != 0 || template.GetPagination().GetTotalItems() != 0 {
			t.Fatal("unexpected professional2 child rows or count")
		}
	}
}
