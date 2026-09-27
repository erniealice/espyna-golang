//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	_ "github.com/lib/pq"
)

type cursorPageResult struct {
	ids  []string
	page *commonpb.PaginationResponse
}

func cursorOffset(limit, page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{Limit: limit, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}}
}

func cursorToken(limit int32, token string) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{Limit: limit, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}
}

func TestEntityPageCursor_LiveOffsetParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for read-only parity")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	client := NewPostgresClientRepository(ops, "client")
	location := NewPostgresLocationRepository(ops, "location")
	role := NewPostgresRoleRepository(ops, "role")
	supplier := NewPostgresSupplierRepository(ops, "supplier")
	workspaceUser := NewPostgresWorkspaceUserRepository(ops, "workspace_user")
	delegateClient := NewPostgresDelegateClientRepository(ops, "delegate_client")
	delegate := NewPostgresDelegateRepository(ops, "delegate")
	readers := []struct {
		name, scopeSQL string
		limit          int32
		read           func(context.Context, *commonpb.PaginationRequest) (cursorPageResult, error)
	}{
		{"client", "SELECT workspace_id FROM client WHERE id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 7, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := client.GetClientListPageData(ctx, &clientpb.GetClientListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.ClientList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"location", "SELECT workspace_id FROM location GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 2, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := location.GetLocationListPageData(ctx, &locationpb.GetLocationListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.LocationList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"role", "SELECT workspace_id FROM role GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 2, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := role.GetRoleListPageData(ctx, &rolepb.GetRoleListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.RoleList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"supplier", "SELECT workspace_id FROM supplier GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 2, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := supplier.GetSupplierListPageData(ctx, &supplierpb.GetSupplierListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.SupplierList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"workspace_user", "SELECT workspace_id FROM workspace_user WHERE active GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 7, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := workspaceUser.GetWorkspaceUserListPageData(ctx, &workspaceuserpb.GetWorkspaceUserListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.WorkspaceUserList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"delegate_client", "SELECT workspace_id FROM delegate_client WHERE active GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 2, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := delegateClient.GetDelegateClientListPageData(ctx, &delegateclientpb.GetDelegateClientListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.DelegateClientList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
		{"delegate", "SELECT workspace_id FROM delegate_client WHERE active GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1", 1, func(ctx context.Context, p *commonpb.PaginationRequest) (cursorPageResult, error) {
			r, e := delegate.GetDelegateListPageData(ctx, &delegatepb.GetDelegateListPageDataRequest{Pagination: p})
			if e != nil {
				return cursorPageResult{}, e
			}
			out := cursorPageResult{page: r.Pagination}
			for _, x := range r.DelegateList {
				out.ids = append(out.ids, x.Id)
			}
			return out, nil
		}},
	}
	for _, tc := range readers {
		t.Run(tc.name, func(t *testing.T) {
			var workspace string
			if err := db.QueryRowContext(ctx, tc.scopeSQL).Scan(&workspace); err != nil {
				t.Skipf("no workspace rows: %v", err)
			}
			scoped := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspace})
			first, err := tc.read(scoped, cursorOffset(tc.limit, 1))
			if err != nil {
				t.Fatal(err)
			}
			for _, token := range []string{"malformed", "offset:0"} {
				fallback, err := tc.read(scoped, cursorToken(tc.limit, token))
				if err != nil {
					t.Fatal(err)
				}
				if fallback.page.GetCurrentPage() != 1 || !slices.Equal(fallback.ids, first.ids) {
					t.Fatalf("token %q: got %v, want page 1 %v", token, fallback.ids, first.ids)
				}
			}
			if first.page.GetTotalPages() < 2 {
				t.Skipf("only %d page(s)", first.page.GetTotalPages())
			}
			pages := []cursorPageResult{first}
			keysetPages := 0
			for n := int32(2); n <= first.page.GetTotalPages(); n++ {
				previous := pages[len(pages)-1]
				if previous.page.GetNextCursor() == "" {
					t.Fatalf("page %d missing next cursor", n-1)
				}
				if strings.HasPrefix(previous.page.GetNextCursor(), "k1:") {
					keysetPages++
				}
				got, err := tc.read(scoped, cursorToken(tc.limit, previous.page.GetNextCursor()))
				if err != nil {
					t.Fatal(err)
				}
				want, err := tc.read(scoped, cursorOffset(tc.limit, n))
				if err != nil {
					t.Fatal(err)
				}
				if got.page.GetCurrentPage() != n || !slices.Equal(got.ids, want.ids) {
					t.Fatalf("next page %d: got %v want %v", n, got.ids, want.ids)
				}
				pages = append(pages, got)
			}
			for n := len(pages) - 1; n > 0; n-- {
				if pages[n].page.GetPrevCursor() == "" {
					t.Fatalf("page %d missing prev cursor", n+1)
				}
				got, err := tc.read(scoped, cursorToken(tc.limit, pages[n].page.GetPrevCursor()))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(got.ids, pages[n-1].ids) {
					t.Fatalf("prev page %d: got %v want %v", n, got.ids, pages[n-1].ids)
				}
			}
			t.Logf("%s: %d ids, %d pages, %d keyset Next pages, Next/Prev exact", tc.name, first.page.GetTotalItems(), len(pages), keysetPages)
			if tc.name == "client" && keysetPages == 0 {
				t.Fatal("client UUID cohort did not exercise keyset")
			}
		})
	}
}

func TestEntityPageCursor_ForeignBoundary(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var readOnly string
	if err := db.QueryRowContext(context.Background(), "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	rows, err := db.QueryContext(context.Background(), `SELECT workspace_id, id FROM client WHERE id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' ORDER BY workspace_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	byWorkspace := map[string][]string{}
	for rows.Next() {
		var ws, id string
		if err := rows.Scan(&ws, &id); err != nil {
			t.Fatal(err)
		}
		byWorkspace[ws] = append(byWorkspace[ws], id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(byWorkspace) < 2 {
		t.Skip("need two populated UUID workspaces")
	}
	var caller, foreignID string
	maxCount := 0
	for ws, ids := range byWorkspace {
		if len(ids) > maxCount {
			caller, maxCount = ws, len(ids)
		}
	}
	if caller == "" || maxCount < 3 {
		t.Skip("no caller workspace has a populated second page")
	}
	for ws, ids := range byWorkspace {
		if ws != caller && len(ids) > 0 {
			foreignID = ids[0]
			break
		}
	}
	if foreignID == "" {
		t.Skip("no foreign UUID boundary")
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: caller})
	client := NewPostgresClientRepository(postgresCore.NewWorkspaceAwareOperations(db), "client")
	read := func(p *commonpb.PaginationRequest) cursorPageResult {
		r, err := client.GetClientListPageData(ctx, &clientpb.GetClientListPageDataRequest{Pagination: p})
		if err != nil {
			t.Fatal(err)
		}
		out := cursorPageResult{page: r.Pagination}
		for _, x := range r.ClientList {
			out.ids = append(out.ids, x.Id)
		}
		return out
	}
	want := read(cursorOffset(2, 2))
	got := read(cursorToken(2, postgresCore.EncodePageCursor(2, "next", foreignID)))
	if !slices.Equal(got.ids, want.ids) || got.page.GetCurrentPage() != 2 {
		t.Fatalf("foreign boundary: got %v want offset page 2 %v", got.ids, want.ids)
	}
	for _, id := range got.ids {
		if id == foreignID {
			t.Fatal("foreign row leaked")
		}
	}
	t.Logf("foreign boundary: %d caller rows, 0 foreign rows, offset page 2 fallback", len(got.ids))
}

func TestClientPageCursor_StaffScopeBoundary(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var readOnly, workspace, boundary string
	if err := db.QueryRowContext(context.Background(), "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	if err := db.QueryRowContext(context.Background(), `SELECT workspace_id,id FROM client WHERE id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' LIMIT 1`).Scan(&workspace, &boundary); err != nil {
		t.Skipf("no UUID client: %v", err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace, PrincipalType: 7, PrincipalID: ""})
	client := NewPostgresClientRepository(postgresCore.NewWorkspaceAwareOperations(db), "client")
	r, err := client.GetClientListPageData(ctx, &clientpb.GetClientListPageDataRequest{Pagination: cursorToken(2, postgresCore.EncodePageCursor(2, "next", boundary))})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ClientList) != 0 || r.Pagination.GetTotalItems() != 0 {
		t.Fatalf("empty STAFF principal returned %d rows / %d total", len(r.ClientList), r.Pagination.GetTotalItems())
	}
	t.Log("empty STAFF principal: boundary excluded, 0 rows")
	var staffID string
	if err := db.QueryRowContext(context.Background(), "SELECT id FROM staff WHERE workspace_id = $1 LIMIT 1", workspace).Scan(&staffID); err != nil {
		t.Skipf("no staff row in selected workspace: %v", err)
	}
	staffCtx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace, PrincipalType: 7, PrincipalID: staffID})
	if _, err := client.GetClientListPageData(staffCtx, &clientpb.GetClientListPageDataRequest{Pagination: cursorOffset(2, 1)}); err != nil {
		t.Fatalf("nonempty STAFF scope query: %v", err)
	}
	t.Log("nonempty STAFF principal: scoped SQL and count executed")
}

