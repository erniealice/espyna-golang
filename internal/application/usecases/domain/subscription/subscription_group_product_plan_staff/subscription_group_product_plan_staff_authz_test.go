package subscription_group_product_plan_staff

// Deny-direction authorization tests for the class-ASSIGNMENT (sgpps) use cases
// — the espyna half of coverage-audit gap A-G3
// (docs/plan/20260724-section-assignment-merged/coverage-audit-20260725.md).
//
// Every other test in this package builds its ActionGatekeeper from
// ports.NewNoOpAuthorizer(), whose IsEnabled() == false makes
// ActionGatekeeper.Check return nil BEFORE it computes a permission code. So no
// test here has ever exercised a real denial: neither a wrong permission string
// nor a deleted gate would fail anything, and the live server runs
// AUTHZ_ENFORCE="" (SHADOW), so it would not catch it either.
//
// These tests wire an authorizer with IsEnabled() == true — the enforce-style
// shape of subscription/create_subscription_materialize_authz_test.go's
// splitAuthorizer — and assert, for each gated use case:
//
//	deny  — the principal holds every permission EXCEPT the one this use case
//	        must demand: Execute errors with the permission-denied message, the
//	        repository records ZERO calls, and the gate consulted EXACTLY that
//	        one permission string.
//	allow — the principal holds ONLY that permission: Execute reaches the
//	        repository on an otherwise-valid edge graph.
//
// The `asked` assertion is what pins the STRING rather than merely the presence
// of a gate: with allowAllExcept, a use case demanding some other code sails
// straight through and the deny case fails.
//
// The expected codes are written as LITERALS on purpose. They are the contract
// with the seeded permission.code rows; deriving them from the same
// entityid constants the code under test uses would be circular.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

const (
	permEdgeCreate = "subscription_group_product_plan_staff:create"
	permEdgeRead   = "subscription_group_product_plan_staff:read"
	permEdgeUpdate = "subscription_group_product_plan_staff:update"
	permEdgeDelete = "subscription_group_product_plan_staff:delete"
	permEdgeList   = "subscription_group_product_plan_staff:list"
)

// ----- authorizer -------------------------------------------------------------

// edgeAuthorizer is an actiongate.Authorizer with enforcement ON, recording
// every permission code it is asked about.
type edgeAuthorizer struct {
	holds func(permission string) bool
	asked []string
}

func (a *edgeAuthorizer) IsEnabled() bool { return true }

func (a *edgeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	return a.holds(permission), nil
}

// edgeAllowAllExcept models a principal holding every permission in the system
// EXCEPT the one under test.
func edgeAllowAllExcept(code string) *edgeAuthorizer {
	return &edgeAuthorizer{holds: func(p string) bool { return p != code }}
}

// edgeGrantOnly models a least-privilege principal holding ONLY the code under test.
func edgeGrantOnly(code string) *edgeAuthorizer {
	return &edgeAuthorizer{holds: func(p string) bool { return p == code }}
}

// ----- repository -------------------------------------------------------------

// authzEdgeRepo counts every domain-service method the gated use cases can
// reach. Update and Assign read/list before they write, so "the repository was
// never called" is only meaningful as a total.
type authzEdgeRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	createCalls   int
	readCalls     int
	updateCalls   int
	deleteCalls   int
	listCalls     int
	listPageCalls int
	itemPageCalls int
	row           *pb.SubscriptionGroupProductPlanStaff
	rows          []*pb.SubscriptionGroupProductPlanStaff
}

func (m *authzEdgeRepo) totalCalls() int {
	return m.createCalls + m.readCalls + m.updateCalls + m.deleteCalls + m.listCalls + m.listPageCalls + m.itemPageCalls
}

func (m *authzEdgeRepo) CreateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	m.createCalls++
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

func (m *authzEdgeRepo) ReadSubscriptionGroupProductPlanStaff(_ context.Context, _ *pb.ReadSubscriptionGroupProductPlanStaffRequest) (*pb.ReadSubscriptionGroupProductPlanStaffResponse, error) {
	m.readCalls++
	if m.row == nil {
		return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
	}
	return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{m.row}}, nil
}

func (m *authzEdgeRepo) UpdateSubscriptionGroupProductPlanStaff(_ context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	m.updateCalls++
	return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlanStaff{req.GetData()}}, nil
}

