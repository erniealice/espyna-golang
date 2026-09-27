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
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type listPageCapture struct {
	query string
	args  []driver.NamedValue
}

func (c *listPageCapture) Connect(context.Context) (driver.Conn, error) {
	return &listPageCaptureConn{c}, nil
}
func (c *listPageCapture) Driver() driver.Driver { return listPageCaptureDriver{} }

type listPageCaptureDriver struct{}

func (listPageCaptureDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("open by DSN is unsupported")
}

type listPageCaptureConn struct{ capture *listPageCapture }

func (*listPageCaptureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is unsupported")
}
func (*listPageCaptureConn) Close() error { return nil }
func (*listPageCaptureConn) Begin() (driver.Tx, error) {
	return nil, errors.New("begin is unsupported")
}
func (c *listPageCaptureConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return listPageEmptyRows{}, nil
}

type listPageEmptyRows struct{}

func (listPageEmptyRows) Columns() []string         { return nil }
func (listPageEmptyRows) Close() error              { return nil }
func (listPageEmptyRows) Next([]driver.Value) error { return io.EOF }

func TestListPageWorkspaceScope_FailsClosedAndBindsParentChain(t *testing.T) {
	for _, tc := range []struct {
		name       string
		call       func(context.Context, *sql.DB) error
		chain      []string
		staffField string
	}{
		{
			name: "job_task",
			call: func(ctx context.Context, db *sql.DB) error {
				_, err := (&PostgresJobTaskRepository{db: db}).GetJobTaskListPageData(ctx, &jobtaskpb.GetJobTaskListPageDataRequest{})
				return err
			},
			chain:      []string{"jp.id = jt.job_phase_id", "j.id = jp.job_id"},
			staffField: "jt.assigned_to",
		},
		{
			name: "task_outcome",
			call: func(ctx context.Context, db *sql.DB) error {
				_, err := (&PostgresTaskOutcomeRepository{db: db}).GetTaskOutcomeListPageData(ctx, &taskoutcomepb.GetTaskOutcomeListPageDataRequest{})
				return err
			},
			chain:      []string{"jt.id = to_.job_task_id", "jp.id = jt.job_phase_id", "j.id = jp.job_id"},
			staffField: "to_.recorded_by",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := &listPageCapture{}
			db := sql.OpenDB(capture)
			defer db.Close()
			for _, ctx := range []context.Context{
				context.Background(),
				identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{}),
			} {
				if err := tc.call(ctx, db); err == nil {
					t.Fatal("missing workspace must return an error")
				}
				if capture.query != "" {
					t.Fatal("missing workspace reached the database")
				}
			}

			for _, principal := range []struct {
				name string
				id   *identity.RequestIdentity
				want int
			}{
				{"admin", &identity.RequestIdentity{WorkspaceID: "workspace-A"}, 4},
				{"staff", &identity.RequestIdentity{WorkspaceID: "workspace-A", PrincipalType: 7, PrincipalID: "staff-A"}, 5},
			} {
				t.Run(principal.name, func(t *testing.T) {
					capture.query = ""
					capture.args = nil
					ctx := identity.WithRequestIdentity(context.Background(), principal.id)
					if err := tc.call(ctx, db); err != nil {
						t.Fatal(err)
					}
					if len(capture.args) != principal.want || capture.args[len(capture.args)-1].Value != "workspace-A" {
						t.Fatalf("workspace binding: got %v args, last=%v", len(capture.args), capture.args[len(capture.args)-1].Value)
					}
					for _, fragment := range append(tc.chain, "j.workspace_id = $"+string(rune('0'+principal.want)), "SELECT COUNT(*) FROM enriched") {
						if !strings.Contains(capture.query, fragment) {
							t.Fatalf("query lacks %q", fragment)
						}
					}
					if principal.want == 5 {
						if capture.args[3].Value != "staff-A" || !strings.Contains(capture.query, tc.staffField) {
							t.Fatal("staff filter or binding missing")
						}
					}
				})
			}
		})
	}
}
