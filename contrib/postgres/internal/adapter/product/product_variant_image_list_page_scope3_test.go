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
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

type ProductVariantImageScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ProductVariantImageScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ProductVariantImageScope3Conn{c}, nil
}
func (c *ProductVariantImageScope3Capture) Driver() driver.Driver {
	return ProductVariantImageScope3Driver{}
}

type ProductVariantImageScope3Driver struct{}

func (ProductVariantImageScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ProductVariantImageScope3Conn struct {
	capture *ProductVariantImageScope3Capture
}

func (*ProductVariantImageScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ProductVariantImageScope3Conn) Close() error { return nil }
func (*ProductVariantImageScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ProductVariantImageScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ProductVariantImageScope3Rows{}, nil
}

type ProductVariantImageScope3Rows struct{}

func (ProductVariantImageScope3Rows) Columns() []string         { return nil }
func (ProductVariantImageScope3Rows) Close() error              { return nil }
func (ProductVariantImageScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestProductVariantImageListPageScope3(t *testing.T) {
	capture := &ProductVariantImageScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresProductVariantImageRepository(postgresCore.NewWorkspaceAwareOperations(db), "product_variant_image")
	call := func(ctx context.Context) error {
		_, err := repo.GetProductVariantImageListPageData(ctx, &productvariantimagepb.GetProductVariantImageListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pvi.product_variant_id`, `p2.workspace_id = $4`, `FROM product_variant p1`, `JOIN product p2 ON p2.id = p1.product_id`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
