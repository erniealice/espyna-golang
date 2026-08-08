//go:build postgresql

package subscription

import (
	"context"
	"math"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	planSchwpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	groupMemberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	sgppPlanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	sgppStaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	groupWorkspaceUserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
	"google.golang.org/protobuf/proto"
)

type listCaptureOps struct {
	t          *testing.T
	params     *interfaces.ListParams
	listResult *interfaces.ListResult
}

func (c *listCaptureOps) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	c.t.Fatalf("unexpected Create on listCaptureOps")
	return nil, nil
}

func (c *listCaptureOps) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	c.t.Fatalf("unexpected Read on listCaptureOps")
	return nil, nil
}

func (c *listCaptureOps) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	c.t.Fatalf("unexpected Update on listCaptureOps")
	return nil, nil
}

func (c *listCaptureOps) Delete(ctx context.Context, tableName string, id string) error {
	c.t.Fatalf("unexpected Delete on listCaptureOps")
	return nil
}

func (c *listCaptureOps) HardDelete(ctx context.Context, tableName string, id string) error {
	c.t.Fatalf("unexpected HardDelete on listCaptureOps")
	return nil
}

func (c *listCaptureOps) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	c.params = params
	return c.listResult, nil
}

func (c *listCaptureOps) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	c.t.Fatalf("unexpected Query on listCaptureOps")
	return nil, nil
}

func (c *listCaptureOps) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	c.t.Fatalf("unexpected QueryOne on listCaptureOps")
	return nil, nil
}

func newListCaptureOps(t *testing.T, result *interfaces.ListResult) *listCaptureOps {
	return &listCaptureOps{t: t, listResult: result}
}

func baseListPageRequest() (*commonpb.SearchRequest, *commonpb.FilterRequest, *commonpb.SortRequest, *commonpb.PaginationRequest) {
	search := &commonpb.SearchRequest{Query: "term-1"}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{Field: "name"}}}
	sort := &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "name"}}}
	pagination := &commonpb.PaginationRequest{
		Limit:  25,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: 2}},
	}
	return search, filters, sort, pagination
}

func baseListResult() *interfaces.ListResult {
	return &interfaces.ListResult{
		Data: []map[string]any{
			{"id": "row-1"},
			{"id": "row-2"},
		},
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  99,
			CurrentPage: int32ptr(3),
			TotalPages:  int32ptr(10),
			HasNext:     true,
		},
	}
}

func int32ptr(v int32) *int32 {
	return &v
}

