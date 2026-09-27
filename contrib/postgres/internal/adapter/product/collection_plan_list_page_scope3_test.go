//go:build postgresql

package product

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
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

type CollectionPlanScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *CollectionPlanScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &CollectionPlanScope3Conn{c}, nil
}
func (c *CollectionPlanScope3Capture) Driver() driver.Driver { return CollectionPlanScope3Driver{} }

type CollectionPlanScope3Driver struct{}

func (CollectionPlanScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type CollectionPlanScope3Conn struct{ capture *CollectionPlanScope3Capture }

func (*CollectionPlanScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*CollectionPlanScope3Conn) Close() error { return nil }
func (*CollectionPlanScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *CollectionPlanScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return CollectionPlanScope3Rows{}, nil
}

type CollectionPlanScope3Rows struct{}

func (CollectionPlanScope3Rows) Columns() []string         { return nil }
func (CollectionPlanScope3Rows) Close() error              { return nil }
func (CollectionPlanScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestCollectionPlanListPageScope3(t *testing.T) {
	capture := &CollectionPlanScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresCollectionPlanRepository(postgresCore.NewWorkspaceAwareOperations(db), "collection_plan")
	call := func(ctx context.Context) error {
		_, err := repo.GetCollectionPlanListPageData(ctx, &collectionplanpb.GetCollectionPlanListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = cp.plan_id`, `p1.workspace_id = $4`, `FROM plan p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
