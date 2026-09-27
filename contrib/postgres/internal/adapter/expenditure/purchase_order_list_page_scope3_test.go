//go:build postgresql

package expenditure

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
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

type PurchaseOrderScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *PurchaseOrderScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &PurchaseOrderScope3Conn{c}, nil
}
func (c *PurchaseOrderScope3Capture) Driver() driver.Driver { return PurchaseOrderScope3Driver{} }

type PurchaseOrderScope3Driver struct{}

func (PurchaseOrderScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type PurchaseOrderScope3Conn struct{ capture *PurchaseOrderScope3Capture }

func (*PurchaseOrderScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*PurchaseOrderScope3Conn) Close() error { return nil }
func (*PurchaseOrderScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *PurchaseOrderScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return PurchaseOrderScope3Rows{}, nil
}

type PurchaseOrderScope3Rows struct{}

func (PurchaseOrderScope3Rows) Columns() []string         { return nil }
func (PurchaseOrderScope3Rows) Close() error              { return nil }
func (PurchaseOrderScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestPurchaseOrderListPageScope3(t *testing.T) {
	capture := &PurchaseOrderScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresPurchaseOrderRepository(postgresCore.NewWorkspaceAwareOperations(db), "purchase_order")
	call := func(ctx context.Context) error {
		_, err := repo.GetPurchaseOrderListPageData(ctx, &purchaseorderpb.GetPurchaseOrderListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = po.supplier_id`, `p1.workspace_id = $4`, `FROM supplier p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