func TestGetSubscriptionListPageData_ForwardsSearchFiltersSortAndPaginationParams(t *testing.T) {
	search, filters, sort, pagination := baseListPageRequest()

	type testCase struct {
		name       string
		call       func(context.Context, *listCaptureOps) error
		wantParams *interfaces.ListParams
	}

	tests := []testCase{
		{
			name: "subscription_group",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupRepository(ops, "subscription_group")
				_, err := repo.GetSubscriptionGroupListPageData(ctx, &sgpppb.GetSubscriptionGroupListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
		{
			name: "subscription_group_product_plan",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupProductPlanRepository(ops, "subscription_group_product_plan")
				_, err := repo.GetSubscriptionGroupProductPlanListPageData(ctx, &sgppPlanpb.GetSubscriptionGroupProductPlanListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
		{
			name: "subscription_group_member",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupMemberRepository(ops, "subscription_group_member")
				_, err := repo.GetSubscriptionGroupMemberListPageData(ctx, &groupMemberpb.GetSubscriptionGroupMemberListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
		{
			name: "subscription_group_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupWorkspaceUserRepository(ops, "subscription_group_workspace_user")
				_, err := repo.GetSubscriptionGroupWorkspaceUserListPageData(ctx, &groupWorkspaceUserpb.GetSubscriptionGroupWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
		{
			name: "subscription_group_product_plan_staff",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupProductPlanStaffRepository(ops, "subscription_group_product_plan_staff")
				_, err := repo.GetSubscriptionGroupProductPlanStaffListPageData(ctx, &sgppStaffpb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
		{
			name: "price_schedule_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresPriceScheduleWorkspaceUserRepository(ops, "price_schedule_workspace_user")
				_, err := repo.GetPriceScheduleWorkspaceUserListPageData(ctx, &planSchwpb.GetPriceScheduleWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
			wantParams: &interfaces.ListParams{
				Search:     search,
				Filters:    filters,
				Sort:       sort,
				Pagination: pagination,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := newListCaptureOps(t, baseListResult())
			if err := tc.call(context.Background(), ops); err != nil {
				t.Fatalf("call failed: %v", err)
			}
			if !proto.Equal(ops.params.Search, tc.wantParams.Search) {
				t.Fatalf("search params not forwarded exactly")
			}
			if !proto.Equal(ops.params.Filters, tc.wantParams.Filters) {
				t.Fatalf("filters params not forwarded exactly")
			}
			if !proto.Equal(ops.params.Sort, tc.wantParams.Sort) {
				t.Fatalf("sort params not forwarded exactly")
			}
			if !proto.Equal(ops.params.Pagination, tc.wantParams.Pagination) {
				t.Fatalf("pagination params not forwarded exactly")
			}
		})
	}
}

func TestGetSubscriptionListPageData_UsesDatabasePagination(t *testing.T) {
	search, filters, sort, pagination := baseListPageRequest()
	wantPagination := baseListResult().Pagination

	type testCase struct {
		name string
		call func(context.Context, *listCaptureOps) (*commonpb.PaginationResponse, error)
	}

	tests := []testCase{
		{
			name: "subscription_group",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresSubscriptionGroupRepository(ops, "subscription_group")
				resp, err := repo.GetSubscriptionGroupListPageData(ctx, &sgpppb.GetSubscriptionGroupListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
		{
			name: "subscription_group_product_plan",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresSubscriptionGroupProductPlanRepository(ops, "subscription_group_product_plan")
				resp, err := repo.GetSubscriptionGroupProductPlanListPageData(ctx, &sgppPlanpb.GetSubscriptionGroupProductPlanListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
		{
			name: "subscription_group_member",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresSubscriptionGroupMemberRepository(ops, "subscription_group_member")
				resp, err := repo.GetSubscriptionGroupMemberListPageData(ctx, &groupMemberpb.GetSubscriptionGroupMemberListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
		{
			name: "subscription_group_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresSubscriptionGroupWorkspaceUserRepository(ops, "subscription_group_workspace_user")
				resp, err := repo.GetSubscriptionGroupWorkspaceUserListPageData(ctx, &groupWorkspaceUserpb.GetSubscriptionGroupWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
		{
			name: "subscription_group_product_plan_staff",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresSubscriptionGroupProductPlanStaffRepository(ops, "subscription_group_product_plan_staff")
				resp, err := repo.GetSubscriptionGroupProductPlanStaffListPageData(ctx, &sgppStaffpb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
		{
			name: "price_schedule_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) (*commonpb.PaginationResponse, error) {
				repo := NewPostgresPriceScheduleWorkspaceUserRepository(ops, "price_schedule_workspace_user")
				resp, err := repo.GetPriceScheduleWorkspaceUserListPageData(ctx, &planSchwpb.GetPriceScheduleWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				if resp != nil {
					return resp.Pagination, err
				}
				return nil, err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := newListCaptureOps(t, baseListResult())
			got, err := tc.call(context.Background(), ops)
			if err != nil {
				t.Fatalf("call failed: %v", err)
			}
			if got == nil {
				t.Fatalf("nil pagination response")
			}
			if !proto.Equal(got, wantPagination) {
				t.Fatalf("pagination response = %#v, want %#v", got, wantPagination)
			}
		})
	}
}

func TestGetSubscriptionListPageData_ConversionErrorsBubble(t *testing.T) {
	search, filters, sort, pagination := baseListPageRequest()
	failedResult := &interfaces.ListResult{
		Data: []map[string]any{
			{"bad": math.Inf(1)},
		},
		Pagination: baseListResult().Pagination,
	}

	type testCase struct {
		name string
		call func(context.Context, *listCaptureOps) error
	}

	tests := []testCase{
		{
			name: "subscription_group",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupRepository(ops, "subscription_group")
				_, err := repo.GetSubscriptionGroupListPageData(ctx, &sgpppb.GetSubscriptionGroupListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
		{
			name: "subscription_group_product_plan",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupProductPlanRepository(ops, "subscription_group_product_plan")
				_, err := repo.GetSubscriptionGroupProductPlanListPageData(ctx, &sgppPlanpb.GetSubscriptionGroupProductPlanListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
		{
			name: "subscription_group_member",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupMemberRepository(ops, "subscription_group_member")
				_, err := repo.GetSubscriptionGroupMemberListPageData(ctx, &groupMemberpb.GetSubscriptionGroupMemberListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
		{
			name: "subscription_group_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupWorkspaceUserRepository(ops, "subscription_group_workspace_user")
				_, err := repo.GetSubscriptionGroupWorkspaceUserListPageData(ctx, &groupWorkspaceUserpb.GetSubscriptionGroupWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
		{
			name: "subscription_group_product_plan_staff",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresSubscriptionGroupProductPlanStaffRepository(ops, "subscription_group_product_plan_staff")
				_, err := repo.GetSubscriptionGroupProductPlanStaffListPageData(ctx, &sgppStaffpb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
		{
			name: "price_schedule_workspace_user",
			call: func(ctx context.Context, ops *listCaptureOps) error {
				repo := NewPostgresPriceScheduleWorkspaceUserRepository(ops, "price_schedule_workspace_user")
				_, err := repo.GetPriceScheduleWorkspaceUserListPageData(ctx, &planSchwpb.GetPriceScheduleWorkspaceUserListPageDataRequest{
					Search:     search,
					Filters:    filters,
					Sort:       sort,
					Pagination: pagination,
				})
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := newListCaptureOps(t, failedResult)
			if err := tc.call(context.Background(), ops); err == nil {
				t.Fatalf("expected conversion failure from list row, got nil")
			}
		})
	}
}