func (m *authzEdgeRepo) DeleteSubscriptionGroupProductPlanStaff(_ context.Context, _ *pb.DeleteSubscriptionGroupProductPlanStaffRequest) (*pb.DeleteSubscriptionGroupProductPlanStaffResponse, error) {
	m.deleteCalls++
	return &pb.DeleteSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
}

func (m *authzEdgeRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	m.listCalls++
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.rows}, nil
}

func (m *authzEdgeRepo) GetSubscriptionGroupProductPlanStaffListPageData(_ context.Context, _ *pb.GetSubscriptionGroupProductPlanStaffListPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse, error) {
	m.listPageCalls++
	return &pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse{}, nil
}

func (m *authzEdgeRepo) GetSubscriptionGroupProductPlanStaffItemPageData(_ context.Context, _ *pb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse, error) {
	m.itemPageCalls++
	return &pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse{}, nil
}

// ----- fixture ----------------------------------------------------------------

// authzEdgeFixture wires one authorizer + one counting repo into the gated use
// cases over a fully ELIGIBLE graph (pps-1: pp-1/staff-1 active in ws-1;
// section sg-1 on plan-1 in ws-1; pp-1 on plan-1), so nothing downstream of the
// gate can reject the request. Any error the allow direction sees is therefore
// a real finding, not fixture noise. The edge itself is legacy-only (no f12/f13),
// which keeps resolveClassEdgeV2 and checkDuplicateClassEdge out of the picture:
// this file is about the gate and nothing else.
type authzEdgeFixture struct {
	repo  *authzEdgeRepo
	authz *edgeAuthorizer
	pps   *mockPPSRepo
	pp    *mockPPRepo
	sg    *mockSGRepo
}

func newAuthzEdgeFixture(a *edgeAuthorizer) *authzEdgeFixture {
	return &authzEdgeFixture{
		authz: a,
		repo: &authzEdgeRepo{row: &pb.SubscriptionGroupProductPlanStaff{
			Id:                  "edge-1",
			Active:              true,
			WorkspaceId:         "ws-1",
			SubscriptionGroupId: "sg-1",
			ProductPlanId:       "pp-1",
			StaffId:             "staff-1",
		}},
		pps: &mockPPSRepo{rows: []*productplanstaffpb.ProductPlanStaff{eligibleRow("ws-1")}},
		pp:  &mockPPRepo{planByID: map[string]string{"pp-1": "plan-1"}},
		sg:  &mockSGRepo{group: group("plan-1", "ws-1")},
	}
}

// gate is the only consumer of the recording authorizer. Services.Authorizer
// stays the no-op: these use cases never call it directly, the ActionGatekeeper
// is the whole authorization surface.
func (f *authzEdgeFixture) gate() *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(f.authz, ports.NewNoOpTranslator())
}

func (f *authzEdgeFixture) services() Services {
	return Services{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Translator:       ports.NewNoOpTranslator(),
		IDGenerator:      stubIDSGPPS{},
		ActionGatekeeper: f.gate(),
	}
}

// edgeAuthzCtx carries a user id — without one ActionGatekeeper.Check fails on a
// DIFFERENT branch ("Authorization failed"), which would make the deny assertion
// ambiguous — and the workspace the fixture graph lives in.
func edgeAuthzCtx() context.Context {
	return appcontext.WithWorkspaceID(appcontext.WithUserID(context.Background(), "teacher-1"), "ws-1")
}

type edgeAuthzCase struct {
	name       string
	permission string
	exec       func(ctx context.Context, f *authzEdgeFixture) error
}

