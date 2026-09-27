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
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

type LocationAttributeScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *LocationAttributeScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &LocationAttributeScope3Conn{c}, nil
}
func (c *LocationAttributeScope3Capture) Driver() driver.Driver {
	return LocationAttributeScope3Driver{}
}

type LocationAttributeScope3Driver struct{}

func (LocationAttributeScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type LocationAttributeScope3Conn struct {
	capture *LocationAttributeScope3Capture
}

func (*LocationAttributeScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*LocationAttributeScope3Conn) Close() error { return nil }
func (*LocationAttributeScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *LocationAttributeScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return LocationAttributeScope3Rows{}, nil
}

type LocationAttributeScope3Rows struct{}

func (LocationAttributeScope3Rows) Columns() []string         { return nil }
func (LocationAttributeScope3Rows) Close() error              { return nil }
func (LocationAttributeScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestLocationAttributeListPageScope3(t *testing.T) {
	capture := &LocationAttributeScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresLocationAttributeRepository(postgresCore.NewWorkspaceAwareOperations(db), "location_attribute")
	call := func(ctx context.Context) error {
		_, err := repo.GetLocationAttributeListPageData(ctx, &locationattributepb.GetLocationAttributeListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = la.location_id`, `p1.workspace_id = $4`, `FROM location p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
