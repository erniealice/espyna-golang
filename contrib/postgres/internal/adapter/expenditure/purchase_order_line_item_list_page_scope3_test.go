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
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

type PurchaseOrderLineItemScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *PurchaseOrderLineItemScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &PurchaseOrderLineItemScope3Conn{c}, nil
}
func (c *PurchaseOrderLineItemScope3Capture) Driver() driver.Driver {
	return PurchaseOrderLineItemScope3Driver{}
}

type PurchaseOrderLineItemScope3Driver struct{}

func (PurchaseOrderLineItemScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type PurchaseOrderLineItemScope3Conn struct {
	capture *PurchaseOrderLineItemScope3Capture
}

func (*PurchaseOrderLineItemScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*PurchaseOrderLineItemScope3Conn) Close() error { return nil }
func (*PurchaseOrderLineItemScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *PurchaseOrderLineItemScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return PurchaseOrderLineItemScope3Rows{}, nil
}

type PurchaseOrderLineItemScope3Rows struct{}

func (PurchaseOrderLineItemScope3Rows) Columns() []string         { return nil }
func (PurchaseOrderLineItemScope3Rows) Close() error              { return nil }
func (PurchaseOrderLineItemScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestPurchaseOrderLineItemListPageScope3(t *testing.T) {
	capture := &PurchaseOrderLineItemScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresPurchaseOrderLineItemRepository(postgresCore.NewWorkspaceAwareOperations(db), "purchase_order_line_item")
	call := func(ctx context.Context) error {
		_, err := repo.GetPurchaseOrderLineItemListPageData(ctx, &purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = poli.purchase_order_id`, `p2.workspace_id = $4`, `FROM purchase_order p1`, `JOIN supplier p2 ON p2.id = p1.supplier_id`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
