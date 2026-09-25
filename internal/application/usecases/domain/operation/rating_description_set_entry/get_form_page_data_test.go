package rating_description_set_entry

import (
	"context"
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	scalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// codex-review-impl3.out.md round-2 disposition #11 ("the update-only editor
// still cannot load the drawer"). These tests prove the two picker sources
// are now split by AUTHORIZED ACTION, not shared under one :read gate:
//   - GetRatingDescriptionSetEntryFormPageData (detail page's READ-ONLY
//     matrix) stays gated on rating_description_set:read.
//   - GetRatingDescriptionSetEntryDrawerFormPageData (Add/Edit entry
//     DRAWER) is gated on rating_description_set:update — an update-only
//     editor can open it; a read-only-only viewer (Section Template
//     Manager: set list/read, nothing else, per copya.md's locked grant
//     matrix) can no longer reach it.
//
// Per copya.md's locked grant matrix (Q24): Education Admin holds all 12
// (both read and update); AY Coordinator holds link list/read/create/
// update/delete + SET list/read only (no set update); Section Template
// Manager holds set list/read only. So only Education Admin/Superadmin can
// open the drawer — AY Coordinator and Section Template Manager both see
// the read-only matrix but not the mutating drawer.

var (
	entrySetReadPermission   = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionRead)
	entrySetUpdatePermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionUpdate)
)

func entryFormPickerServices(authz *entryAuthzTestAuthorizer) GetRatingDescriptionSetEntryFormPageDataServices {
	return GetRatingDescriptionSetEntryFormPageDataServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func TestGetRatingDescriptionSetEntryFormPageData_MatrixSource_AuthorizesOnRead(t *testing.T) {
	authz := allowOnlyEntryPermission(entrySetReadPermission)
	uc := NewGetRatingDescriptionSetEntryFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz))
	if _, err := uc.Execute(newEntryAuthzContext(), nil); err != nil {
		t.Fatalf("expected success for the read-only matrix source when only %q is granted, got %v", entrySetReadPermission, err)
	}
}

func TestGetRatingDescriptionSetEntryFormPageData_MatrixSource_UpdateOnlyDenied(t *testing.T) {
	// An update-only grant (no separate :read) must NOT authorize the
	// matrix source — it stays scoped to :read specifically.
	authz := allowOnlyEntryPermission(entrySetUpdatePermission)
	uc := NewGetRatingDescriptionSetEntryFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz))
	if _, err := uc.Execute(newEntryAuthzContext(), nil); err == nil {
		t.Fatal("expected denial: rating_description_set:update alone must not authorize the read-only matrix source")
	}
}

func TestGetRatingDescriptionSetEntryDrawerFormPageData_AuthorizesOnUpdate(t *testing.T) {
	// The update-only editor case the round-2 finding named: granting ONLY
	// :update (no separate :read) must still authorize the drawer.
	authz := allowOnlyEntryPermission(entrySetUpdatePermission)
	uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz))
	if _, err := uc.Execute(newEntryAuthzContext(), nil); err != nil {
		t.Fatalf("expected success for the drawer when only %q is granted, got %v", entrySetUpdatePermission, err)
	}
}

func TestGetRatingDescriptionSetEntryDrawerFormPageData_ReadOnlyDenied(t *testing.T) {
	// The read-only viewer case the round-2 finding named: granting ONLY
	// :read (no :update) must deny the drawer, even though the same viewer
	// can still load the read-only matrix via the sibling use case.
	authz := allowOnlyEntryPermission(entrySetReadPermission)
	uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz))
	if _, err := uc.Execute(newEntryAuthzContext(), nil); err == nil {
		t.Fatal("expected denial: rating_description_set:read alone must not open the mutating drawer")
	}
}

// TestGetRatingDescriptionSetEntryDrawerFormPageData_PermissionMatrix
// exercises the three copya.md-prescribed roles directly.
func TestGetRatingDescriptionSetEntryDrawerFormPageData_PermissionMatrix(t *testing.T) {
	cases := []struct {
		role    string
		granted []string
		wantOK  bool
	}{
		{role: "Education Admin (all 12, incl. set update)", granted: []string{entrySetReadPermission, entrySetUpdatePermission}, wantOK: true},
		{role: "AY Coordinator (link CRUD + set list/read, NO set update)", granted: []string{entrySetReadPermission}, wantOK: false},
		{role: "Section Template Manager (set list/read only)", granted: []string{entrySetReadPermission}, wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			perms := map[string]bool{}
			for _, p := range tc.granted {
				perms[p] = true
			}
			authz := &entryAuthzTestAuthorizer{enabled: true, perms: perms}
			uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz))
			_, err := uc.Execute(newEntryAuthzContext(), nil)
			gotOK := err == nil
			if gotOK != tc.wantOK {
				t.Fatalf("%s: expected drawer access=%v, got %v (err=%v)", tc.role, tc.wantOK, gotOK, err)
			}
		})
	}
}

