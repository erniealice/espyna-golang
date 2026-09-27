//go:build postgresql

package entity

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
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

type ClientAttributeScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ClientAttributeScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ClientAttributeScope3Conn{c}, nil
}
func (c *ClientAttributeScope3Capture) Driver() driver.Driver { return ClientAttributeScope3Driver{} }

type ClientAttributeScope3Driver struct{}

func (ClientAttributeScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ClientAttributeScope3Conn struct{ capture *ClientAttributeScope3Capture }

func (*ClientAttributeScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ClientAttributeScope3Conn) Close() error { return nil }
func (*ClientAttributeScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ClientAttributeScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ClientAttributeScope3Rows{}, nil
}

type ClientAttributeScope3Rows struct{}

func (ClientAttributeScope3Rows) Columns() []string         { return nil }
func (ClientAttributeScope3Rows) Close() error              { return nil }
func (ClientAttributeScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestClientAttributeListPageScope3(t *testing.T) {
	capture := &ClientAttributeScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresClientAttributeRepository(postgresCore.NewWorkspaceAwareOperations(db), "client_attribute")
	call := func(ctx context.Context) error {
		_, err := repo.GetClientAttributeListPageData(ctx, &clientattributepb.GetClientAttributeListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = ca.client_id`, `p1.workspace_id = $4`, `FROM client p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
