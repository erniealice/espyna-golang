package rating_description_set

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	scalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// codex-review-impl3.out.md finding #1 ("the management lists still fail
// under the prescribed permissions"). These tests prove the LIST page loads
// under exactly copya.md's locked grant matrix (Q24) — Education Admin
// (all 12), AY Coordinator (set list/read only), and Section Template
// Manager (set list/read only, the narrowest role) — with a SINGLE
// authorization check against rating_description_set:list, never the
// sibling entities' own score_scale:list / rating_description_set_
// product_plan:list gates that Section Template Manager (and, for
// score_scale:list, NO role in the clone's catalog) does not hold.

type listSummaryFakeSetRepo struct {
	pb.UnimplementedRatingDescriptionSetDomainServiceServer
	sets []*pb.RatingDescriptionSet
	err  error
}

func (r *listSummaryFakeSetRepo) ListRatingDescriptionSets(_ context.Context, _ *pb.ListRatingDescriptionSetsRequest) (*pb.ListRatingDescriptionSetsResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &pb.ListRatingDescriptionSetsResponse{Data: r.sets, Success: true}, nil
}

type listSummaryFakeScoreScaleRepo struct {
	scalepb.UnimplementedScoreScaleDomainServiceServer
	scales []*scalepb.ScoreScale
	err    error
	calls  int
}

func (r *listSummaryFakeScoreScaleRepo) ListScoreScales(_ context.Context, _ *scalepb.ListScoreScalesRequest) (*scalepb.ListScoreScalesResponse, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return &scalepb.ListScoreScalesResponse{Data: r.scales, Success: true}, nil
}

// listSummaryFakeEntryRepo implements BOTH the generic
// RatingDescriptionSetEntryDomainServiceServer (so it satisfies the
// Repositories field's static type) AND domainports.
// RatingDescriptionSetEntryCounter (codex-review-impl4.out.md round-3/
// round-4 disposition #2 — the list summary use case now aggregates counts
// through the scoped counter port, never a workspace-wide List loop).
// countCalls tracks CountActiveRatingDescriptionSetEntriesBySet invocations;
// legacy listCalls (ListRatingDescriptionSetEntries) must stay 0 once the
// counter path is wired, proving the capped scan is truly gone.
type listSummaryFakeEntryRepo struct {
	entrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
	entries    []*entrypb.RatingDescriptionSetEntry
	err        error
	listCalls  int
	countCalls int
	lastIDs    []string
}

func (r *listSummaryFakeEntryRepo) ListRatingDescriptionSetEntries(_ context.Context, _ *entrypb.ListRatingDescriptionSetEntriesRequest) (*entrypb.ListRatingDescriptionSetEntriesResponse, error) {
	r.listCalls++
	if r.err != nil {
		return nil, r.err
	}
	return &entrypb.ListRatingDescriptionSetEntriesResponse{Data: r.entries, Success: true}, nil
}

func (r *listSummaryFakeEntryRepo) CountActiveRatingDescriptionSetEntriesBySet(_ context.Context, ratingDescriptionSetIds []string) (map[string]int32, error) {
	r.countCalls++
	r.lastIDs = ratingDescriptionSetIds
	if r.err != nil {
		return nil, r.err
	}
	wanted := map[string]bool{}
	for _, id := range ratingDescriptionSetIds {
		wanted[id] = true
	}
	counts := map[string]int32{}
	for _, e := range r.entries {
		if e == nil || !e.GetActive() || !wanted[e.GetRatingDescriptionSetId()] {
			continue
		}
		counts[e.GetRatingDescriptionSetId()]++
	}
	return counts, nil
}

// listSummaryFakeEntryRepoListOnly implements ONLY the generic list RPC
// (no counter port) — proves a non-conforming adapter degrades the Entries
// column to 0 instead of falling back to the removed capped scan.
type listSummaryFakeEntryRepoListOnly struct {
	entrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
}

// listSummaryFakeLinkRepo mirrors listSummaryFakeEntryRepo for the Links
// column / RatingDescriptionSetProductPlanCounter.
type listSummaryFakeLinkRepo struct {
	linkpb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer
	links      []*linkpb.RatingDescriptionSetProductPlan
	err        error
	listCalls  int
	countCalls int
}

func (r *listSummaryFakeLinkRepo) ListRatingDescriptionSetProductPlans(_ context.Context, _ *linkpb.ListRatingDescriptionSetProductPlansRequest) (*linkpb.ListRatingDescriptionSetProductPlansResponse, error) {
	r.listCalls++
	if r.err != nil {
		return nil, r.err
	}
	return &linkpb.ListRatingDescriptionSetProductPlansResponse{Data: r.links, Success: true}, nil
}

