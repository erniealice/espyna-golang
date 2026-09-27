//go:build postgresql

package revenue

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
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

type DeferredRevenueScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *DeferredRevenueScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &DeferredRevenueScope3Conn{c}, nil
}
func (c *DeferredRevenueScope3Capture) Driver() driver.Driver { return DeferredRevenueScope3Driver{} }

type DeferredRevenueScope3Driver struct{}

func (DeferredRevenueScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type DeferredRevenueScope3Conn struct{ capture *DeferredRevenueScope3Capture }

func (*DeferredRevenueScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*DeferredRevenueScope3Conn) Close() error { return nil }
func (*DeferredRevenueScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *DeferredRevenueScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return DeferredRevenueScope3Rows{}, nil
}

type DeferredRevenueScope3Rows struct{}

func (DeferredRevenueScope3Rows) Columns() []string         { return nil }
func (DeferredRevenueScope3Rows) Close() error              { return nil }
func (DeferredRevenueScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestDeferredRevenueListPageScope3(t *testing.T) {
	capture := &DeferredRevenueScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresDeferredRevenueRepository(postgresCore.NewWorkspaceAwareOperations(db), "deferred_revenue")
	call := func(ctx context.Context) error {
		_, err := repo.GetDeferredRevenueListPageData(ctx, &deferredrevenuepb.GetDeferredRevenueListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = dr.liability_account_id`, `p1.workspace_id = $4`, `FROM account p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
