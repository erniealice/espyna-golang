package rating_description_set_product_plan

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// codex-review-impl3.out.md findings #1 and #3. These tests prove the
// assignment LIST page loads under copya.md's locked grant matrix (Q24) —
// Education Admin (all 12) and AY Coordinator (link list/read/create/
// update/delete) both hold rating_description_set_product_plan:list — with
// a SINGLE authorization check, and that offering names are sourced through
// the workspace-scoped port (never the generic, unscoped ProductPlan.
// ListProductPlans).

type listSummaryLinkFakeRepo struct {
	pb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer
	links []*pb.RatingDescriptionSetProductPlan
	err   error
}

func (r *listSummaryLinkFakeRepo) GetRatingDescriptionSetProductPlanListPageData(_ context.Context, req *pb.GetRatingDescriptionSetProductPlanListPageDataRequest) (*pb.GetRatingDescriptionSetProductPlanListPageDataResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	var matched []*pb.RatingDescriptionSetProductPlan
	for _, l := range r.links {
		if l.GetPriceScheduleId() == req.GetPriceScheduleId() {
			matched = append(matched, l)
		}
	}
	return &pb.GetRatingDescriptionSetProductPlanListPageDataResponse{RatingDescriptionSetProductPlanList: matched, Success: true}, nil
}

type listSummaryScheduleFakeRepo struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	schedules []*priceschedulepb.PriceSchedule
	err       error
}

func (r *listSummaryScheduleFakeRepo) ListPriceSchedules(_ context.Context, _ *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &priceschedulepb.ListPriceSchedulesResponse{Data: r.schedules, Success: true}, nil
}

// listSummaryProductPlanFakeRepo implements BOTH the generic (unimplemented)
// interface AND ListWorkspaceScopedProductPlans, so the use case's
// constructor type-assertion against domainports.WorkspaceScopedProductPlanReader
// succeeds — proving the use case calls the SCOPED method, never the
// generic ListProductPlans (which panics/errors here since it is NOT
// implemented on this fake).
type listSummaryProductPlanFakeRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	items []*productplanpb.ProductPlan
	err   error
	calls int
}

func (r *listSummaryProductPlanFakeRepo) ListWorkspaceScopedProductPlans(_ context.Context) ([]*productplanpb.ProductPlan, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.items, nil
}

type listSummarySetFakeRepo struct {
	setpb.UnimplementedRatingDescriptionSetDomainServiceServer
	sets []*setpb.RatingDescriptionSet
	err  error
}

func (r *listSummarySetFakeRepo) ListRatingDescriptionSets(_ context.Context, _ *setpb.ListRatingDescriptionSetsRequest) (*setpb.ListRatingDescriptionSetsResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &setpb.ListRatingDescriptionSetsResponse{Data: r.sets, Success: true}, nil
}

var linkListSummaryPermission = entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionList)

func linkListSummaryAuthorizer(perm string, allow bool) *relinkTestAuthorizer {
	return &relinkTestAuthorizer{enabled: true, perms: map[string]bool{perm: allow}}
}

func linkListSummaryServices(authz *relinkTestAuthorizer) GetRatingDescriptionSetProductPlanListSummaryPageDataServices {
	return GetRatingDescriptionSetProductPlanListSummaryPageDataServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

// TestGetRatingDescriptionSetProductPlanListSummaryPageData_PermissionMatrix
// exercises Education Admin and AY Coordinator — both hold
// rating_description_set_product_plan:list per copya.md's locked grant
// matrix — and proves offering names come from the workspace-scoped port.
func TestGetRatingDescriptionSetProductPlanListSummaryPageData_PermissionMatrix(t *testing.T) {
	roles := []string{
		"Education Admin (all 12, floor = list)",
		"AY Coordinator (link list/read/create/update/delete)",
	}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			authz := linkListSummaryAuthorizer(linkListSummaryPermission, true)
			linkRepo := &listSummaryLinkFakeRepo{links: []*pb.RatingDescriptionSetProductPlan{
				{Id: "link-1", ProductPlanId: "pp-1", PriceScheduleId: "ay-1", RatingDescriptionSetId: "set-1", Active: true},
			}}
			scheduleRepo := &listSummaryScheduleFakeRepo{schedules: []*priceschedulepb.PriceSchedule{
				{Id: "ay-1", Name: "AY 2026-27", Active: true},
			}}
			ppRepo := &listSummaryProductPlanFakeRepo{items: []*productplanpb.ProductPlan{{Id: "pp-1", Name: "Design", Active: true}}}
			setRepo := &listSummarySetFakeRepo{sets: []*setpb.RatingDescriptionSet{{Id: "set-1", Name: "Design Y1", Version: 1}}}

			uc := NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
				GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories{
					RatingDescriptionSetProductPlan: linkRepo,
					PriceSchedule:                   scheduleRepo,
					ProductPlan:                     ppRepo,
					RatingDescriptionSet:            setRepo,
				},
				linkListSummaryServices(authz),
			)

			resp, err := uc.ExecuteForScheduleID(newRelinkContext(), "ay-1")
			if err != nil {
				t.Fatalf("expected success for %s (granted only %q), got %v", role, linkListSummaryPermission, err)
			}
			if resp.SelectedScheduleId != "ay-1" {
				t.Fatalf("expected selected schedule ay-1, got %q", resp.SelectedScheduleId)
			}
			if len(resp.Links) != 1 {
				t.Fatalf("expected 1 link row, got %d", len(resp.Links))
			}
			row := resp.Links[0]
			if row.ProductPlanName != "Design" {
				t.Errorf("expected offering name %q, got %q", "Design", row.ProductPlanName)
			}
			if row.Set.Name != "Design Y1" || row.Set.Version != 1 {
				t.Errorf("expected set info {Design Y1, v1}, got %+v", row.Set)
			}
			if ppRepo.calls != 1 {
				t.Fatalf("expected the workspace-scoped port to be called exactly once, got %d", ppRepo.calls)
			}
			if len(authz.asked) != 1 || authz.asked[0] != linkListSummaryPermission {
				t.Fatalf("expected exactly one authorization check for %q, got %v", linkListSummaryPermission, authz.asked)
			}
		})
	}
}

