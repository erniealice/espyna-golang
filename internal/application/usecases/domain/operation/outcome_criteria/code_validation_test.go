package outcome_criteria

import (
	"context"
	"errors"
	"fmt"
	"testing"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

func sp(s string) *string { return &s }

// pageWindow mirrors the generic adapter's 1-based offset windowing (limit capped
// at 100) so the fake pages exactly like the real List the validators drive.
func pageWindow(total int, p *commonpb.PaginationRequest) (start, end int) {
	page, limit := 1, ocReadPageSize
	if p != nil {
		if p.Limit > 0 && int(p.Limit) <= ocReadPageSize {
			limit = int(p.Limit)
		}
		if off := p.GetOffset(); off != nil && off.Page > 0 {
			page = int(off.Page)
		}
	}
	start = (page - 1) * limit
	if start > total {
		start = total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return start, end
}

// fakeOCRepo is an OutcomeCriteria stub that answers ListOutcomeCriterias like the
// real generic adapter: it applies the single equality filter the validators use
// (code / criteria_group_id), honours an explicit `active` boolean filter (and
// defaults to active-only when none is supplied), and paginates. Embedding the
// generated Unimplemented server satisfies the rest of the interface.
type fakeOCRepo struct {
	pb.UnimplementedOutcomeCriteriaDomainServiceServer
	rows      []*pb.OutcomeCriteria
	listErr   error
	listCalls int
}

func (f *fakeOCRepo) ListOutcomeCriterias(_ context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	eqField, eqValue := "", ""
	activeWanted := true // generic adapter default when no explicit active filter
	if fr := req.GetFilters(); fr != nil {
		for _, tf := range fr.Filters {
			if tf.GetField() == "active" {
				if bf, ok := tf.FilterType.(*commonpb.TypedFilter_BooleanFilter); ok {
					activeWanted = bf.BooleanFilter.GetValue()
					continue
				}
			}
			eqField = tf.GetField()
			eqValue = tf.GetStringFilter().GetValue()
		}
	}
	var matched []*pb.OutcomeCriteria
	for _, r := range f.rows {
		if r.GetActive() != activeWanted {
			continue
		}
		switch eqField {
		case "code":
			if r.GetCode() != eqValue {
				continue
			}
		case "criteria_group_id":
			if r.GetCriteriaGroupId() != eqValue {
				continue
			}
		}
		matched = append(matched, r)
	}
	start, end := pageWindow(len(matched), req.GetPagination())
	return &pb.ListOutcomeCriteriasResponse{Data: matched[start:end], Success: true}, nil
}

func TestNormalizeCriteriaCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"  Conduct ", "conduct", true},
		{"days_present", "days_present", true},
		{"UPPER", "upper", true},
		{"m07", "m07", true},
		{"1abc", "", false},   // must start with a letter
		{"bad-code", "", false}, // hyphen not allowed
		{"has space", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeCriteriaCode(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("normalizeCriteriaCode(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestCreateCodeInvariants_DomainCollision: a create reusing an ACTIVE code held
// by another group in the SAME domain is rejected; a different domain or the same
// lineage is allowed.
func TestCreateCodeInvariants_DomainCollision(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g-existing", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := context.Background()

	// Different group, same domain → collision.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g-new", Code: sp("conduct"), WorkspaceId: sp("w1")}); err == nil {
		t.Error("expected a collision error for a code reused by another group in the same domain")
	}
	// Different workspace → different domain → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g-new", Code: sp("conduct"), WorkspaceId: sp("w2")}); err != nil {
		t.Errorf("a code reused in a DIFFERENT domain must be allowed, got %v", err)
	}
	// Same lineage legitimately shares the code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g-existing", Code: sp("conduct"), WorkspaceId: sp("w1")}); err != nil {
		t.Errorf("the criterion's own lineage may share the code, got %v", err)
	}
	// No code → no-op.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g-new", WorkspaceId: sp("w1")}); err != nil {
		t.Errorf("no code supplied must be a no-op, got %v", err)
	}
}