func TestClientPageCursor_DerivedSortSearchParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var readOnly, workspace string
	if err := db.QueryRowContext(context.Background(), "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT workspace_id FROM client GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1").Scan(&workspace); err != nil {
		t.Skipf("no clients: %v", err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
	client := NewPostgresClientRepository(postgresCore.NewWorkspaceAwareOperations(db), "client")
	read := func(p *commonpb.PaginationRequest) cursorPageResult {
		r, err := client.GetClientListPageData(ctx, &clientpb.GetClientListPageDataRequest{
			Search:     &commonpb.SearchRequest{Query: "a"},
			Sort:       &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "active_subscriptions", Direction: commonpb.SortDirection_DESC}}},
			Pagination: p,
		})
		if err != nil {
			t.Fatal(err)
		}
		out := cursorPageResult{page: r.Pagination}
		for _, x := range r.ClientList {
			out.ids = append(out.ids, x.Id)
		}
		return out
	}
	first := read(cursorOffset(7, 1))
	if first.page.GetTotalPages() < 2 {
		t.Skipf("search has %d page(s)", first.page.GetTotalPages())
	}
	keyset := read(cursorToken(7, first.page.GetNextCursor()))
	offset := read(cursorOffset(7, 2))
	if !slices.Equal(keyset.ids, offset.ids) {
		t.Fatalf("derived-sort page 2: keyset %v, offset %v", keyset.ids, offset.ids)
	}
	back := read(cursorToken(7, keyset.page.GetPrevCursor()))
	if !slices.Equal(back.ids, first.ids) {
		t.Fatalf("derived-sort previous: got %v, want %v", back.ids, first.ids)
	}
	t.Logf("derived active_subscriptions DESC + search: %d rows, page 2 and Prev exact", first.page.GetTotalItems())
}

