package subscription_group_product_plan

// Deny-direction authorization tests for the class (sgpp) use cases — the
// espyna half of coverage-audit gap A-G3.
//
// Every other test in this package builds its ActionGatekeeper with
// ports.NewNoOpAuthorizer(), whose IsEnabled() == false makes
// ActionGatekeeper.Check short-circuit to nil before it ever computes a
// permission code. So no test in this tree has ever exercised a real denial,
// and neither a permission_code typo nor a renamed entityid would fail
// anything here. These tests wire an authorizer with IsEnabled() == true
// (the enforce-style shape of
// subscription/create_subscription_materialize_authz_test.go's splitAuthorizer)
// and assert both directions for all five use cases:
//
//	deny  — the principal lacks EXACTLY the class permission for that action:
//	        Execute must error with the permission-denied message AND the
//	        repository must not be touched at all.
//	allow — the principal holds ONLY that one permission and nothing else:
//	        Execute must reach the repository.
//
// Run together the two directions pin the permission STRING, not just the
// presence of a gate: a wrong entity or action makes the deny case pass the
// gate (allow-all-except grants the string actually asked for) and the allow
// case fail it (grant-only denies everything but the expected string).
//
// The expected codes below are deliberately written as LITERALS rather than
// derived from entityid.EntityPermission(...). They are the contract with the
// seeded permission.code rows in the database; deriving them from the same
// constants the code under test uses would be circular and would let a rename
// slip through green.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

const (
	permClassCreate = "subscription_group_product_plan:create"
	permClassRead   = "subscription_group_product_plan:read"
	permClassUpdate = "subscription_group_product_plan:update"
	permClassDelete = "subscription_group_product_plan:delete"
	permClassList   = "subscription_group_product_plan:list"
)

// ----- authorizer ---------------------------------------------------------

// recordingAuthorizer is an actiongate.Authorizer whose IsEnabled() is true, so
// Check actually resolves a permission code and consults holds(). It records
// every code it was asked about.
type recordingAuthorizer struct {
	holds func(permission string) bool
	asked []string
}

func (a *recordingAuthorizer) IsEnabled() bool { return true }

func (a *recordingAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	return a.holds(permission), nil
}

// allowAllExcept models a principal holding every permission in the system
// EXCEPT the one under test — so a use case asking for the wrong code sails
// through, which is exactly the failure this suite must catch.
func allowAllExcept(code string) *recordingAuthorizer {
	return &recordingAuthorizer{holds: func(p string) bool { return p != code }}
}

// grantOnly models a least-privilege principal holding ONLY the code under test.
func grantOnly(code string) *recordingAuthorizer {
	return &recordingAuthorizer{holds: func(p string) bool { return p == code }}
}

// ----- repository ---------------------------------------------------------

// authzClassRepo counts every domain-service method the five use cases can
// reach, so "the repository was never called" is assertable as a single total
// rather than per-method (Update, for instance, reads and lists before writing).
type authzClassRepo struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	createCalls int
	readCalls   int
	updateCalls int
	deleteCalls int
	listCalls   int
	row         *pb.SubscriptionGroupProductPlan
}

func (m *authzClassRepo) totalCalls() int {
	return m.createCalls + m.readCalls + m.updateCalls + m.deleteCalls + m.listCalls
}

func (m *authzClassRepo) CreateSubscriptionGroupProductPlan(_ context.Context, req *pb.CreateSubscriptionGroupProductPlanRequest) (*pb.CreateSubscriptionGroupProductPlanResponse, error) {
	m.createCalls++
	return &pb.CreateSubscriptionGroupProductPlanResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{req.GetData()}}, nil
}

func (m *authzClassRepo) ReadSubscriptionGroupProductPlan(_ context.Context, _ *pb.ReadSubscriptionGroupProductPlanRequest) (*pb.ReadSubscriptionGroupProductPlanResponse, error) {
	m.readCalls++
	if m.row == nil {
		return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true}, nil
	}
	return &pb.ReadSubscriptionGroupProductPlanResponse{Success: true, Data: []*pb.SubscriptionGroupProductPlan{m.row}}, nil
}

func (m *authzClassRepo) UpdateSubscriptionGroupProductPlan(_ context.Context, _ *pb.UpdateSubscriptionGroupProductPlanRequest) (*pb.UpdateSubscriptionGroupProductPlanResponse, error) {
	m.updateCalls++
	return &pb.UpdateSubscriptionGroupProductPlanResponse{Success: true}, nil
}

func (m *authzClassRepo) DeleteSubscriptionGroupProductPlan(_ context.Context, _ *pb.DeleteSubscriptionGroupProductPlanRequest) (*pb.DeleteSubscriptionGroupProductPlanResponse, error) {
	m.deleteCalls++
	return &pb.DeleteSubscriptionGroupProductPlanResponse{Success: true}, nil
}