func (r *listSummaryFakeLinkRepo) CountActiveRatingDescriptionSetProductPlansBySet(_ context.Context, ratingDescriptionSetIds []string) (map[string]int32, error) {
	r.countCalls++
	if r.err != nil {
		return nil, r.err
	}
	wanted := map[string]bool{}
	for _, id := range ratingDescriptionSetIds {
		wanted[id] = true
	}
	counts := map[string]int32{}
	for _, l := range r.links {
		if l == nil || !l.GetActive() || !wanted[l.GetRatingDescriptionSetId()] {
			continue
		}
		counts[l.GetRatingDescriptionSetId()]++
	}
	return counts, nil
}

var listSummaryPermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionList)

func listSummaryServices(authz *lifecycleTestAuthorizer) GetRatingDescriptionSetListSummaryPageDataServices {
	return GetRatingDescriptionSetListSummaryPageDataServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

// TestGetRatingDescriptionSetListSummaryPageData_PermissionMatrix exercises
// the three prescribed roles from copya.md's locked grant matrix. All three
// hold rating_description_set:list; none of them needs any other grant for
// the page to load fully enriched.
func TestGetRatingDescriptionSetListSummaryPageData_PermissionMatrix(t *testing.T) {
	roles := []string{
		"Education Admin (all 12, floor = list)",
		"AY Coordinator (set list/read only)",
		"Section Template Manager (set list/read only)",
	}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			authz := allowAuthorizer(listSummaryPermission)
			setRepo := &listSummaryFakeSetRepo{sets: []*pb.RatingDescriptionSet{{Id: "set-1", Name: "Set One", ScoreScaleId: "scale-1"}}}
			scaleRepo := &listSummaryFakeScoreScaleRepo{scales: []*scalepb.ScoreScale{{Id: "scale-1", Name: "Scale One"}}}
			entryRepo := &listSummaryFakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1", RatingDescriptionSetId: "set-1", Active: true}}}
			linkRepo := &listSummaryFakeLinkRepo{links: []*linkpb.RatingDescriptionSetProductPlan{{Id: "link-1", RatingDescriptionSetId: "set-1", Active: true}}}

			uc := NewGetRatingDescriptionSetListSummaryPageDataUseCase(
				GetRatingDescriptionSetListSummaryPageDataRepositories{
					RatingDescriptionSet:            setRepo,
					ScoreScale:                      scaleRepo,
					RatingDescriptionSetEntry:       entryRepo,
					RatingDescriptionSetProductPlan: linkRepo,
				},
				listSummaryServices(authz),
			)

			resp, err := uc.Execute(newLifecycleContext(), nil)
			if err != nil {
				t.Fatalf("expected success for %s (granted only %q), got %v", role, listSummaryPermission, err)
			}
			if len(resp.Rows) != 1 {
				t.Fatalf("expected 1 row, got %d", len(resp.Rows))
			}
			row := resp.Rows[0]
			if row.ScoreScaleName != "Scale One" {
				t.Errorf("expected scale name %q, got %q", "Scale One", row.ScoreScaleName)
			}
			if row.EntryCount != 1 {
				t.Errorf("expected entry count 1, got %d", row.EntryCount)
			}
			if row.LinkCount != 1 {
				t.Errorf("expected link count 1, got %d", row.LinkCount)
			}
			// Exactly ONE authorization check — the sibling entities' own
			// gates (score_scale:list, rating_description_set_product_plan:
			// list) were never consulted, proving they are bypassed, not
			// merely satisfied by coincidence.
			if len(authz.asked) != 1 || authz.asked[0] != listSummaryPermission {
				t.Fatalf("expected exactly one authorization check for %q, got %v", listSummaryPermission, authz.asked)
			}
			// codex-review-impl4.out.md round-3/round-4 disposition #2: counts
			// come from the scoped aggregation port, never the workspace-wide
			// List RPC (the former 5,000-row-capped path).
			if entryRepo.countCalls != 1 || entryRepo.listCalls != 0 {
				t.Fatalf("expected exactly one CountActive...BySet call and zero List calls on the entry repo, got count=%d list=%d", entryRepo.countCalls, entryRepo.listCalls)
			}
			if linkRepo.countCalls != 1 || linkRepo.listCalls != 0 {
				t.Fatalf("expected exactly one CountActive...BySet call and zero List calls on the link repo, got count=%d list=%d", linkRepo.countCalls, linkRepo.listCalls)
			}
			if len(entryRepo.lastIDs) != 1 || entryRepo.lastIDs[0] != "set-1" {
				t.Fatalf("expected the counter to be scoped to exactly [set-1], got %v", entryRepo.lastIDs)
			}
		})
	}
}