// fakeParentSetReadRepo implements ONLY ReadRatingDescriptionSet, standing
// in for the parent-authorized read GetRatingDescriptionSetEntryDrawer-
// FormPageDataUseCase performs itself (via WithParentReads) — never through
// rating_description_set's own :read-gated ReadRatingDescriptionSetUseCase.
type fakeParentSetReadRepo struct {
	setpb.UnimplementedRatingDescriptionSetDomainServiceServer
	scoreScaleID string
}

func (r *fakeParentSetReadRepo) ReadRatingDescriptionSet(_ context.Context, req *setpb.ReadRatingDescriptionSetRequest) (*setpb.ReadRatingDescriptionSetResponse, error) {
	return &setpb.ReadRatingDescriptionSetResponse{
		Data:    []*setpb.RatingDescriptionSet{{Id: req.GetData().GetId(), ScoreScaleId: r.scoreScaleID}},
		Success: true,
	}, nil
}

// TestGetRatingDescriptionSetEntryDrawerFormPageData_EditLoadsEntryAndScaleForUpdateOnlyEditor
// is the codex-review-impl4.out.md "Update-only drawer" case directly: an
// editor holding ONLY rating_description_set:update (no separate :read)
// must still be able to load the Edit drawer's entry AND its parent set's
// score_scale_id — both through THIS use case's single :update
// authorization, never the entry's or the set's own separate :read gate
// (only ONE authorization check is made at all).
func TestGetRatingDescriptionSetEntryDrawerFormPageData_EditLoadsEntryAndScaleForUpdateOnlyEditor(t *testing.T) {
	authz := allowOnlyEntryPermission(entrySetUpdatePermission)
	entryRepo := &fakeEntryCRUDRepo{} // ReadRatingDescriptionSetEntry -> {Id: "entry-1"}
	setRepo := &fakeParentSetReadRepo{scoreScaleID: "scale-9"}
	uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz)).
		WithParentReads(setRepo, entryRepo)

	resp, err := uc.ExecuteForIDs(newEntryAuthzContext(), "", "entry-1")
	if err != nil {
		t.Fatalf("expected the update-only editor to load the Edit drawer, got %v", err)
	}
	if resp.Entry == nil || resp.Entry.ID != "entry-1" {
		t.Fatalf("expected the entry to be resolved via the parent-authorized read, got %+v", resp.Entry)
	}
	if len(authz.asked) != 1 || authz.asked[0] != entrySetUpdatePermission {
		t.Fatalf("expected exactly one authorization check (%q), got %v", entrySetUpdatePermission, authz.asked)
	}
}

// TestGetRatingDescriptionSetEntryDrawerFormPageData_AddResolvesScaleForUpdateOnlyEditor
// is the Add-drawer half of the same finding: the parent-scale lookup for
// the Level picker must not require rating_description_set:read either.
func TestGetRatingDescriptionSetEntryDrawerFormPageData_AddResolvesScaleForUpdateOnlyEditor(t *testing.T) {
	authz := allowOnlyEntryPermission(entrySetUpdatePermission)
	setRepo := &fakeParentSetReadRepo{scoreScaleID: "scale-7"}
	uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz)).
		WithParentReads(setRepo, nil)

	resp, err := uc.ExecuteForIDs(newEntryAuthzContext(), "set-1", "")
	if err != nil {
		t.Fatalf("expected the update-only editor to load the Add drawer's Level picker scale, got %v", err)
	}
	if resp.ScoreScaleId != "scale-7" {
		t.Fatalf("expected score_scale_id %q, got %q", "scale-7", resp.ScoreScaleId)
	}
	if resp.Entry != nil {
		t.Fatalf("expected no entry for the Add drawer, got %+v", resp.Entry)
	}
	if len(authz.asked) != 1 || authz.asked[0] != entrySetUpdatePermission {
		t.Fatalf("expected exactly one authorization check (%q), got %v", entrySetUpdatePermission, authz.asked)
	}
}

// fakeEmptyEntryReadRepo's ReadRatingDescriptionSetEntry returns zero rows
// (an id that does not resolve) so
// TestGetRatingDescriptionSetEntryDrawerFormPageData_UnknownEntryIsNilNotError
// can prove that is reported as a nil Entry with a nil error — absence, not
// a technical failure (the fayna caller maps a nil Entry to Not Found).
type fakeEmptyEntryReadRepo struct {
	entrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
}

func (r *fakeEmptyEntryReadRepo) ReadRatingDescriptionSetEntry(_ context.Context, _ *entrypb.ReadRatingDescriptionSetEntryRequest) (*entrypb.ReadRatingDescriptionSetEntryResponse, error) {
	return &entrypb.ReadRatingDescriptionSetEntryResponse{Success: true}, nil
}