func (m *authzClassRepo) ListSubscriptionGroupProductPlans(_ context.Context, _ *pb.ListSubscriptionGroupProductPlansRequest) (*pb.ListSubscriptionGroupProductPlansResponse, error) {
	m.listCalls++
	return &pb.ListSubscriptionGroupProductPlansResponse{Success: true}, nil
}

// ----- fixture ------------------------------------------------------------

// authzClassFixture wires one authorizer + one counting repo into all five use
// cases. The cross-domain repos are populated with a fully VALID class graph
// (sg-1 / pp-1 / tmpl-1 in ws-1), so nothing downstream of the gate can reject
// the request — any error the allow direction sees would therefore be a real
// finding, not fixture noise.
type authzClassFixture struct {
	repo     *authzClassRepo
	authz    *recordingAuthorizer
	plan     *mockPlanRepo
	group    *mockGroupRepo
	template *mockJobTemplateRepo
}

func newAuthzClassFixture(a *recordingAuthorizer) *authzClassFixture {
	return &authzClassFixture{
		authz: a,
		repo: &authzClassRepo{row: &pb.SubscriptionGroupProductPlan{
			Id:                  "class-1",
			Active:              true,
			WorkspaceId:         "ws-1",
			SubscriptionGroupId: "sg-1",
			ProductPlanId:       "pp-1",
			JobTemplateId:       "tmpl-1",
			Status:              pb.SubscriptionGroupProductPlanStatus_SUBSCRIPTION_GROUP_PRODUCT_PLAN_STATUS_ACTIVE,
		}},
		plan:     &mockPlanRepo{byID: map[string]*productplanpb.ProductPlan{"pp-1": classPlan("pp-1", "plan-1", "product-1")}},
		group:    &mockGroupRepo{group: classGroup("plan-1", "ws-1")},
		template: &mockJobTemplateRepo{byID: map[string]*jobtemplatepb.JobTemplate{"tmpl-1": classTemplate("tmpl-1", "product-1")}},
	}
}

// gate is the ONLY consumer of the recording authorizer. Services.Authorizer is
// left as the no-op because these use cases never call it directly — the
// ActionGatekeeper is the whole authorization surface.
func (f *authzClassFixture) gate() *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(f.authz, ports.NewNoOpTranslator())
}

// classAuthzCtx carries both a user id (ActionGatekeeper.Check fails on a
// DIFFERENT "Authorization failed" path without one, which would make the deny
// assertion ambiguous) and the workspace the fixture graph lives in.
func classAuthzCtx() context.Context {
	return appcontext.WithWorkspaceID(appcontext.WithUserID(context.Background(), "teacher-1"), "ws-1")
}

type classAuthzCase struct {
	name       string
	permission string
	exec       func(ctx context.Context, f *authzClassFixture) error
}

func classAuthzCases() []classAuthzCase {
	return []classAuthzCase{
		{
			name:       "Create",
			permission: permClassCreate,
			exec: func(ctx context.Context, f *authzClassFixture) error {
				uc := NewCreateSubscriptionGroupProductPlanUseCase(
					CreateSubscriptionGroupProductPlanRepositories{
						SubscriptionGroupProductPlan: f.repo,
						ProductPlan:                  f.plan,
						SubscriptionGroup:            f.group,
						JobTemplate:                  f.template,
					},
					CreateSubscriptionGroupProductPlanServices{
						Authorizer:       ports.NewNoOpAuthorizer(),
						Translator:       ports.NewNoOpTranslator(),
						IDGenerator:      stubIDClass{},
						ActionGatekeeper: f.gate(),
					},
				)
				_, err := uc.Execute(ctx, classReq("sg-1", "pp-1", "tmpl-1"))
				return err
			},
		},
		{
			name:       "Read",
			permission: permClassRead,
			exec: func(ctx context.Context, f *authzClassFixture) error {
				uc := NewReadSubscriptionGroupProductPlanUseCase(
					ReadSubscriptionGroupProductPlanRepositories{SubscriptionGroupProductPlan: f.repo},
					ReadSubscriptionGroupProductPlanServices{
						Authorizer:       ports.NewNoOpAuthorizer(),
						Translator:       ports.NewNoOpTranslator(),
						ActionGatekeeper: f.gate(),
					},
				)
				_, err := uc.Execute(ctx, &pb.ReadSubscriptionGroupProductPlanRequest{
					Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
				})
				return err
			},
		},
		{
			name:       "Update",
			permission: permClassUpdate,
			exec: func(ctx context.Context, f *authzClassFixture) error {
				uc := NewUpdateSubscriptionGroupProductPlanUseCase(
					UpdateSubscriptionGroupProductPlanRepositories{
						SubscriptionGroupProductPlan: f.repo,
						ProductPlan:                  f.plan,
						SubscriptionGroup:            f.group,
						JobTemplate:                  f.template,
					},
					UpdateSubscriptionGroupProductPlanServices{
						Authorizer:       ports.NewNoOpAuthorizer(),
						Translator:       ports.NewNoOpTranslator(),
						ActionGatekeeper: f.gate(),
					},
				)
				_, err := uc.Execute(ctx, &pb.UpdateSubscriptionGroupProductPlanRequest{
					Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
				})
				return err
			},
		},
		{
			name:       "Delete",
			permission: permClassDelete,
			exec: func(ctx context.Context, f *authzClassFixture) error {
				uc := NewDeleteSubscriptionGroupProductPlanUseCase(
					DeleteSubscriptionGroupProductPlanRepositories{SubscriptionGroupProductPlan: f.repo},
					DeleteSubscriptionGroupProductPlanServices{
						Authorizer:       ports.NewNoOpAuthorizer(),
						Translator:       ports.NewNoOpTranslator(),
						ActionGatekeeper: f.gate(),
						// nil ReferenceChecker: the in-use guard degrades to
						// un-guarded (see in_use_guard.go), so the ONLY thing
						// standing between the caller and the delete is the gate.
					},
				)
				_, err := uc.Execute(ctx, &pb.DeleteSubscriptionGroupProductPlanRequest{
					Data: &pb.SubscriptionGroupProductPlan{Id: "class-1"},
				})
				return err
			},
		},
		{
			name:       "List",
			permission: permClassList,
			exec: func(ctx context.Context, f *authzClassFixture) error {
				uc := NewListSubscriptionGroupProductPlansUseCase(
					ListSubscriptionGroupProductPlansRepositories{SubscriptionGroupProductPlan: f.repo},
					ListSubscriptionGroupProductPlansServices{
						Authorizer:       ports.NewNoOpAuthorizer(),
						Translator:       ports.NewNoOpTranslator(),
						ActionGatekeeper: f.gate(),
					},
				)
				_, err := uc.Execute(ctx, &pb.ListSubscriptionGroupProductPlansRequest{})
				return err
			},
		},
	}
}