func TestGetRatingDescriptionSetProductPlanListSummaryPageData_DeniedWithoutListPermission(t *testing.T) {
	authz := linkListSummaryAuthorizer(linkListSummaryPermission, false)
	linkRepo := &listSummaryLinkFakeRepo{}
	uc := NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
		GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories{RatingDescriptionSetProductPlan: linkRepo},
		linkListSummaryServices(authz),
	)
	if _, err := uc.ExecuteForScheduleID(newRelinkContext(), "ay-1"); err == nil {
		t.Fatal("expected denial without rating_description_set_product_plan:list")
	}
}

// TestGetRatingDescriptionSetProductPlanListSummaryPageData_MissingOfferingNameDegrades
// proves a missing/failing workspace-scoped product_plan port degrades to a
// blank offering name (the fayna view falls back to the raw product_plan_id)
// instead of failing the whole page — an optional label, not core content.
func TestGetRatingDescriptionSetProductPlanListSummaryPageData_MissingOfferingNameDegrades(t *testing.T) {
	authz := linkListSummaryAuthorizer(linkListSummaryPermission, true)
	linkRepo := &listSummaryLinkFakeRepo{links: []*pb.RatingDescriptionSetProductPlan{
		{Id: "link-1", ProductPlanId: "pp-1", PriceScheduleId: "ay-1", RatingDescriptionSetId: "set-1", Active: true},
	}}
	scheduleRepo := &listSummaryScheduleFakeRepo{schedules: []*priceschedulepb.PriceSchedule{{Id: "ay-1", Name: "AY 2026-27", Active: true}}}
	ppRepo := &listSummaryProductPlanFakeRepo{err: errors.New("simulated failure")}

	uc := NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
		GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories{
			RatingDescriptionSetProductPlan: linkRepo,
			PriceSchedule:                   scheduleRepo,
			ProductPlan:                     ppRepo,
		},
		linkListSummaryServices(authz),
	)
	resp, err := uc.ExecuteForScheduleID(newRelinkContext(), "ay-1")
	if err != nil {
		t.Fatalf("expected the page to still load when the offering name lookup fails, got %v", err)
	}
	if len(resp.Links) != 1 || resp.Links[0].ProductPlanName != "" {
		t.Fatalf("expected 1 link row with a blank offering name, got %+v", resp.Links)
	}
}

// TestGetRatingDescriptionSetProductPlanListSummaryPageData_ProductPlanNotWorkspaceScopedReaderDegrades
// proves a concrete ProductPlan repo that does NOT implement the scoped
// port (e.g. a hypothetical non-Postgres adapter) degrades to blank
// offering names — it must never fall back to an unscoped generic list.
type genericOnlyProductPlanFakeRepo struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
}

func TestGetRatingDescriptionSetProductPlanListSummaryPageData_ProductPlanNotWorkspaceScopedReaderDegrades(t *testing.T) {
	authz := linkListSummaryAuthorizer(linkListSummaryPermission, true)
	linkRepo := &listSummaryLinkFakeRepo{links: []*pb.RatingDescriptionSetProductPlan{
		{Id: "link-1", ProductPlanId: "pp-1", PriceScheduleId: "ay-1", RatingDescriptionSetId: "set-1", Active: true},
	}}
	uc := NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
		GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories{
			RatingDescriptionSetProductPlan: linkRepo,
			ProductPlan:                     &genericOnlyProductPlanFakeRepo{},
		},
		linkListSummaryServices(authz),
	)
	resp, err := uc.ExecuteForScheduleID(newRelinkContext(), "ay-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(resp.Links) != 1 || resp.Links[0].ProductPlanName != "" {
		t.Fatalf("expected a blank offering name when the port is unsupported, got %+v", resp.Links)
	}
}