func edgeAuthzCases() []edgeAuthzCase {
	return []edgeAuthzCase{
		{
			name:       "Create",
			permission: permEdgeCreate,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewCreateSubscriptionGroupProductPlanStaffUseCase(
					CreateSubscriptionGroupProductPlanStaffRepositories{
						SubscriptionGroupProductPlanStaff: f.repo,
						ProductPlanStaff:                  f.pps,
						ProductPlan:                       f.pp,
						SubscriptionGroup:                 f.sg,
					},
					CreateSubscriptionGroupProductPlanStaffServices(s),
				)
				_, err := uc.Execute(ctx, &pb.CreateSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{SubscriptionGroupId: "sg-1", ProductPlanId: "pp-1", StaffId: "staff-1"},
				})
				return err
			},
		},
		{
			name:       "Read",
			permission: permEdgeRead,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewReadSubscriptionGroupProductPlanStaffUseCase(
					ReadSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: f.repo},
					ReadSubscriptionGroupProductPlanStaffServices(s),
				)
				_, err := uc.Execute(ctx, &pb.ReadSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{Id: "edge-1"},
				})
				return err
			},
		},
		{
			name:       "Update",
			permission: permEdgeUpdate,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewUpdateSubscriptionGroupProductPlanStaffUseCase(
					UpdateSubscriptionGroupProductPlanStaffRepositories{
						SubscriptionGroupProductPlanStaff: f.repo,
						ProductPlanStaff:                  f.pps,
						ProductPlan:                       f.pp,
						SubscriptionGroup:                 f.sg,
					},
					UpdateSubscriptionGroupProductPlanStaffServices(s),
				)
				_, err := uc.Execute(ctx, &pb.UpdateSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{Id: "edge-1", Role: "primary"},
				})
				return err
			},
		},
		{
			name:       "Delete",
			permission: permEdgeDelete,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewDeleteSubscriptionGroupProductPlanStaffUseCase(
					DeleteSubscriptionGroupProductPlanStaffRepositories{SubscriptionGroupProductPlanStaff: f.repo},
					DeleteSubscriptionGroupProductPlanStaffServices(s),
				)
				_, err := uc.Execute(ctx, &pb.DeleteSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{Id: "edge-1"},
				})
				return err
			},
		},
		{
			name:       "List",
			permission: permEdgeList,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewListSubscriptionGroupProductPlanStaffsUseCase(
					ListSubscriptionGroupProductPlanStaffsRepositories{SubscriptionGroupProductPlanStaff: f.repo},
					ListSubscriptionGroupProductPlanStaffsServices(s),
				)
				_, err := uc.Execute(ctx, &pb.ListSubscriptionGroupProductPlanStaffsRequest{})
				return err
			},
		},
		{
			// The Teachers-tab row/count sub-gate reaches this use case; it must
			// demand the LIST permission, not read.
			name:       "GetListPageData",
			permission: permEdgeList,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewGetSubscriptionGroupProductPlanStaffListPageDataUseCase(
					GetSubscriptionGroupProductPlanStaffListPageDataRepositories{SubscriptionGroupProductPlanStaff: f.repo},
					GetSubscriptionGroupProductPlanStaffListPageDataServices(s),
				)
				_, err := uc.Execute(ctx, &pb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{})
				return err
			},
		},
		{
			name:       "GetItemPageData",
			permission: permEdgeRead,
			exec: func(ctx context.Context, f *authzEdgeFixture) error {
				s := f.services()
				uc := NewGetSubscriptionGroupProductPlanStaffItemPageDataUseCase(
					GetSubscriptionGroupProductPlanStaffItemPageDataRepositories{SubscriptionGroupProductPlanStaff: f.repo},
					GetSubscriptionGroupProductPlanStaffItemPageDataServices(s),
				)
				_, err := uc.Execute(ctx, &pb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest{})
				return err
			},
		},
	}
}

// ----- tests ------------------------------------------------------------------

// TestEdgeUseCases_MissingPermission_DeniedAndRepoUntouched is the deny
// direction. The repository must record ZERO calls: a gate that ran after the
// read/write would leak assignment rows or mutate them before erroring.
func TestEdgeUseCases_MissingPermission_DeniedAndRepoUntouched(t *testing.T) {
	for _, tc := range edgeAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := newAuthzEdgeFixture(edgeAllowAllExcept(tc.permission))

			err := tc.exec(edgeAuthzCtx(), f)
			if err == nil {
				t.Fatalf("%s must be denied for a principal without %q", tc.name, tc.permission)
			}
			// NoOpTranslator returns the default verbatim; actiongate's deny
			// branch uses "Permission denied". Asserting the exact branch
			// matters: a missing user id, a nil gatekeeper and a nil authorizer
			// all error too, but say "Authorization failed"/"not configured".
			if !strings.Contains(err.Error(), "Permission denied") {
				t.Errorf("%s: want the permission-denied branch of the action gate, got %q", tc.name, err.Error())
			}
			if got := f.repo.totalCalls(); got != 0 {
				t.Errorf("%s: a denied call must never reach the repository; got %d calls (create=%d read=%d update=%d delete=%d list=%d listPage=%d itemPage=%d)",
					tc.name, got, f.repo.createCalls, f.repo.readCalls, f.repo.updateCalls, f.repo.deleteCalls, f.repo.listCalls, f.repo.listPageCalls, f.repo.itemPageCalls)
			}
			if len(f.authz.asked) != 1 || f.authz.asked[0] != tc.permission {
				t.Errorf("%s: the gate must consult exactly %q; it asked for %v", tc.name, tc.permission, f.authz.asked)
			}
		})
	}
}