// TestGetRatingDescriptionSetListSummaryPageData_CounterPortNotImplementedDegrades
// proves a repository that only implements the generic list RPC (not the
// scoped counter port) degrades that column to 0 instead of falling back to
// the removed, workspace-wide, 5,000-row-capped scan — same nil-safe-to-
// empty convention as WorkspaceScopedProductPlanReader.
func TestGetRatingDescriptionSetListSummaryPageData_CounterPortNotImplementedDegrades(t *testing.T) {
	authz := allowAuthorizer(listSummaryPermission)
	setRepo := &listSummaryFakeSetRepo{sets: []*pb.RatingDescriptionSet{{Id: "set-1", Name: "Set One"}}}
	entryRepo := &listSummaryFakeEntryRepoListOnly{}

	uc := NewGetRatingDescriptionSetListSummaryPageDataUseCase(
		GetRatingDescriptionSetListSummaryPageDataRepositories{RatingDescriptionSet: setRepo, RatingDescriptionSetEntry: entryRepo},
		listSummaryServices(authz),
	)
	resp, err := uc.Execute(newLifecycleContext(), nil)
	if err != nil {
		t.Fatalf("expected the page to still load when the entry repo does not implement the counter port, got %v", err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0].EntryCount != 0 {
		t.Fatalf("expected 1 row with EntryCount 0, got %+v", resp.Rows)
	}
}

func TestGetRatingDescriptionSetListSummaryPageData_DeniedWithoutListPermission(t *testing.T) {
	authz := denyAuthorizer(listSummaryPermission)
	setRepo := &listSummaryFakeSetRepo{}
	uc := NewGetRatingDescriptionSetListSummaryPageDataUseCase(
		GetRatingDescriptionSetListSummaryPageDataRepositories{RatingDescriptionSet: setRepo},
		listSummaryServices(authz),
	)
	if _, err := uc.Execute(newLifecycleContext(), nil); err == nil {
		t.Fatal("expected denial without rating_description_set:list")
	}
}

// TestGetRatingDescriptionSetListSummaryPageData_MissingScaleNameDegrades
// proves a failing/absent score_scale lookup degrades to a blank Scale
// name for every row instead of failing the whole page (brief: "a missing
// optional label must not fail the page" — this is exactly the live
// regression f3-verify found and patched in fayna as a stopgap; this test
// proves the espyna-side replacement never reintroduces it).
func TestGetRatingDescriptionSetListSummaryPageData_MissingScaleNameDegrades(t *testing.T) {
	authz := allowAuthorizer(listSummaryPermission)
	setRepo := &listSummaryFakeSetRepo{sets: []*pb.RatingDescriptionSet{{Id: "set-1", Name: "Set One", ScoreScaleId: "scale-1"}}}
	scaleRepo := &listSummaryFakeScoreScaleRepo{err: errors.New("simulated: no score_scale:list grant")}

	uc := NewGetRatingDescriptionSetListSummaryPageDataUseCase(
		GetRatingDescriptionSetListSummaryPageDataRepositories{RatingDescriptionSet: setRepo, ScoreScale: scaleRepo},
		listSummaryServices(authz),
	)
	resp, err := uc.Execute(newLifecycleContext(), nil)
	if err != nil {
		t.Fatalf("expected the page to still load when the scale name lookup fails, got %v", err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0].ScoreScaleName != "" {
		t.Fatalf("expected 1 row with a blank scale name, got %+v", resp.Rows)
	}
}

// TestGetRatingDescriptionSetListSummaryPageData_EntryCountFailurePropagates
// proves a genuine entry-count read failure (core list content, not an
// optional label) still fails the page — unlike the scale name.
func TestGetRatingDescriptionSetListSummaryPageData_EntryCountFailurePropagates(t *testing.T) {
	authz := allowAuthorizer(listSummaryPermission)
	setRepo := &listSummaryFakeSetRepo{sets: []*pb.RatingDescriptionSet{{Id: "set-1", Name: "Set One"}}}
	entryRepo := &listSummaryFakeEntryRepo{err: errors.New("simulated db failure")}

	uc := NewGetRatingDescriptionSetListSummaryPageDataUseCase(
		GetRatingDescriptionSetListSummaryPageDataRepositories{RatingDescriptionSet: setRepo, RatingDescriptionSetEntry: entryRepo},
		listSummaryServices(authz),
	)
	if _, err := uc.Execute(newLifecycleContext(), nil); err == nil {
		t.Fatal("expected the page to fail when the entry-count read genuinely fails")
	}
}
