//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
	_ "github.com/lib/pq"
)

func TestListPageScope4RequiresWorkspaceBeforeQuery(t *testing.T) {
	fx := &userSecurityFixture{queryErr: errors.New("query-hit")}
	checks := []struct {
		name string
		call func(context.Context) error
	}{
		{"staff", func(ctx context.Context) error {
			_, err := (&PostgresStaffRepository{dbOps: fx}).GetStaffListPageData(ctx, &staffpb.GetStaffListPageDataRequest{})
			return err
		}},
		{"payment_term", func(ctx context.Context) error {
			_, err := (&PostgresPaymentTermRepository{dbOps: fx}).GetPaymentTermListPageData(ctx, &paymenttermpb.GetPaymentTermListPageDataRequest{})
			return err
		}},
		{"client_workspace_user", func(ctx context.Context) error {
			_, err := (&PostgresClientWorkspaceUserRepository{}).GetClientWorkspaceUserListPageData(ctx, &clientworkspaceuserpb.GetClientWorkspaceUserListPageDataRequest{})
			return err
		}},
		{"role_permission", func(ctx context.Context) error {
			_, err := (&PostgresRolePermissionRepository{dbOps: fx}).GetRolePermissionListPageData(ctx, &rolepermissionpb.GetRolePermissionListPageDataRequest{})
			return err
		}},
		{"workspace_user_role", func(ctx context.Context) error {
			_, err := (&PostgresWorkspaceUserRoleRepository{dbOps: fx}).GetWorkspaceUserRoleListPageData(ctx, &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{})
			return err
		}},
	}
	for _, tc := range checks {
		for _, ctx := range []context.Context{context.Background(), identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{})} {
			t.Run(tc.name, func(t *testing.T) {
				before := fx.queryCalls
				if err := tc.call(ctx); err == nil || strings.Contains(err.Error(), "query-hit") {
					t.Fatalf("error = %v, want missing workspace", err)
				}
				if fx.queryCalls != before {
					t.Fatalf("query calls = %d, want %d", fx.queryCalls, before)
				}
			})
		}
	}
}

