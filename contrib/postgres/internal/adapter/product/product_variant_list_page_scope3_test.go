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
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

type ProductVariantScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ProductVariantScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ProductVariantScope3Conn{c}, nil
}
func (c *ProductVariantScope3Capture) Driver() driver.Driver { return ProductVariantScope3Driver{} }

type ProductVariantScope3Driver struct{}

func (ProductVariantScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ProductVariantScope3Conn struct{ capture *ProductVariantScope3Capture }

func (*ProductVariantScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ProductVariantScope3Conn) Close() error { return nil }
func (*ProductVariantScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ProductVariantScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ProductVariantScope3Rows{}, nil
}

type ProductVariantScope3Rows struct{}

func (ProductVariantScope3Rows) Columns() []string         { return nil }
func (ProductVariantScope3Rows) Close() error              { return nil }
func (ProductVariantScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestProductVariantListPageScope3(t *testing.T) {
	capture := &ProductVariantScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresProductVariantRepository(postgresCore.NewWorkspaceAwareOperations(db), "product_variant")
	call := func(ctx context.Context) error {
		_, err := repo.GetProductVariantListPageData(ctx, &productvariantpb.GetProductVariantListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pv.product_id`, `p1.workspace_id = $4`, `FROM product p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
