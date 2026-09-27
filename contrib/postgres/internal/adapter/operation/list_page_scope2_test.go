//go:build postgresql

package operation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	optionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
	thresholdpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	checkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

func TestScope2ListPagesRequireWorkspace(t *testing.T) {
	calls := map[string]func(context.Context) error{
		"criteria_option": func(ctx context.Context) error {
			_, err := (&PostgresCriteriaOptionRepository{}).GetCriteriaOptionListPageData(ctx, &optionpb.GetCriteriaOptionListPageDataRequest{})
			return err
		},
		"criteria_threshold": func(ctx context.Context) error {
			_, err := (&PostgresCriteriaThresholdRepository{}).GetCriteriaThresholdListPageData(ctx, &thresholdpb.GetCriteriaThresholdListPageDataRequest{})
			return err
		},
		"task_outcome_check": func(ctx context.Context) error {
			_, err := (&PostgresTaskOutcomeCheckRepository{}).GetTaskOutcomeCheckListPageData(ctx, &checkpb.GetTaskOutcomeCheckListPageDataRequest{})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				ctx  context.Context
				want error
			}{
				{context.Background(), identity.ErrIdentityNotInContext},
				{identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{}), identity.ErrWorkspaceNotSelected},
			} {
				if err := call(tc.ctx); !errors.Is(err, tc.want) {
					t.Fatalf("missing workspace: got %v want %v", err, tc.want)
				}
			}
		})
	}
}

func TestScope2SyntheticTenantPredicatesDB(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cases := []struct{ name, ctes, child, alias, predicate string }{
		{"criteria_option", `outcome_criteria(id,workspace_id) AS (VALUES ('pa','A'),('pb','B')), criteria_option(id,outcome_criteria_id,active) AS (VALUES ('a','pa',true),('b','pb',true),('n',NULL,true),('d','missing',true))`, "criteria_option co", "co", criteriaOptionListWorkspacePredicate},
		{"criteria_threshold", `outcome_criteria(id,workspace_id) AS (VALUES ('pa','A'),('pb','B')), criteria_threshold(id,outcome_criteria_id,active) AS (VALUES ('a','pa',true),('b','pb',true),('n',NULL,true),('d','missing',true))`, "criteria_threshold ct", "ct", criteriaThresholdListWorkspacePredicate},
		{"task_outcome_check", `job(id,workspace_id) AS (VALUES ('ja','A'),('jb','B')), job_phase(id,job_id) AS (VALUES ('jpa','ja'),('jpb','jb')), job_task(id,job_phase_id) AS (VALUES ('jta','jpa'),('jtb','jpb')), task_outcome(id,job_task_id) AS (VALUES ('ta','jta'),('tb','jtb')), task_outcome_check(id,task_outcome_id) AS (VALUES ('a','ta'),('b','tb'),('n',NULL),('d','missing'))`, "task_outcome_check toc", "toc", taskOutcomeCheckListWorkspacePredicate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query := `WITH ` + tc.ctes + ` SELECT ` + tc.alias + `.id FROM ` + tc.child + ` WHERE $1::text='' AND $2::int=50 AND $3::int=0 AND ` + tc.predicate + ` ORDER BY 1`
			for _, scope := range []struct {
				ws   string
				want []string
			}{{"A", []string{"a"}}, {"B", []string{"b"}}, {"missing", nil}} {
				rows, err := db.QueryContext(context.Background(), query, "", 50, 0, scope.ws)
				if err != nil {
					t.Fatal(err)
				}
				var got []string
				for rows.Next() {
					var id string
					if err := rows.Scan(&id); err != nil {
						t.Fatal(err)
					}
					got = append(got, id)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				rows.Close()
				if !reflect.DeepEqual(got, scope.want) {
					t.Fatalf("workspace %s: got %v want %v", scope.ws, got, scope.want)
				}
			}
		})
	}
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "scope2-no-such-workspace"})
	optionRepo := NewPostgresCriteriaOptionRepository(ops, "criteria_option").(*PostgresCriteriaOptionRepository)
	optionResp, err := optionRepo.GetCriteriaOptionListPageData(ctx, &optionpb.GetCriteriaOptionListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(optionResp.GetCriteriaOptionList()) != 0 {
		t.Fatal("foreign option rows")
	}
	thresholdRepo := NewPostgresCriteriaThresholdRepository(ops, "criteria_threshold").(*PostgresCriteriaThresholdRepository)
	thresholdResp, err := thresholdRepo.GetCriteriaThresholdListPageData(ctx, &thresholdpb.GetCriteriaThresholdListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(thresholdResp.GetCriteriaThresholdList()) != 0 {
		t.Fatal("foreign threshold rows")
	}
	checkRepo := NewPostgresTaskOutcomeCheckRepository(ops, "task_outcome_check").(*PostgresTaskOutcomeCheckRepository)
	checkResp, err := checkRepo.GetTaskOutcomeCheckListPageData(ctx, &checkpb.GetTaskOutcomeCheckListPageDataRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(checkResp.GetTaskOutcomeCheckList()) != 0 {
		t.Fatal("foreign check rows")
	}
}
