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
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type CostPlanScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *CostPlanScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &CostPlanScope3Conn{c}, nil
}
func (c *CostPlanScope3Capture) Driver() driver.Driver { return CostPlanScope3Driver{} }

type CostPlanScope3Driver struct{}

func (CostPlanScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type CostPlanScope3Conn struct{ capture *CostPlanScope3Capture }

func (*CostPlanScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*CostPlanScope3Conn) Close() error { return nil }
func (*CostPlanScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *CostPlanScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return CostPlanScope3Rows{}, nil
}

type CostPlanScope3Rows struct{}

func (CostPlanScope3Rows) Columns() []string         { return nil }
func (CostPlanScope3Rows) Close() error              { return nil }
func (CostPlanScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestCostPlanListPageScope3(t *testing.T) {
	capture := &CostPlanScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresCostPlanRepository(postgresCore.NewWorkspaceAwareOperations(db), "cost_plan")
	call := func(ctx context.Context) error {
		_, err := repo.GetCostPlanListPageData(ctx, &costplanpb.GetCostPlanListPageDataRequest{})
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
	for _, fragment := range []string{`cp.workspace_id = $4`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