// ----- tests --------------------------------------------------------------

// TestClassUseCases_MissingPermission_DeniedAndRepoUntouched is the deny
// direction: the principal holds every permission in the system EXCEPT the one
// class permission the use case must demand. Execute has to fail with the
// permission-denied message, and the repository must record ZERO calls — a gate
// that ran after the read/write would leak data or mutate rows before erroring.
func TestClassUseCases_MissingPermission_DeniedAndRepoUntouched(t *testing.T) {
	for _, tc := range classAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := newAuthzClassFixture(allowAllExcept(tc.permission))

			err := tc.exec(classAuthzCtx(), f)
			if err == nil {
				t.Fatalf("%s must be denied for a principal without %q", tc.name, tc.permission)
			}
			// NoOpTranslator returns the default verbatim; actiongate's deny
			// branch uses "Permission denied". Asserting the exact branch
			// matters: a missing user id, a nil gatekeeper and a nil authorizer
			// all also error, but say "Authorization failed" / "not configured".
			if !strings.Contains(err.Error(), "Permission denied") {
				t.Errorf("%s: want the permission-denied branch of the action gate, got %q", tc.name, err.Error())
			}
			if got := f.repo.totalCalls(); got != 0 {
				t.Errorf("%s: a denied call must never reach the repository; got %d calls (create=%d read=%d update=%d delete=%d list=%d)",
					tc.name, got, f.repo.createCalls, f.repo.readCalls, f.repo.updateCalls, f.repo.deleteCalls, f.repo.listCalls)
			}
			if len(f.authz.asked) != 1 || f.authz.asked[0] != tc.permission {
				t.Errorf("%s: the gate must consult exactly %q; it asked for %v", tc.name, tc.permission, f.authz.asked)
			}
		})
	}
}

// TestClassUseCases_ExactPermission_Allowed is the allow direction, and it is
// what makes the deny direction meaningful: the principal holds ONLY the one
// class permission under test. If a use case asked for any other code — a typo,
// a wrong action, a renamed entityid — this authorizer denies it and the call
// never reaches the repository.
func TestClassUseCases_ExactPermission_Allowed(t *testing.T) {
	for _, tc := range classAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := newAuthzClassFixture(grantOnly(tc.permission))

			if err := tc.exec(classAuthzCtx(), f); err != nil {
				t.Fatalf("%s: a principal holding exactly %q must pass the gate on a valid class graph; got %v", tc.name, tc.permission, err)
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

// TestClassUseCases_NoPrincipal_Denied covers the second fail-closed branch of
// ActionGatekeeper.Check: with enforcement ON but no user id in context (an
// unauthenticated or identity-less request reaching a use case), the gate must
// refuse before HasPermission is ever consulted, and again without touching the
// repository. Without this, a context-plumbing regression that dropped the user
// id would be indistinguishable from a permission problem.
func TestClassUseCases_NoPrincipal_Denied(t *testing.T) {
	for _, tc := range classAuthzCases() {
		t.Run(tc.name, func(t *testing.T) {
			// Holds everything: only the missing principal can deny here.
			f := newAuthzClassFixture(&recordingAuthorizer{holds: func(string) bool { return true }})

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