func TestCreateCodeInvariants_ReadErrorPropagates(t *testing.T) {
	repo := &fakeOCRepo{listErr: errors.New("boom")}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	if err := uc.checkCodeInvariants(context.Background(), &pb.OutcomeCriteria{CriteriaGroupId: "g", Code: sp("conduct"), WorkspaceId: sp("w1")}); err == nil {
		t.Error("a uniqueness-read failure must surface as an error, not be swallowed")
	}
}

// TestUpdateCodeInvariants_LineageImmutable: once a group has an established code,
// changing it to a different code is rejected; filling NULL->established or a
// no-op reassignment is allowed.
func TestUpdateCodeInvariants_LineageImmutable(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true}

	// Mutating an established code to a DIFFERENT value → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("other_code"), WorkspaceId: sp("w1")}, existing); err == nil {
		t.Error("changing an established lineage code to a different value must be rejected")
	}
	// Re-asserting the SAME established code → allowed (and no self-collision).
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}, existing); err != nil {
		t.Errorf("re-asserting the same established code must be allowed, got %v", err)
	}
}

// TestUpdateCodeInvariants_FillNullAndCollision: a lineage with no established
// code may take a new code (NULL->code), but not one already held by another
// group in the same domain.
func TestUpdateCodeInvariants_FillNullAndCollision(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-g1", CriteriaGroupId: "g1", WorkspaceId: sp("w1"), Active: true},               // g1: no code yet
		{Id: "oc-g2", CriteriaGroupId: "g2", Code: sp("attend"), WorkspaceId: sp("w1"), Active: true}, // g2 holds "attend"
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", WorkspaceId: sp("w1"), Active: true}

	// Fill NULL -> a fresh, unused code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}, existing); err != nil {
		t.Errorf("filling a NULL lineage code with an unused code must be allowed, got %v", err)
	}
	// Fill NULL -> a code held by another group in the same domain → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", Code: sp("attend"), WorkspaceId: sp("w1")}, existing); err == nil {
		t.Error("taking a code already held by another group in the same domain must be rejected")
	}
}

// TestCreateCodeInvariants_LineageRejection locks the CREATE-path lineage wiring
// (the exact gap codex flagged: Create never compared the proposed code with its
// group's established code). A create adding a new version under an existing
// group with a DIFFERENT code must be rejected; continuing the established code,
// or opening a brand-new lineage, is allowed.
func TestCreateCodeInvariants_LineageRejection(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := context.Background()

	// New version under established group g1 with a DIFFERENT code → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("other_code"), WorkspaceId: sp("w1"), Active: true}); err == nil {
		t.Error("a create under an established lineage with a different code must be rejected")
	}
	// New version under g1 continuing the established code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true}); err != nil {
		t.Errorf("a create continuing the lineage's established code must be allowed, got %v", err)
	}
	// Brand-new lineage → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g-brand-new", Code: sp("fresh_code"), WorkspaceId: sp("w1"), Active: true}); err != nil {
		t.Errorf("a create of a brand-new lineage must be allowed, got %v", err)
	}
}

// TestListAllVersionsMatching_PaginationCompleteness locks the truncation fix: a
// group whose versions spill past a single default page must still be read
// COMPLETELY (the old read capped at the adapter's 100-row default). A version on
// the second page cannot be dropped.
func TestListAllVersionsMatching_PaginationCompleteness(t *testing.T) {
	var rows []*pb.OutcomeCriteria
	for i := 0; i < 150; i++ {
		rows = append(rows, &pb.OutcomeCriteria{
			Id: fmt.Sprintf("oc-%d", i), CriteriaGroupId: "g1",
			Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true,
		})
	}
	repo := &fakeOCRepo{rows: rows}

	got, err := listAllVersionsMatching(context.Background(), repo, "criteria_group_id", "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 150 {
		t.Fatalf("expected 150 rows from a complete cross-page read, got %d (truncation)", len(got))
	}
	if repo.listCalls < 2 {
		t.Fatalf("expected the read to page (>=2 list calls), got %d", repo.listCalls)
	}
}

