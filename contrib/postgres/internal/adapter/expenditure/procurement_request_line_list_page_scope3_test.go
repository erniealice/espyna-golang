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
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

type ProcurementRequestLineScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *ProcurementRequestLineScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &ProcurementRequestLineScope3Conn{c}, nil
}
func (c *ProcurementRequestLineScope3Capture) Driver() driver.Driver {
	return ProcurementRequestLineScope3Driver{}
}

type ProcurementRequestLineScope3Driver struct{}

func (ProcurementRequestLineScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type ProcurementRequestLineScope3Conn struct {
	capture *ProcurementRequestLineScope3Capture
}

func (*ProcurementRequestLineScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*ProcurementRequestLineScope3Conn) Close() error { return nil }
func (*ProcurementRequestLineScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *ProcurementRequestLineScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return ProcurementRequestLineScope3Rows{}, nil
}

type ProcurementRequestLineScope3Rows struct{}

func (ProcurementRequestLineScope3Rows) Columns() []string         { return nil }
func (ProcurementRequestLineScope3Rows) Close() error              { return nil }
func (ProcurementRequestLineScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestProcurementRequestLineListPageScope3(t *testing.T) {
	capture := &ProcurementRequestLineScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresProcurementRequestLineRepository(postgresCore.NewWorkspaceAwareOperations(db), "procurement_request_line")
	call := func(ctx context.Context) error {
		_, err := repo.GetProcurementRequestLineListPageData(ctx, &procurementrequestlinepb.GetProcurementRequestLineListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = prl.procurement_request_id`, `p1.workspace_id = $4`, `FROM procurement_request p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