// TestGetRatingDescriptionSetEntryDrawerFormPageData_UnknownEntryIsNilNotError
// proves an entry id that does not resolve to a row is reported as a nil
// Entry with a nil error, never a hard failure.
func TestGetRatingDescriptionSetEntryDrawerFormPageData_UnknownEntryIsNilNotError(t *testing.T) {
	authz := allowOnlyEntryPermission(entrySetUpdatePermission)
	entryRepo := &fakeEmptyEntryReadRepo{}
	uc := NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(GetRatingDescriptionSetEntryFormPageDataRepositories{}, entryFormPickerServices(authz)).
		WithParentReads(nil, entryRepo)

	resp, err := uc.ExecuteForIDs(newEntryAuthzContext(), "", "missing-entry")
	if err != nil {
		t.Fatalf("expected a nil error for an unresolved entry id, got %v", err)
	}
	if resp.Entry != nil {
		t.Fatalf("expected a nil Entry for an unresolved entry id, got %+v", resp.Entry)
	}
}

// TestFetchEntryFormPickerData_PagesPastSingleAdapterPage proves the
// criteria/band picker reads loop through ALL pages instead of stopping at
// the adapter's 100-row default cap (codex-review-impl4.out.md round-3/
// round-4 disposition #2: "criteria/bands beyond 100 remain unavailable").
// 133 criteria and 141 bands (each > one 100-row page, not a clean
// multiple) must all come back, proving the loop's short-page stop
// condition is correct and not an off-by-one.
func TestFetchEntryFormPickerData_PagesPastSingleAdapterPage(t *testing.T) {
	criteria := make([]*criteriapb.OutcomeCriteria, 0, 133)
	for i := 0; i < 133; i++ {
		criteria = append(criteria, &criteriapb.OutcomeCriteria{Id: fmt.Sprintf("crit-%03d", i), Name: fmt.Sprintf("Criterion %03d", i), Active: true})
	}
	bands := make([]*scalebandpb.ScoreScaleBand, 0, 141)
	for i := 0; i < 141; i++ {
		bands = append(bands, &scalebandpb.ScoreScaleBand{Id: fmt.Sprintf("band-%03d", i), OutputLabel: fmt.Sprintf("Band %03d", i), ScoreScaleId: "scale-1", Active: true, SequenceOrder: int32(i)})
	}
	criteriaRepo := &pagedCriteriaRepo{items: criteria}
	bandRepo := &pagedBandRepo{items: bands}

	resp, err := fetchEntryFormPickerData(context.Background(), GetRatingDescriptionSetEntryFormPageDataRepositories{
		OutcomeCriteria: criteriaRepo,
		ScoreScaleBand:  bandRepo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Criteria) != 133 {
		t.Fatalf("expected all 133 criteria, got %d", len(resp.Criteria))
	}
	if len(resp.Bands) != 141 {
		t.Fatalf("expected all 141 bands, got %d", len(resp.Bands))
	}
	if criteriaRepo.calls < 2 {
		t.Fatalf("expected the criteria picker to page (>1 call for 133 rows at 100/page), got %d calls", criteriaRepo.calls)
	}
	if bandRepo.calls < 2 {
		t.Fatalf("expected the band picker to page (>1 call for 141 rows at 100/page), got %d calls", bandRepo.calls)
	}
}

// pagedCriteriaRepo/pagedBandRepo honor the offset-pagination request
// (page number x limit) exactly like the real postgres adapter, so this
// test proves the picker loop's page-to-page advance and short-page stop
// condition, not just that "some" data came back.
type pagedCriteriaRepo struct {
	criteriapb.UnimplementedOutcomeCriteriaDomainServiceServer
	items []*criteriapb.OutcomeCriteria
	calls int
}

func (r *pagedCriteriaRepo) ListOutcomeCriterias(_ context.Context, req *criteriapb.ListOutcomeCriteriasRequest) (*criteriapb.ListOutcomeCriteriasResponse, error) {
	r.calls++
	page := int(req.GetPagination().GetOffset().GetPage())
	limit := int(req.GetPagination().GetLimit())
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(r.items) {
		start = len(r.items)
	}
	end := start + limit
	if end > len(r.items) {
		end = len(r.items)
	}
	return &criteriapb.ListOutcomeCriteriasResponse{Data: r.items[start:end], Success: true}, nil
}

type pagedBandRepo struct {
	scalebandpb.UnimplementedScoreScaleBandDomainServiceServer
	items []*scalebandpb.ScoreScaleBand
	calls int
}

func (r *pagedBandRepo) ListScoreScaleBands(_ context.Context, req *scalebandpb.ListScoreScaleBandsRequest) (*scalebandpb.ListScoreScaleBandsResponse, error) {
	r.calls++
	page := int(req.GetPagination().GetOffset().GetPage())
	limit := int(req.GetPagination().GetLimit())
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(r.items) {
		start = len(r.items)
	}
	end := start + limit
	if end > len(r.items) {
		end = len(r.items)
	}
	return &scalebandpb.ListScoreScaleBandsResponse{Data: r.items[start:end], Success: true}, nil
}