// The caller workspace comes from a live, read-only lane. The direct SQL oracle
// checks both preservation and absence of foreign rows, including empty pages.
func TestListPageScope4LiveWorkspaceParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only DB required: mode=%q err=%v", readOnly, err)
	}
	wsRows, err := db.QueryContext(ctx, `SELECT id FROM workspace ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	var workspaces []string
	for wsRows.Next() {
		var ws string
		if err := wsRows.Scan(&ws); err != nil {
			t.Fatal(err)
		}
		workspaces = append(workspaces, ws)
	}
	if err := wsRows.Err(); err != nil {
		t.Fatal(err)
	}
	wsRows.Close()
	if len(workspaces) == 0 {
		t.Skip("no workspaces")
	}
	kit := postgresCore.NewWorkspaceAwareOperations(db)
	sortID := &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
	page := &commonpb.PaginationRequest{Limit: 50, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 1}}}
	checks := []struct {
		name, oracle string
		read         func(context.Context) ([]string, error)
	}{
		{"staff", `SELECT id FROM staff WHERE active = true AND workspace_id = $1 ORDER BY id LIMIT 50`, func(c context.Context) ([]string, error) {
			r, e := NewPostgresStaffRepository(kit, "staff").GetStaffListPageData(c, &staffpb.GetStaffListPageDataRequest{Sort: sortID, Pagination: page})
			if e != nil {
				return nil, e
			}
			var ids []string
			for _, v := range r.StaffList {
				ids = append(ids, v.Id)
			}
			return ids, nil
		}},
		{"payment_term", `SELECT id FROM payment_term WHERE entity_scope IN ('client','both') AND workspace_id = $1 ORDER BY id LIMIT 50`, func(c context.Context) ([]string, error) {
			r, e := NewPostgresPaymentTermRepository(kit, "payment_term").GetPaymentTermListPageData(c, &paymenttermpb.GetPaymentTermListPageDataRequest{Sort: sortID, Pagination: page})
			if e != nil {
				return nil, e
			}
			var ids []string
			for _, v := range r.PaymentTermList {
				ids = append(ids, v.Id)
			}
			return ids, nil
		}},
		{"client_workspace_user", `SELECT id FROM client_workspace_user WHERE active = true AND workspace_id = $1 ORDER BY date_created DESC, id ASC LIMIT 50`, func(c context.Context) ([]string, error) {
			r, e := NewPostgresClientWorkspaceUserRepository(kit, "client_workspace_user").GetClientWorkspaceUserListPageData(c, &clientworkspaceuserpb.GetClientWorkspaceUserListPageDataRequest{Pagination: page})
			if e != nil {
				return nil, e
			}
			var ids []string
			for _, v := range r.ClientWorkspaceUserList {
				ids = append(ids, v.Id)
			}
			return ids, nil
		}},
		{"role_permission", `SELECT rp.id FROM role_permission rp JOIN role r ON r.id=rp.role_id WHERE rp.active = true AND r.workspace_id = $1 ORDER BY rp.id LIMIT 50`, func(c context.Context) ([]string, error) {
			r, e := NewPostgresRolePermissionRepository(kit, "role_permission").GetRolePermissionListPageData(c, &rolepermissionpb.GetRolePermissionListPageDataRequest{Sort: sortID, Pagination: page})
			if e != nil {
				return nil, e
			}
			var ids []string
			for _, v := range r.RolePermissionList {
				ids = append(ids, v.Id)
			}
			return ids, nil
		}},
		{"workspace_user_role", `SELECT wur.id FROM workspace_user_role wur JOIN workspace_user wu ON wu.id=wur.workspace_user_id WHERE wur.active = true AND wu.workspace_id = $1 ORDER BY wur.id LIMIT 50`, func(c context.Context) ([]string, error) {
			r, e := NewPostgresWorkspaceUserRoleRepository(kit, "workspace_user_role").GetWorkspaceUserRoleListPageData(c, &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{Sort: sortID, Pagination: page})
			if e != nil {
				return nil, e
			}
			var ids []string
			for _, v := range r.WorkspaceUserRoleList {
				ids = append(ids, v.Id)
			}
			return ids, nil
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for _, ws := range workspaces {
				rows, err := db.QueryContext(ctx, check.oracle, ws)
				if err != nil {
					t.Fatal(err)
				}
				var want []string
				for rows.Next() {
					var id string
					if err := rows.Scan(&id); err != nil {
						t.Fatal(err)
					}
					want = append(want, id)
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				rows.Close()
				got, err := check.read(identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: ws}))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(got, want) {
					t.Fatalf("workspace page parity: got %d rows, want %d", len(got), len(want))
				}
			}
			t.Logf("matched direct scoped oracle for %d workspaces", len(workspaces))
		})
	}
}

func TestListPageScope4PredicateAndBind(t *testing.T) {
	const workspace = "scope4-workspace"
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
	fx := &userSecurityFixture{queryErr: errors.New("query-hit")}
	checks := []struct {
		name, predicate, relation string
		bind                      int
		call                      func() error
	}{
		{"staff", "s.workspace_id = $4::text", "FROM staff s", 3, func() error {
			_, err := (&PostgresStaffRepository{dbOps: fx}).GetStaffListPageData(ctx, &staffpb.GetStaffListPageDataRequest{})
			return err
		}},
		{"payment_term", "workspace_id = $4::text", "FROM payment_term", 3, func() error {
			_, err := (&PostgresPaymentTermRepository{dbOps: fx, tableName: "payment_term"}).GetPaymentTermListPageData(ctx, &paymenttermpb.GetPaymentTermListPageDataRequest{})
			return err
		}},
		{"role_permission", "r.workspace_id = $3::text", "JOIN role r ON r.id = rp.role_id", 2, func() error {
			_, err := (&PostgresRolePermissionRepository{dbOps: fx}).GetRolePermissionListPageData(ctx, &rolepermissionpb.GetRolePermissionListPageDataRequest{})
			return err
		}},
		{"workspace_user_role", "wu.workspace_id = $4::text", "JOIN workspace_user wu ON wu.id = wur.workspace_user_id", 3, func() error {
			_, err := (&PostgresWorkspaceUserRoleRepository{dbOps: fx}).GetWorkspaceUserRoleListPageData(ctx, &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{})
			return err
		}},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			before := fx.queryCalls
			if err := tc.call(); err == nil || !strings.Contains(err.Error(), "query-hit") {
				t.Fatalf("error = %v, want executor error", err)
			}
			if fx.queryCalls != before+1 || !strings.Contains(fx.query, tc.predicate) || !strings.Contains(fx.query, tc.relation) {
				t.Fatalf("query calls = %d, predicate/relation missing in %q", fx.queryCalls, fx.query)
			}
			if len(fx.queryArgs) <= tc.bind || fx.queryArgs[tc.bind] != workspace {
				t.Fatalf("bind at %d = %v, want %q", tc.bind, fx.queryArgs, workspace)
			}
			if strings.Contains(fx.query, "$4::text = '' OR") {
				t.Fatal("workspace wildcard remains")
			}
		})
	}
}