// TestEdgeUseCases_ExactPermission_Allowed is the allow direction, and it is what
// makes the deny direction meaningful: the principal holds ONLY the one
// permission under test. A use case asking for any other code — a typo, a wrong
// action, a renamed entityid — is denied here and never reaches the repository.
func TestEdgeUseCases_ExactPermission_Allowed(t *testing.T) {
	for _, tc := range edgeAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := newAuthzEdgeFixture(edgeGrantOnly(tc.permission))

			if err := tc.exec(edgeAuthzCtx(), f); err != nil {
				t.Fatalf("%s: a principal holding exactly %q must pass the gate on an eligible edge; got %v", tc.name, tc.permission, err)
			}
			if got := f.repo.totalCalls(); got == 0 {
				t.Errorf("%s: an authorized call must reach the repository; got 0 calls", tc.name)
			}
			if len(f.authz.asked) != 1 || f.authz.asked[0] != tc.permission {
				t.Errorf("%s: the gate must consult exactly %q; it asked for %v", tc.name, tc.permission, f.authz.asked)
			}
		})
	}
}

// TestEdgeUseCases_NoPrincipal_Denied covers the OTHER fail-closed branch of
// ActionGatekeeper.Check: enforcement on, but no user id in context (an
// identity-less request reaching a use case). The gate must refuse before
// HasPermission is consulted and without touching the repository — otherwise a
// context-plumbing regression that dropped the principal would be
// indistinguishable from a permission problem.
func TestEdgeUseCases_NoPrincipal_Denied(t *testing.T) {
	for _, tc := range edgeAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Holds everything: only the missing principal can deny here.
			f := newAuthzEdgeFixture(&edgeAuthorizer{holds: func(string) bool { return true }})

			ctx := appcontext.WithWorkspaceID(context.Background(), "ws-1")
			err := tc.exec(ctx, f)
			if err == nil {
				t.Fatalf("%s must be denied when the context carries no principal", tc.name)
			}
			if !strings.Contains(err.Error(), "Authorization failed") {
				t.Errorf("%s: want the missing-principal branch (\"Authorization failed\"), got %q", tc.name, err.Error())
			}
			if got := f.repo.totalCalls(); got != 0 {
				t.Errorf("%s: a principal-less call must never reach the repository; got %d calls", tc.name, got)
			}
			if len(f.authz.asked) != 0 {
				t.Errorf("%s: the gate must not consult the authorizer without a principal; it asked for %v", tc.name, f.authz.asked)
			}
		})
	}
}

// ----- Assign: the upsert must not be a gate bypass ---------------------------
//
// AssignSubscriptionGroupProductPlanStaffUseCase.Execute has NO gate of its own —
// by design it delegates to the Create/Update/Delete use cases, "which
// fail-closed check [their] own action permission" (its doc comment). That is an
// assertion about behavior with nothing asserting it: if a future refactor
// inlined the writes, or constructed the sub-use-cases with a different
// gatekeeper, the drawer's Assign/Change/Clear actions would become unguarded
// and every existing Assign test (all NoOpAuthorizer) would stay green.
//
// Each case below drives ONE branch of the upsert with the corresponding
// permission withheld, and asserts the branch's write never happened. The
// active-edge LIST that precedes the branch is expected — it is the lookup that
// selects the branch — so these assert the specific write counter, not the total.

