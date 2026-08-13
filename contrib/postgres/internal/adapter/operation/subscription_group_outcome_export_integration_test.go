//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
	bindingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
	_ "github.com/lib/pq"
)

// TestIntegration_SubscriptionGroupOutcomeExport_ReadOnlyLiveShape is a
// privacy-safe read oracle over the configured integration lane. It discovers
// one active group/category pair by aggregate shape only, executes options and
// one selected matrix through the production adapter, and never logs learner,
// group, category, or template identifiers/names.
func TestIntegration_SubscriptionGroupOutcomeExport_ReadOnlyLiveShape(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	var workspaceID, groupID, categoryID string
	var rawAcademic, outcomeCapable, s1Capable int
	err = db.QueryRowContext(ctx, `
		SELECT sg.workspace_id, sg.id, jc.id,
		       count(DISTINCT jt.id) FILTER (WHERE TRUE) AS raw_academic,
		       count(DISTINCT jt.id) FILTER (
		         WHERE EXISTS (
		           SELECT 1 FROM job_template_phase jtp
		            WHERE jtp.workspace_id = sg.workspace_id
		              AND jtp.job_template_id = jt.id
		              AND jtp.active = true
		         )
		         OR EXISTS (
		           SELECT 1 FROM job_outcome_summary jos
		            WHERE jos.workspace_id = sg.workspace_id
		              AND jos.job_id = j.id
		              AND jos.active = true
		         )
		       ) AS outcome_capable,
		       count(DISTINCT jt.id) FILTER (
		         WHERE EXISTS (
		           SELECT 1 FROM job_template_phase jtp
		            WHERE jtp.workspace_id = sg.workspace_id
		              AND jtp.job_template_id = jt.id
		              AND jtp.code = 's1'
		              AND jtp.active = true
		         )
		       ) AS s1_capable
		  FROM subscription_group sg
		  JOIN subscription_group_member sgm
		    ON sgm.subscription_group_id = sg.id
		   AND sgm.workspace_id = sg.workspace_id
		   AND sgm.active = true
		  JOIN job j
		    ON j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
		   AND j.origin_id = sgm.subscription_id
		   AND j.client_id = sgm.client_id
		   AND j.workspace_id = sg.workspace_id
		   AND j.active = true
		  JOIN job_template jt
		    ON jt.id = j.job_template_id
		   AND jt.workspace_id = sg.workspace_id
		   AND jt.active = true
		  JOIN job_category jc
		    ON jc.id = jt.job_category_id
		   AND jc.workspace_id = sg.workspace_id
		   AND jc.active = true
		 WHERE sg.active = true
		   AND lower(btrim(jc.code)) = 'academic'
		GROUP BY sg.workspace_id, sg.id, jc.id
		HAVING count(DISTINCT jt.id) = 12
		 ORDER BY sg.id
		 LIMIT 1`).Scan(&workspaceID, &groupID, &categoryID, &rawAcademic, &outcomeCapable, &s1Capable)
	if err == sql.ErrNoRows {
		t.Skip("no privacy-safe active academic integration shape")
	}
	if err != nil {
		t.Fatalf("discover aggregate integration shape: %v", err)
	}
	if rawAcademic != 12 || outcomeCapable != 11 || s1Capable != 11 {
		t.Fatalf("unexpected integration shape counts raw=%d outcome=%d s1=%d", rawAcademic, outcomeCapable, s1Capable)
	}

	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
		WorkspaceID: workspaceID,
		UserID:      "subscription-group-outcome-export-read-oracle",
	})
	query := NewPostgresSubscriptionGroupOutcomeExportQuery(db)
	wide := ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true}
	options, err := query.GetSubscriptionGroupOutcomeExportScoped(ctx, &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: groupID,
	}, wide)
	if err != nil {
		t.Fatalf("options query failed (identifiers withheld): %v", err)
	}
	if options.GetContext() == nil || len(options.GetJobCategories()) == 0 {
		t.Fatal("options query returned no scoped context/category")
	}

	selected := (*exportpb.JobCategoryOption)(nil)
	for _, category := range options.GetJobCategories() {
		if category.GetJobCategoryId() == categoryID {
			selected = category
			break
		}
	}
	if selected == nil {
		t.Fatal("discovered category was not returned by the scoped options query")
	}

	matrixReq := &exportpb.GetSubscriptionGroupOutcomeExportRequest{
		SubscriptionGroupId: groupID,
		JobCategoryId:       &categoryID,
	}
	for _, phase := range selected.GetJobTemplatePhases() {
		if !phase.GetAmbiguous() {
			matrixReq.OutcomeSelector = &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
				JobTemplatePhaseCode: phase.GetCode(),
			}
			break
		}
	}
	if matrixReq.GetOutcomeSelector() == nil {
		if !selected.GetFinalOutcomeAvailable() {
			t.Skip("discovered category has no unambiguous selectable outcome")
		}
		matrixReq.OutcomeSelector = &exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome{FinalOutcome: true}
	}
	matrix, err := query.GetSubscriptionGroupOutcomeExportScoped(ctx, matrixReq, wide)
	if err != nil {
		t.Fatalf("matrix query failed (identifiers withheld): %v", err)
	}
	if len(matrix.GetJobTemplateColumns()) != 11 || len(matrix.GetClientRows()) == 0 {
		t.Fatal("selected exact-profile matrix did not return eleven columns and at least one client row")
	}
	for _, row := range matrix.GetClientRows() {
		if len(row.GetCells()) != len(matrix.GetJobTemplateColumns()) {
			t.Fatal("selected matrix is not a complete client-by-column rectangle")
		}
		for _, cell := range row.GetCells() {
			if cell.GetEnrollmentEvidence() == nil {
				t.Fatal("selected matrix contains a cell without enrollment evidence")
			}
		}
	}

	resolved, err := query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
		SubscriptionGroupId:     groupID,
		JobCategoryId:           categoryID,
		RenderProfile:           bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
		ExpectedPlanId:          matrix.GetContext().PlanId,
		ExpectedPriceScheduleId: matrix.GetContext().PriceScheduleId,
	}, wide)
	if err != nil {
		t.Fatalf("render resolver failed (identifiers withheld): %v", err)
	}
	if !resolved.GetFound() || resolved.GetDocument() == nil {
		t.Fatal("render resolver did not find the published exact-profile document")
	}
}
