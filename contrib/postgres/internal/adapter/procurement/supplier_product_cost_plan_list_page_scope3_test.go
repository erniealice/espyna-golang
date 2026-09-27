//go:build postgresql

package procurement

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
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
)

type SupplierProductCostPlanScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *SupplierProductCostPlanScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &SupplierProductCostPlanScope3Conn{c}, nil
}
func (c *SupplierProductCostPlanScope3Capture) Driver() driver.Driver {
	return SupplierProductCostPlanScope3Driver{}
}

type SupplierProductCostPlanScope3Driver struct{}

func (SupplierProductCostPlanScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type SupplierProductCostPlanScope3Conn struct {
	capture *SupplierProductCostPlanScope3Capture
}

func (*SupplierProductCostPlanScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*SupplierProductCostPlanScope3Conn) Close() error { return nil }
func (*SupplierProductCostPlanScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *SupplierProductCostPlanScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return SupplierProductCostPlanScope3Rows{}, nil
}

type SupplierProductCostPlanScope3Rows struct{}

func (SupplierProductCostPlanScope3Rows) Columns() []string         { return nil }
func (SupplierProductCostPlanScope3Rows) Close() error              { return nil }
func (SupplierProductCostPlanScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestSupplierProductCostPlanListPageScope3(t *testing.T) {
	capture := &SupplierProductCostPlanScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresSupplierProductCostPlanRepository(postgresCore.NewWorkspaceAwareOperations(db), "supplier_product_cost_plan")
	call := func(ctx context.Context) error {
		_, err := repo.GetSupplierProductCostPlanListPageData(ctx, &supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataRequest{})
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
	if len(capture.args) != 3 || capture.args[2].Value != "workspace-A" {
		t.Fatalf("workspace bind: %#v", capture.args)
	}
	for _, fragment := range []string{`p1.id = spcp.supplier_product_plan_id`, `p2.workspace_id = $3`, `FROM supplier_product_plan p1`, `JOIN supplier_plan p2 ON p2.id = p1.supplier_plan_id`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
