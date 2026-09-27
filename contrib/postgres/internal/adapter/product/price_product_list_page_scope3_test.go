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
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

type PriceProductScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *PriceProductScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &PriceProductScope3Conn{c}, nil
}
func (c *PriceProductScope3Capture) Driver() driver.Driver { return PriceProductScope3Driver{} }

type PriceProductScope3Driver struct{}

func (PriceProductScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type PriceProductScope3Conn struct{ capture *PriceProductScope3Capture }

func (*PriceProductScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*PriceProductScope3Conn) Close() error { return nil }
func (*PriceProductScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *PriceProductScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return PriceProductScope3Rows{}, nil
}

type PriceProductScope3Rows struct{}

func (PriceProductScope3Rows) Columns() []string         { return nil }
func (PriceProductScope3Rows) Close() error              { return nil }
func (PriceProductScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestPriceProductListPageScope3(t *testing.T) {
	capture := &PriceProductScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresPriceProductRepository(postgresCore.NewWorkspaceAwareOperations(db), "price_product")
	call := func(ctx context.Context) error {
		_, err := repo.GetPriceProductListPageData(ctx, &priceproductpb.GetPriceProductListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pp.product_id`, `p1.workspace_id = $4`, `FROM product p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