func (f *authzEdgeFixture) assignUC() *AssignSubscriptionGroupProductPlanStaffUseCase {
	s := f.services()
	return NewAssignSubscriptionGroupProductPlanStaffUseCase(
		AssignSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: f.repo,
			ProductPlanStaff:                  f.pps,
			ProductPlan:                       f.pp,
			SubscriptionGroup:                 f.sg,
		},
		AssignSubscriptionGroupProductPlanStaffServices(s),
	)
}

func assignAuthzReq(staffID string) *AssignSubscriptionGroupProductPlanStaffRequest {
	return &AssignSubscriptionGroupProductPlanStaffRequest{
		WorkspaceID:         "ws-1",
		SubscriptionGroupID: "sg-1",
		ProductPlanID:       "pp-1",
		StaffID:             staffID,
		Role:                "primary",
	}
}

// TestAssign_CreateBranch_MissingCreatePermission_Denied: no prior edge, so the
// upsert falls through to the create branch. Withholding sgpps:create must stop
// it.
func TestAssign_CreateBranch_MissingCreatePermission_Denied(t *testing.T) {
	f := newAuthzEdgeFixture(edgeAllowAllExcept(permEdgeCreate))
	f.repo.rows = nil // no active and no inactive edge for (sg-1, pp-1)

	_, err := f.assignUC().Execute(edgeAuthzCtx(), assignAuthzReq("staff-1"))
	if err == nil {
		t.Fatalf("Assign's create branch must inherit the sgpps:create gate")
	}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("want the permission-denied branch, got %q", err.Error())
	}
	if f.repo.createCalls != 0 {
		t.Errorf("denied Assign must not create an edge; got %d create calls", f.repo.createCalls)
	}
	if len(f.authz.asked) != 1 || f.authz.asked[0] != permEdgeCreate {
		t.Errorf("the create branch must consult exactly %q; it asked for %v", permEdgeCreate, f.authz.asked)
	}
}

// TestAssign_UpdateBranch_MissingUpdatePermission_Denied: an active edge exists
// for a DIFFERENT teacher, so assigning staff-2 takes the change-the-servicer
// branch. Withholding sgpps:update must stop it.
func TestAssign_UpdateBranch_MissingUpdatePermission_Denied(t *testing.T) {
	f := newAuthzEdgeFixture(edgeAllowAllExcept(permEdgeUpdate))
	f.repo.rows = []*pb.SubscriptionGroupProductPlanStaff{f.repo.row}

	_, err := f.assignUC().Execute(edgeAuthzCtx(), assignAuthzReq("staff-2"))
	if err == nil {
		t.Fatalf("Assign's change-servicer branch must inherit the sgpps:update gate")
	}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("want the permission-denied branch, got %q", err.Error())
	}
	if f.repo.updateCalls != 0 {
		t.Errorf("denied Assign must not update the edge; got %d update calls", f.repo.updateCalls)
	}
	if len(f.authz.asked) != 1 || f.authz.asked[0] != permEdgeUpdate {
		t.Errorf("the change-servicer branch must consult exactly %q; it asked for %v", permEdgeUpdate, f.authz.asked)
	}
}

// TestAssign_ClearBranch_MissingDeletePermission_Denied: an empty StaffID is the
// drawer's inline Clear, which soft-deactivates the active edge. Withholding
// sgpps:delete must stop it — otherwise a read-only principal could unassign
// every teacher on a class.
func TestAssign_ClearBranch_MissingDeletePermission_Denied(t *testing.T) {
	f := newAuthzEdgeFixture(edgeAllowAllExcept(permEdgeDelete))
	f.repo.rows = []*pb.SubscriptionGroupProductPlanStaff{f.repo.row}

	_, err := f.assignUC().Execute(edgeAuthzCtx(), assignAuthzReq(""))
	if err == nil {
		t.Fatalf("Assign's clear branch must inherit the sgpps:delete gate")
	}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("want the permission-denied branch, got %q", err.Error())
	}
	if f.repo.deleteCalls != 0 {
		t.Errorf("denied Assign must not clear the edge; got %d delete calls", f.repo.deleteCalls)
	}
	if len(f.authz.asked) != 1 || f.authz.asked[0] != permEdgeDelete {
		t.Errorf("the clear branch must consult exactly %q; it asked for %v", permEdgeDelete, f.authz.asked)
	}
}