// TestLineageEstablishedCode_IncludesInactive locks the all-version scope: a
// lineage whose ONLY coded version is INACTIVE (soft-deleted) still anchors the
// code. An active-only read (the old default) would have returned "" and
// false-allowed a divergent code — inconsistent with the DB criteria_group
// anchor, which reserves the code across all versions.
func TestLineageEstablishedCode_IncludesInactive(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-dead", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: false},
	}}
	code, err := lineageEstablishedCode(context.Background(), repo, "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "conduct" {
		t.Fatalf("an inactive (soft-deleted) version must still anchor the lineage code; got %q", code)
	}
}

// ── Purpose-built anchor-reader seam ────────────────────────────────────────

// fakeAnchorRepo implements the lineageAnchorReader seam over in-memory maps,
// mimicking the PostgreSQL adapter's point lookups against criteria_group. Its
// embedded fakeOCRepo would serve the paginated fallback — the tests assert
// listCalls stays 0, proving the purpose-built path is PREFERRED.
type fakeAnchorRepo struct {
	fakeOCRepo
	anchors      map[string]string    // criteria_group_id -> established code
	owners       map[string]string    // scope|ws|industry|code -> owning group id
	domains      map[string][3]string // criteria_group_id -> {scope, workspace, industry} claim
	lineageCalls int
	ownerCalls   int
	domainCalls  int
}

func (f *fakeAnchorRepo) LineageEstablishedCode(_ context.Context, criteriaGroupID string) (string, error) {
	f.lineageCalls++
	return f.anchors[criteriaGroupID], nil
}

func (f *fakeAnchorRepo) CodeOwnerGroup(_ context.Context, scopeKey, workspaceKey, industryKey, code string) (string, error) {
	f.ownerCalls++
	return f.owners[scopeKey+"|"+workspaceKey+"|"+industryKey+"|"+code], nil
}

func (f *fakeAnchorRepo) LineageClaimedDomain(_ context.Context, criteriaGroupID string) (string, string, string, bool, error) {
	f.domainCalls++
	d, ok := f.domains[criteriaGroupID]
	return d[0], d[1], d[2], ok, nil
}

// TestCheckCodeInvariants_PurposeBuiltPathPreferred proves that when the
// repository offers the anchor point lookups, the validators use them and never
// touch the paginated list fallback — the codex FIX-FIRST 1 shape (bounded,
// indexed, single-statement reads instead of a 100xN page scan).
func TestCheckCodeInvariants_PurposeBuiltPathPreferred(t *testing.T) {
	repo := &fakeAnchorRepo{
		anchors: map[string]string{"g1": "conduct"},
		owners:  map[string]string{"|w1||conduct": "g1"},
		domains: map[string][3]string{"g1": {"", "w1", ""}},
	}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := context.Background()

	// Same lineage continuing its code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}); err != nil {
		t.Errorf("continuing the anchored code must be allowed, got %v", err)
	}
	// Different code under the anchored lineage → lineage rejection.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("other_code"), WorkspaceId: sp("w1")}); err == nil {
		t.Error("a divergent code under an anchored lineage must be rejected")
	}
	// Different group claiming the owned domain code → collision rejection.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g2", Code: sp("conduct"), WorkspaceId: sp("w1")}); err == nil {
		t.Error("a code owned by another group in the same domain must be rejected")
	}
	if repo.listCalls != 0 {
		t.Errorf("purpose-built path must not fall back to the paginated list (listCalls=%d)", repo.listCalls)
	}
	if repo.lineageCalls == 0 || repo.ownerCalls == 0 {
		t.Errorf("expected anchor point lookups to be used (lineageCalls=%d ownerCalls=%d)", repo.lineageCalls, repo.ownerCalls)
	}
}

