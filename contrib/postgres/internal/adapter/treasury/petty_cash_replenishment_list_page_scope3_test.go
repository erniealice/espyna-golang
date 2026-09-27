//go:build postgresql

package treasury

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
	pettycashreplenishmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_replenishment"
)

type PettyCashReplenishmentScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *PettyCashReplenishmentScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &PettyCashReplenishmentScope3Conn{c}, nil
}
func (c *PettyCashReplenishmentScope3Capture) Driver() driver.Driver {
	return PettyCashReplenishmentScope3Driver{}
}

type PettyCashReplenishmentScope3Driver struct{}

func (PettyCashReplenishmentScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type PettyCashReplenishmentScope3Conn struct {
	capture *PettyCashReplenishmentScope3Capture
}

func (*PettyCashReplenishmentScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*PettyCashReplenishmentScope3Conn) Close() error { return nil }
func (*PettyCashReplenishmentScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *PettyCashReplenishmentScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return PettyCashReplenishmentScope3Rows{}, nil
}

type PettyCashReplenishmentScope3Rows struct{}

func (PettyCashReplenishmentScope3Rows) Columns() []string         { return nil }
func (PettyCashReplenishmentScope3Rows) Close() error              { return nil }
func (PettyCashReplenishmentScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestPettyCashReplenishmentListPageScope3(t *testing.T) {
	capture := &PettyCashReplenishmentScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresPettyCashReplenishmentRepository(postgresCore.NewWorkspaceAwareOperations(db), "petty_cash_replenishment")
	call := func(ctx context.Context) error {
		_, err := repo.GetPettyCashReplenishmentListPageData(ctx, &pettycashreplenishmentpb.GetPettyCashReplenishmentListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = pcr.fund_id`, `p2.workspace_id = $4`, `FROM petty_cash_fund p1`, `JOIN location p2 ON p2.id = p1.location_id`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
