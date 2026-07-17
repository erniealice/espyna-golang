package outcome_criteria

import (
	"context"
	"errors"
	"testing"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

func sp(s string) *string { return &s }

// fakeOCRepo is an OutcomeCriteria stub that answers ListOutcomeCriterias by the
// single equality filter the validators use (code / criteria_group_id). Only
// active rows are stored (matching the list default). Embedding the generated
// Unimplemented server satisfies the rest of the interface.
type fakeOCRepo struct {
	pb.UnimplementedOutcomeCriteriaDomainServiceServer
	rows    []*pb.OutcomeCriteria
	listErr error
}

func (f *fakeOCRepo) ListOutcomeCriterias(_ context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	field, value := "", ""
	if fr := req.GetFilters(); fr != nil && len(fr.Filters) > 0 {
		field = fr.Filters[0].GetField()
		value = fr.Filters[0].GetStringFilter().GetValue()
	}
	var out []*pb.OutcomeCriteria
	for _, r := range f.rows {
		switch field {
		case "code":
			if r.GetCode() == value {
				out = append(out, r)
			}
		case "criteria_group_id":
			if r.GetCriteriaGroupId() == value {
				out = append(out, r)
			}
		default:
			out = append(out, r)
		}
	}
	return &pb.ListOutcomeCriteriasResponse{Data: out, Success: true}, nil
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
		{Id: "oc-1", CriteriaGroupId: "g-existing", Code: sp("conduct"), WorkspaceId: sp("w1")},
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
		{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")},
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}

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
		{Id: "oc-g1", CriteriaGroupId: "g1", WorkspaceId: sp("w1")},               // g1: no code yet
		{Id: "oc-g2", CriteriaGroupId: "g2", Code: sp("attend"), WorkspaceId: sp("w1")}, // g2 holds "attend"
	}}
	uc := &UpdateOutcomeCriteriaUseCase{
		repositories: UpdateOutcomeCriteriaRepositories{OutcomeCriteria: repo},
		services:     UpdateOutcomeCriteriaServices{},
	}
	ctx := context.Background()
	existing := &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", WorkspaceId: sp("w1")}

	// Fill NULL -> a fresh, unused code → allowed.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", Code: sp("conduct"), WorkspaceId: sp("w1")}, existing); err != nil {
		t.Errorf("filling a NULL lineage code with an unused code must be allowed, got %v", err)
	}
	// Fill NULL -> a code held by another group in the same domain → rejected.
	if err := uc.checkCodeInvariants(ctx, &pb.OutcomeCriteria{Id: "oc-g1", CriteriaGroupId: "g1", Code: sp("attend"), WorkspaceId: sp("w1")}, existing); err == nil {
		t.Error("taking a code already held by another group in the same domain must be rejected")
	}
}