// ── Trusted-workspace stamping (finding 10) ─────────────────────────────────

// TestStampTrustedWorkspace: the context workspace (the SAME source persistence
// injects) overwrites an omitted OR forged client workspace before validation;
// a workspace-less context (service-to-service) leaves the payload untouched.
func TestStampTrustedWorkspace(t *testing.T) {
	trusted := contextutil.WithWorkspaceID(context.Background(), "w1")

	forged := &pb.OutcomeCriteria{WorkspaceId: sp("w2")}
	stampTrustedWorkspace(trusted, forged)
	if forged.GetWorkspaceId() != "w1" {
		t.Errorf("a forged workspace must be overwritten with the trusted context value, got %q", forged.GetWorkspaceId())
	}

	omitted := &pb.OutcomeCriteria{}
	stampTrustedWorkspace(trusted, omitted)
	if omitted.GetWorkspaceId() != "w1" {
		t.Errorf("an omitted workspace must be stamped with the trusted context value, got %q", omitted.GetWorkspaceId())
	}

	s2s := &pb.OutcomeCriteria{WorkspaceId: sp("w-explicit")}
	stampTrustedWorkspace(context.Background(), s2s)
	if s2s.GetWorkspaceId() != "w-explicit" {
		t.Errorf("a workspace-less context must leave the payload untouched, got %q", s2s.GetWorkspaceId())
	}
}

// TestCreateCodeInvariants_ForgedWorkspaceCaught reproduces finding 10 end to
// end at the use-case level: the trusted list scoping returns w1 rows, so a
// proposal carrying a FORGED (w2) or OMITTED workspace used to compare w1 !=
// w2/"" and false-allow the collision. After stamping, both are caught.
func TestCreateCodeInvariants_ForgedWorkspaceCaught(t *testing.T) {
	// fakeOCRepo, like the real WorkspaceAware List, returns the w1 row
	// regardless of what the client claims its own workspace is.
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g-existing", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := contextutil.WithWorkspaceID(context.Background(), "w1")

	for name, data := range map[string]*pb.OutcomeCriteria{
		"forged":  {CriteriaGroupId: "g-new", Code: sp("conduct"), WorkspaceId: sp("w2")},
		"omitted": {CriteriaGroupId: "g-new", Code: sp("conduct")},
	} {
		stampTrustedWorkspace(ctx, data)
		if err := uc.checkCodeInvariants(ctx, data); err == nil {
			t.Errorf("%s workspace: the collision must be caught after trusted stamping", name)
		}
	}
}

// ── Coded writes require a lineage (finding 11) ─────────────────────────────

func TestCreateValidation_CodeRequiresGroup(t *testing.T) {
	uc := &CreateOutcomeCriteriaUseCase{services: CreateOutcomeCriteriaServices{}}
	ctx := context.Background()

	if err := uc.validateBusinessRules(ctx, &pb.OutcomeCriteria{Name: "n", Code: sp("conduct")}); err == nil {
		t.Error("a coded create without criteria_group_id must fail validation")
	}
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Code: sp("conduct"), WorkspaceId: sp("w1")}); err == nil {
		t.Error("checkCodeInvariants must also reject a coded proposal without a group (defense in depth)")
	}
	if err := uc.validateBusinessRules(ctx, &pb.OutcomeCriteria{Name: "n"}); err != nil {
		t.Errorf("an uncoded create without a group stays valid, got %v", err)
	}
}

// ── Update merged-row semantics: re-home + mixed-NULL lineages ──────────────

