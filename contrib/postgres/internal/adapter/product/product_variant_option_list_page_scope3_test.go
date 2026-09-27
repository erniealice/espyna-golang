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
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

type ProductVariantOptionScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ProductVariantOptionScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ProductVariantOptionScope3Conn{c}, nil
}
func (c *ProductVariantOptionScope3Capture) Driver() driver.Driver {
	return ProductVariantOptionScope3Driver{}
}

type ProductVariantOptionScope3Driver struct{}

func (ProductVariantOptionScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ProductVariantOptionScope3Conn struct {
	capture *ProductVariantOptionScope3Capture
}

func (*ProductVariantOptionScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ProductVariantOptionScope3Conn) Close() error { return nil }
func (*ProductVariantOptionScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ProductVariantOptionScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ProductVariantOptionScope3Rows{}, nil
}

type ProductVariantOptionScope3Rows struct{}

func (ProductVariantOptionScope3Rows) Columns() []string         { return nil }
func (ProductVariantOptionScope3Rows) Close() error              { return nil }
func (ProductVariantOptionScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestProductVariantOptionListPageScope3(t *testing.T) {
	capture := &ProductVariantOptionScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresProductVariantOptionRepository(postgresCore.NewWorkspaceAwareOperations(db), "product_variant_option")
	call := func(ctx context.Context) error {
		_, err := repo.GetProductVariantOptionListPageData(ctx, &productvariantoptionpb.GetProductVariantOptionListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pvo.product_variant_id`, `p2.workspace_id = $4`, `FROM product_variant p1`, `JOIN product p2 ON p2.id = p1.product_id`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
