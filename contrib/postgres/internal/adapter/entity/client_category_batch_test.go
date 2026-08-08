//go:build postgresql

package entity

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

func TestHydrateClientCategoriesLoadsReturnedPageOnce(t *testing.T) {
	t.Parallel()

	clients := []*clientpb.Client{{Id: "client-2"}, nil, {Id: "client-1"}, {Id: "client-2"}}
	calls := 0
	err := hydrateClientCategories(context.Background(), clients, func(_ context.Context, ids []string) (map[string][]*clientcategorypb.ClientCategory, error) {
		calls++
		if want := []string{"client-2", "client-1"}; !reflect.DeepEqual(ids, want) {
			t.Fatalf("page IDs = %v, want %v", ids, want)
		}
		return map[string][]*clientcategorypb.ClientCategory{
			"client-1": {{Id: "category-link-1", ClientId: "client-1"}},
		}, nil
	})
	if err != nil {
		t.Fatalf("hydrateClientCategories: %v", err)
	}
	if calls != 1 {
		t.Fatalf("batch loader calls = %d, want 1", calls)
	}
	if got := clients[2].GetCategories(); len(got) != 1 || got[0].GetId() != "category-link-1" {
		t.Fatalf("client categories = %v, want category-link-1", got)
	}
}

func TestHydrateClientCategoriesSkipsEmptyPage(t *testing.T) {
	t.Parallel()

	calls := 0
	err := hydrateClientCategories(context.Background(), []*clientpb.Client{nil, {}}, func(context.Context, []string) (map[string][]*clientcategorypb.ClientCategory, error) {
		calls++
		return nil, nil
	})
	if err != nil {
		t.Fatalf("hydrateClientCategories: %v", err)
	}
	if calls != 0 {
		t.Fatalf("batch loader calls = %d, want 0", calls)
	}
}

func TestHydrateClientCategoriesPropagatesBatchError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("category query failed")
	err := hydrateClientCategories(context.Background(), []*clientpb.Client{{Id: "client-1"}}, func(context.Context, []string) (map[string][]*clientcategorypb.ClientCategory, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}
}

func TestClientActiveSubscriptionSortSQLIsConditionalAndSetOriented(t *testing.T) {
	tests := []struct {
		name           string
		sort           *commonpb.SortRequest
		wantJoin       bool
		wantProjection string
	}{
		{name: "default page", wantProjection: "0::bigint AS active_subscriptions"},
		{
			name:           "ordinary sort",
			sort:           &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "name"}}},
			wantProjection: "0::bigint AS active_subscriptions",
		},
		{
			name:           "derived count sort",
			sort:           &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "active_subscriptions"}}},
			wantJoin:       true,
			wantProjection: "COALESCE(sub.active_subscriptions, 0)::bigint AS active_subscriptions",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projection, join := clientActiveSubscriptionSortSQL(tc.sort)
			if projection != tc.wantProjection {
				t.Fatalf("projection = %q, want %q", projection, tc.wantProjection)
			}
			if (join != "") != tc.wantJoin {
				t.Fatalf("join present = %t, want %t: %q", join != "", tc.wantJoin, join)
			}
			if strings.Contains(strings.ToUpper(join), "LATERAL") {
				t.Fatalf("derived sort regressed to correlated LATERAL SQL: %q", join)
			}
			if tc.wantJoin && (!strings.Contains(join, "GROUP BY s.client_id") || !strings.Contains(join, "s.workspace_id = $1")) {
				t.Fatalf("derived sort is not a scoped set aggregate: %q", join)
			}
		})
	}
}