// TestUpdateCodeInvariants_ReHomeCaught locks the finding-2B non-concurrent
// route: moving a coded row to a FRESH group WITHOUT resupplying its code used
// to no-op the validator (code omitted) while persistence re-homed the row.
// The merged-row check must treat the stored code + supplied group as the
// effective claim and reject it against the old group's standing ownership.
func TestUpdateCodeInvariants_ReHomeCaught(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true}

	// Re-home WITHOUT resupplying code → old group's claim collides.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g-fresh"}, existing); err == nil {
		t.Error("re-homing a coded row to a fresh group without resupplying code must be rejected")
	}
	// Re-home WITH the code resupplied → same rejection.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g-fresh", Code: sp("conduct")}, existing); err == nil {
		t.Error("re-homing a coded row with its code must be rejected against the old group's claim")
	}
	// A no-move partial update of the coded row stays allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1"}, existing); err != nil {
		t.Errorf("a partial update that keeps group+code must pass, got %v", err)
	}
	// A coded row whose existing group is EMPTY (legacy) must be rejected on any
	// code-bearing update rather than anchoring to a phantom lineage.
	legacy := &pb.OutcomeCriteria{Id: "oc-legacy", CriteriaGroupId: "", Code: sp("orphan_code"), WorkspaceId: sp("w1")}
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-legacy", Code: sp("orphan_code")}, legacy); err == nil {
		t.Error("a coded update on a group-less row must be rejected (code requires a lineage)")
	}
}

// TestUpdateCodeInvariants_MixedNullLineage locks the redefined NULL contract
// (finding 1A resolution): NULL-code versions are PERMITTED inside an anchored
// group — only CODED versions must agree. Coding the NULL sibling with the
// established code is allowed; coding it divergently is rejected.
func TestUpdateCodeInvariants_MixedNullLineage(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-coded", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
		{Id: "oc-null", CriteriaGroupId: "g1", WorkspaceId: sp("w1"), Active: true}, // NULL-code sibling — legal
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	nullSibling := &pb.OutcomeCriteria{Id: "oc-null", CriteriaGroupId: "g1", WorkspaceId: sp("w1"), Active: true}

	// Partial update of the NULL sibling (still uncoded) → no-op, allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-null"}, nullSibling); err != nil {
		t.Errorf("an uncoded version inside an anchored group is permitted (coded-versions-agree contract), got %v", err)
	}
	// Coding it with the established code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-null", Code: sp("conduct")}, nullSibling); err != nil {
		t.Errorf("coding a NULL sibling with the established code must be allowed, got %v", err)
	}
	// Coding it divergently → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-null", Code: sp("other_code")}, nullSibling); err == nil {
		t.Error("coding a NULL sibling with a DIVERGENT code must be rejected")
	}
}

// ── NEW-1: lineage domain consistency on coded writes ───────────────────────

// TestCreateCodeInvariants_DomainConsistency_AnchorSeam locks the NEW-1 guard
// on the purpose-built path: a coded create into an EXISTING group whose
// (scope, workspace, industry) diverges from the group's claimed domain is
// rejected — the DB cannot catch this (the populate trigger's ON CONFLICT DO
// NOTHING never re-stamps an existing anchor; the domain unique index only
// fires on NEW anchors). Matching domain accepted; NULL-code exempt.
func TestCreateCodeInvariants_DomainConsistency_AnchorSeam(t *testing.T) {
	repo := &fakeAnchorRepo{
		anchors: map[string]string{"g1": "conduct"},
		owners:  map[string]string{"|w1||conduct": "g1"},
		domains: map[string][3]string{"g1": {"", "w1", ""}},
	}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := context.Background()

	// Matching domain → accepted.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}); err != nil {
		t.Errorf("a coded create matching the group's claimed domain must be allowed, got %v", err)
	}
	// Divergent workspace → rejected on the domain claim.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w2")}); err == nil {
		t.Error("a coded create whose workspace diverges from the group's claimed domain must be rejected")
	}
	// Divergent scope → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Scope: 2}); err == nil {
		t.Error("a coded create whose scope diverges from the group's claimed domain must be rejected")
	}
	// Divergent industry → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), IndustryCode: sp("edu")}); err == nil {
		t.Error("a coded create whose industry diverges from the group's claimed domain must be rejected")
	}
	// NULL-code write into the same group with a divergent domain → exempt.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", WorkspaceId: sp("w2")}); err != nil {
		t.Errorf("an uncoded write is exempt from the domain guard, got %v", err)
	}
	if repo.domainCalls == 0 {
		t.Error("expected the purpose-built LineageClaimedDomain lookup to be used")
	}
	if repo.listCalls != 0 {
		t.Errorf("purpose-built path must not fall back to the paginated list (listCalls=%d)", repo.listCalls)
	}
}

