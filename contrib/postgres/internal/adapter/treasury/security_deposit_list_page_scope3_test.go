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
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

type SecurityDepositScope3Capture struct {
	query string
	args  []driver.NamedValue
	calls int
}

func (c *SecurityDepositScope3Capture) Connect(context.Context) (driver.Conn, error) {
	return &SecurityDepositScope3Conn{c}, nil
}
func (c *SecurityDepositScope3Capture) Driver() driver.Driver { return SecurityDepositScope3Driver{} }

type SecurityDepositScope3Driver struct{}

func (SecurityDepositScope3Driver) Open(string) (driver.Conn, error) {
	return nil, errors.New("DSN unsupported")
}

type SecurityDepositScope3Conn struct{ capture *SecurityDepositScope3Capture }

func (*SecurityDepositScope3Conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*SecurityDepositScope3Conn) Close() error { return nil }
func (*SecurityDepositScope3Conn) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction unsupported")
}
func (c *SecurityDepositScope3Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.calls++
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return SecurityDepositScope3Rows{}, nil
}

type SecurityDepositScope3Rows struct{}

func (SecurityDepositScope3Rows) Columns() []string         { return nil }
func (SecurityDepositScope3Rows) Close() error              { return nil }
func (SecurityDepositScope3Rows) Next([]driver.Value) error { return io.EOF }

func TestSecurityDepositListPageScope3(t *testing.T) {
	capture := &SecurityDepositScope3Capture{}
	db := sql.OpenDB(capture)
	defer db.Close()
	repo := NewPostgresSecurityDepositRepository(postgresCore.NewWorkspaceAwareOperations(db), "security_deposit")
	call := func(ctx context.Context) error {
		_, err := repo.GetSecurityDepositListPageData(ctx, &securitydepositpb.GetSecurityDepositListPageDataRequest{})
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
	for _, fragment := range []string{`p1.id = sd.account_id`, `p1.workspace_id = $4`, `FROM account p1`} {
		if !strings.Contains(capture.query, fragment) {
			t.Fatalf("SQL lacks %q", fragment)
		}
	}
}
