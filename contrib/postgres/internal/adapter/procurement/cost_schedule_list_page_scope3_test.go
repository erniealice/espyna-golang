//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type CostScheduleScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *CostScheduleScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &CostScheduleScope3Conn{c}, nil
}
func (c *CostScheduleScope3Capture) Driver() driver.Driver { return CostScheduleScope3Driver{} }

type CostScheduleScope3Driver struct{}

func (CostScheduleScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type CostScheduleScope3Conn struct{ capture *CostScheduleScope3Capture }

func (*CostScheduleScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*CostScheduleScope3Conn) Close() error { return nil }
func (*CostScheduleScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *CostScheduleScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return CostScheduleScope3Rows{}, nil
}

type CostScheduleScope3Rows struct{}

func (CostScheduleScope3Rows) Columns() []string         { return nil }
func (CostScheduleScope3Rows) Close() error              { return nil }
func (CostScheduleScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestCostScheduleListPageScope3(t *testing.T) {
	capture := &CostScheduleScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresCostScheduleRepository(postgresCore.NewWorkspaceAwareOperations(db), "cost_schedule")
	call := func(ctx context.Context) error {
		_, err := repo.GetCostScheduleListPageData(ctx, &costschedulepb.GetCostScheduleListPageDataRequest{})
		return err
	}
	for _, tc := range []struct {
		ctx  context.Context
		want error
	}{
		{context.Background(), identity.ErrIdentityNotInContext},
		{identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{}), identity.ErrWorkspaceNotSelected},
	} {
		if err := call(tc.ctx); !errors.Is(err, tc.want) {
			t.Fatalf("missing workspace: got %v, want %v", err, tc.want)
		}
		if capture.calls != 0 {
			t.Fatalf("missing workspace made %d DB calls", capture.calls)
		}
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "workspace-A"})
	if err := call(ctx); err != nil {
		t.Fatal(err)
	}
	if capture.calls != 1 {
		t.Fatalf("scoped request made %d DB calls, want 1", capture.calls)
	}
	if len(capture.args) != 4 || capture.args[3].Value != "workspace-A" {
		t.Fatalf("workspace bind: %#v", capture.args)
	}
	for _, fragment := range []string{`cs.workspace_id = $4`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
