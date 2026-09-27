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
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

type SupplierContractLineScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *SupplierContractLineScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &SupplierContractLineScope3Conn{c}, nil
}
func (c *SupplierContractLineScope3Capture) Driver() driver.Driver {
	return SupplierContractLineScope3Driver{}
}

type SupplierContractLineScope3Driver struct{}

func (SupplierContractLineScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type SupplierContractLineScope3Conn struct {
	capture *SupplierContractLineScope3Capture
}

func (*SupplierContractLineScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*SupplierContractLineScope3Conn) Close() error { return nil }
func (*SupplierContractLineScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *SupplierContractLineScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return SupplierContractLineScope3Rows{}, nil
}

type SupplierContractLineScope3Rows struct{}

func (SupplierContractLineScope3Rows) Columns() []string         { return nil }
func (SupplierContractLineScope3Rows) Close() error              { return nil }
func (SupplierContractLineScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestSupplierContractLineListPageScope3(t *testing.T) {
	capture := &SupplierContractLineScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresSupplierContractLineRepository(postgresCore.NewWorkspaceAwareOperations(db), "supplier_contract_line")
	call := func(ctx context.Context) error {
		_, err := repo.GetSupplierContractLineListPageData(ctx, &suppliercontractlinepb.GetSupplierContractLineListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = scl.supplier_contract_id`, `p1.workspace_id = $4`, `FROM supplier_contract p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
