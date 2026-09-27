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
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

type ProductAttributeScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ProductAttributeScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ProductAttributeScope3Conn{c}, nil
}
func (c *ProductAttributeScope3Capture) Driver() driver.Driver { return ProductAttributeScope3Driver{} }

type ProductAttributeScope3Driver struct{}

func (ProductAttributeScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ProductAttributeScope3Conn struct {
	capture *ProductAttributeScope3Capture
}

func (*ProductAttributeScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ProductAttributeScope3Conn) Close() error { return nil }
func (*ProductAttributeScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ProductAttributeScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ProductAttributeScope3Rows{}, nil
}

type ProductAttributeScope3Rows struct{}

func (ProductAttributeScope3Rows) Columns() []string         { return nil }
func (ProductAttributeScope3Rows) Close() error              { return nil }
func (ProductAttributeScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestProductAttributeListPageScope3(t *testing.T) {
	capture := &ProductAttributeScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresProductAttributeRepository(postgresCore.NewWorkspaceAwareOperations(db), "product_attribute")
	call := func(ctx context.Context) error {
		_, err := repo.GetProductAttributeListPageData(ctx, &productattributepb.GetProductAttributeListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pa.product_id`, `p1.workspace_id = $4`, `FROM product p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