// TestCreateCodeInvariants_DomainConsistency_Fallback proves the same guard on
// the seam-less fallback path: the claim derives from the lineage's first
// coded version via the bounded complete scan.
func TestCreateCodeInvariants_DomainConsistency_Fallback(t *testing.T) {
	repo := &fakeOCRepo{rows: []*pb.OutcomeCriteria{
		{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true},
	}}
	uc := &CreateOutcomeCriteriaUseCase{
		repositories: CreateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     CreateOutcomeCriteriaServices{},
	}
	ctx := context.Background()

	// Divergent workspace (same code, same group) → rejected. Note the domain
	// collision check would NOT catch this (different domain ⇒ not a
	// collision), so a rejection proves the domain guard specifically.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w2")}); err == nil {
		t.Error("fallback path: a divergent-workspace coded create must be rejected")
	}
	// Matching domain → accepted.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}); err != nil {
		t.Errorf("fallback path: a matching-domain coded create must be allowed, got %v", err)
	}
}

// TestUpdateCodeInvariants_DomainConsistency locks the merged-row leg: an
// update supplying a divergent scope/industry onto a CODED row (code kept from
// the stored row) is rejected against the group's claimed domain; a no-change
// partial update passes; an uncoded row stays exempt.
func TestUpdateCodeInvariants_DomainConsistency(t *testing.T) {
	repo := &fakeAnchorRepo{
		anchors: map[string]string{"g1": "conduct"},
		owners:  map[string]string{"|w1||conduct": "g1"},
		domains: map[string][3]string{"g1": {"", "w1", ""}},
	}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1"), Active: true}

	// Supplied scope diverging from the claim (code kept via merge) → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", Scope: 2}, existing); err == nil {
		t.Error("an update moving a coded row's scope off its group's claimed domain must be rejected")
	}
	// Supplied industry diverging → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1", IndustryCode: sp("edu")}, existing); err == nil {
		t.Error("an update moving a coded row's industry off its group's claimed domain must be rejected")
	}
	// No-change partial update → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-1"}, existing); err != nil {
		t.Errorf("a partial update keeping the claimed domain must pass, got %v", err)
	}
	// Uncoded row (stays uncoded) with a divergent scope → exempt.
	uncoded := &pb.OutcomeCriteria{Id: "oc-null", CriteriaGroupId: "g1", WorkspaceId: sp("w1"), Active: true}
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-null", Scope: 2}, uncoded); err != nil {
		t.Errorf("an uncoded row is exempt from the domain guard, got %v", err)
	}
}

// TestCriteriaScopeKey pins the anchor's scope normalization: zero enum -> ''
// (protojson omits zero enums -> NULL column -> COALESCE ''), else the enum
// name exactly as protojson persists it.
func TestCriteriaScopeKey(t *testing.T) {
	if got := criteriaScopeKey(&pb.OutcomeCriteria{}); got != "" {
		t.Errorf("unspecified scope must normalize to '', got %q", got)
	}
	coded := &pb.OutcomeCriteria{Scope: 2} // any non-zero member
	if got := criteriaScopeKey(coded); got != coded.GetScope().String() || got == "" {
		t.Errorf("non-zero scope must normalize to its enum name, got %q", got)
	}
}
