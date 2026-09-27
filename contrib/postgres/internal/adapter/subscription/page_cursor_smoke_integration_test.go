//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/proto"
)

// TestScopedPageAdapterSmoke_ReadOnly executes each selected adapter's real
// offset SQL against an explicitly supplied local, read-only database lane.
// It catches SQL placeholder, projection and scan drift that compilation misses.
func TestScopedPageAdapterSmoke_ReadOnly(t *testing.T) {
	dsn, ws := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_WORKSPACE_ID")
	if dsn == "" || ws == "" {
		t.Skip("TEST_DATABASE_URL and TEST_WORKSPACE_ID required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: ws})
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" {
		t.Fatal("database session is not read-only")
	}
	cases := []struct{ entity, method string }{
		{"balance", "GetBalanceListPageData"},
		{"balance_attribute", "GetBalanceAttributeListPageData"},
		{"billing_event", "GetBillingEventListPageData"},
		{"invoice", "GetInvoiceListPageData"},
		{"invoice_attribute", "GetInvoiceAttributeListPageData"},
		{"license", "GetLicenseListPageData"},
		{"license_history", "GetLicenseHistoryListPageData"},
		{"plan_attribute", "GetPlanAttributeListPageData"},
		{"price_plan", "GetPricePlanListPageData"},
		{"price_schedule", "GetPriceScheduleListPageData"},
		{"product_price_plan", "GetProductPricePlanListPageData"},
		{"subscription", "GetSubscriptionListPageData"},
		{"subscription_attribute", "GetSubscriptionAttributeListPageData"},
		{"subscription_seat", "GetSubscriptionSeatListPageData"},
		{"subscription_workspace_user", "GetSubscriptionWorkspaceUserListPageData"},
	}
	for _, tc := range cases {
		t.Run(tc.entity, func(t *testing.T) {
			repo, err := registry.CreateRepository("postgresql", tc.entity, db, tc.entity)
			if err != nil {
				t.Fatal(err)
			}
			fn := reflect.ValueOf(repo).MethodByName(tc.method)
			if !fn.IsValid() {
				t.Fatalf("method %s missing", tc.method)
			}
			call := func(p *commonpb.PaginationRequest) proto.Message {
				t.Helper()
				req := reflect.New(fn.Type().In(1).Elem())
				field := req.Elem().FieldByName("Pagination")
				if !field.IsValid() || !field.CanSet() {
					t.Fatal("request has no writable pagination field")
				}
				field.Set(reflect.ValueOf(p))
				result := fn.Call([]reflect.Value{reflect.ValueOf(ctx), req})
				if e := result[1].Interface(); e != nil {
					t.Fatal(e)
				}
				return result[0].Interface().(proto.Message)
			}
			offset := func(n int32) *commonpb.PaginationRequest {
				return &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: n}}}
			}
			cursor := func(token string) *commonpb.PaginationRequest {
				return &commonpb.PaginationRequest{Limit: 2, Method: &commonpb.PaginationRequest_Cursor{Cursor: &commonpb.CursorPagination{Token: token}}}
			}
			first, second := call(offset(1)), call(offset(2))
			missing := call(cursor("k1:2:next:00000000-0000-0000-0000-000000000099"))
			if !proto.Equal(second, missing) {
				t.Fatal("missing boundary did not match offset page 2")
			}
			pagination := reflect.ValueOf(first).MethodByName("GetPagination").Call(nil)[0]
			next := pagination.MethodByName("GetNextCursor").Call(nil)[0].String()
			if next == "" {
				t.Log("offset and missing-boundary fallback match; valid cursor unproven on this fixture")
				return
			}
			if actual := call(cursor(next)); !proto.Equal(second, actual) {
				t.Fatal("valid Next cursor did not match offset page 2")
			}
			t.Log("offset page 2, missing-boundary fallback and valid Next cursor match")
		})
	}
}