func TestLocationAreaPageCursor_EmptyScopedQuery(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var readOnly, workspace string
	if err := db.QueryRowContext(context.Background(), "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT id FROM workspace LIMIT 1").Scan(&workspace); err != nil {
		t.Skipf("no workspace: %v", err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
	repo := NewPostgresLocationAreaRepository(postgresCore.NewWorkspaceAwareOperations(db), "location_area")
	r, err := repo.GetLocationAreaListPageData(ctx, &locationareapb.GetLocationAreaListPageDataRequest{Pagination: cursorOffset(7, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.LocationAreaList) != 0 || r.Pagination.GetTotalItems() != 0 {
		t.Skipf("workspace has %d location areas; this probe targets empty scope", r.Pagination.GetTotalItems())
	}
	t.Log("empty scoped location_area page and count executed; no two-page parity data")
}

func TestWorkspaceUserPageCursor_JoinedSortParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var readOnly, workspace string
	if err := db.QueryRowContext(context.Background(), "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database must be read-only")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT workspace_id FROM workspace_user WHERE active GROUP BY workspace_id ORDER BY COUNT(*) DESC LIMIT 1").Scan(&workspace); err != nil {
		t.Skipf("no workspace users: %v", err)
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: workspace})
	repo := NewPostgresWorkspaceUserRepository(postgresCore.NewWorkspaceAwareOperations(db), "workspace_user")
	read := func(p *commonpb.PaginationRequest) cursorPageResult {
		r, err := repo.GetWorkspaceUserListPageData(ctx, &workspaceuserpb.GetWorkspaceUserListPageDataRequest{
			Sort:       &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "u.first_name", Direction: commonpb.SortDirection_ASC}}},
			Pagination: p,
		})
		if err != nil {
			t.Fatal(err)
		}
		out := cursorPageResult{page: r.Pagination}
		for _, x := range r.WorkspaceUserList {
			out.ids = append(out.ids, x.Id)
		}
		return out
	}
	first := read(cursorOffset(7, 1))
	if first.page.GetTotalPages() < 2 {
		t.Skipf("only %d page(s)", first.page.GetTotalPages())
	}
	if !strings.HasPrefix(first.page.GetNextCursor(), "k1:") {
		t.Skip("first joined-sort boundary is a legacy ID")
	}
	keyset := read(cursorToken(7, first.page.GetNextCursor()))
	offset := read(cursorOffset(7, 2))
	if !slices.Equal(keyset.ids, offset.ids) {
		t.Fatalf("joined sort page 2: keyset %v, offset %v", keyset.ids, offset.ids)
	}
	back := read(cursorToken(7, keyset.page.GetPrevCursor()))
	if !slices.Equal(back.ids, first.ids) {
		t.Fatalf("joined sort Prev: got %v, want %v", back.ids, first.ids)
	}
	t.Logf("u.first_name ASC joined sort: %d rows, page 2 and Prev exact", first.page.GetTotalItems())
}
