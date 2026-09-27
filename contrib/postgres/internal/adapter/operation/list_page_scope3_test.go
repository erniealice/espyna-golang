//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	phasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
	templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type scope3Capture struct {
	query string
	args  []driver.NamedValue
}

func (c *scope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &scope3Conn{capture: c}, nil
}
func (*scope3Capture) Driver() driver.Driver { return scope3Driver{} }

type scope3Driver struct{}

func (scope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("scope3 opens by connector only")
}

type scope3Conn struct{ capture *scope3Capture }

func (*scope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*scope3Conn) Close() error              { return nil }
func (*scope3Conn) Begin() (driver.Tx, error) { return nil, errors.New("begin unsupported") }
func (c *scope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return scope3EmptyRows{}, nil
}

type scope3EmptyRows struct{}

func (scope3EmptyRows) Columns() []string         { return nil }
func (scope3EmptyRows) Close() error              { return nil }
func (scope3EmptyRows) Next([]driver.Value) error { return io.EOF }

func TestScope3ListPagesFailClosedAndBindParentSet(t *testing.T) {
	for _, tc := range []struct {
		name      string
		call      func(context.Context, *sql.DB) error
		fragments []string
		staff     bool
	}{
		{
			name: "phase_outcome_summary",
			call: func(ctx context.Context, db *sql.DB) error {
				_, err := (&PostgresPhaseOutcomeSummaryRepository{db: db}).GetPhaseOutcomeSummaryListPageData(ctx, &phasepb.GetPhaseOutcomeSummaryListPageDataRequest{})
				return err
			},
			fragments: []string{"scoped_jobs AS MATERIALIZED", "j.workspace_id = $4", "pos.job_id IN (SELECT id FROM scoped_jobs)", "enriched AS MATERIALIZED", "SELECT COUNT(*) FROM enriched"},
			staff:     true,
		},
		{
			name: "template_task_criteria",
			call: func(ctx context.Context, db *sql.DB) error {
				_, err := (&PostgresTemplateTaskCriteriaRepository{db: db}).GetTemplateTaskCriteriaListPageData(ctx, &templatepb.GetTemplateTaskCriteriaListPageDataRequest{})
				return err
			},
			fragments: []string{"scoped_tasks AS MATERIALIZED", "SELECT jtt.id", "jtp.id = jtt.job_template_phase_id", "jt.id = jtp.job_template_id", "jt.workspace_id = $4", "ttc.job_template_task_id IN (SELECT id FROM scoped_tasks)", "enriched AS MATERIALIZED", "SELECT COUNT(*) FROM enriched"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := &scope3Capture{}
			db := sql.OpenDB(capture)
			defer db.Close()
			for _, missing := range []struct {
				ctx  context.Context
				want error
			}{
				{context.Background(), identity.ErrIdentityNotInContext},
				{identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{}), identity.ErrWorkspaceNotSelected},
			} {
				if err := tc.call(missing.ctx, db); !errors.Is(err, missing.want) {
					t.Fatalf("missing workspace: got %v want %v", err, missing.want)
				}
				if capture.query != "" {
					t.Fatal("missing workspace reached database")
				}
			}
			for _, principal := range []struct {
				name string
				id   *identity.RequestIdentity
				argc int
			}{
				{"admin", &identity.RequestIdentity{WorkspaceID: "workspace-A"}, 4},
				{"staff", &identity.RequestIdentity{WorkspaceID: "workspace-A", PrincipalType: 7, PrincipalID: "staff-A"}, 5},
			} {
				if principal.name == "staff" && !tc.staff {
					continue
				}
				t.Run(principal.name, func(t *testing.T) {
					capture.query = ""
					capture.args = nil
					ctx := identity.WithRequestIdentity(context.Background(), principal.id)
					if err := tc.call(ctx, db); err != nil {
						t.Fatal(err)
					}
					if len(capture.args) != principal.argc || capture.args[principal.argc-1].Value != "workspace-A" {
						t.Fatalf("workspace bind: got %v", capture.args)
					}
					for _, fragment := range tc.fragments {
						if principal.name == "staff" && fragment == "j.workspace_id = $4" {
							fragment = "j.workspace_id = $5"
						}
						if !strings.Contains(capture.query, fragment) {
							t.Fatalf("query lacks %q", fragment)
						}
					}
					if tc.staff && principal.name == "staff" {
						if capture.args[3].Value != "staff-A" || !strings.Contains(capture.query, "pos.issued_by = $4") {
							t.Fatal("staff filter or bind missing")
						}
					}
				})
			}
		})
	}
}
