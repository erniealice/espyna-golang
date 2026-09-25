package rating_description_set_entry

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

// W3 follow-up finding (2026-09-25, codex-review-impl1.out.md): "entry
// permissions use the wrong resource ... require
// rating_description_set_entry:*. The agreed twelve permissions grant parent
// -set permissions instead. An Education Admin with the prescribed grants
// cannot author entries." These tests prove every entry use case checks the
// PARENT set's permission code (rating_description_set:update for
// create/update/delete, rating_description_set:read for read/list) and NEVER
// rating_description_set_entry:* — the clone has zero of those grants, so an
// authorized caller would otherwise always be denied.

type entryAuthzTestAuthorizer struct {
	enabled bool
	perms   map[string]bool
	asked   []string
}

func (a *entryAuthzTestAuthorizer) IsEnabled() bool { return a.enabled }

func (a *entryAuthzTestAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	if a.perms == nil {
		return false, nil
	}
	return a.perms[permission], nil
}

func newEntryAuthzContext() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

// allowOnly grants exactly the given permission string and nothing else —
// including no rating_description_set_entry:* code — so a use case checking
// the WRONG resource is denied (proving the fix; a pre-fix use case checking
// rating_description_set_entry:* would fail every one of these tests).
func allowOnlyEntryPermission(perm string) *entryAuthzTestAuthorizer {
	return &entryAuthzTestAuthorizer{enabled: true, perms: map[string]bool{perm: true}}
}

type entryAuthzTestTransactor struct {
	executeCalls int
}

func (t *entryAuthzTestTransactor) SupportsTransactions() bool               { return true }
func (t *entryAuthzTestTransactor) IsTransactionActive(context.Context) bool { return false }
func (t *entryAuthzTestTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.executeCalls++
	return fn(ctx)
}

// fakeEntryCRUDRepo implements pb.RatingDescriptionSetEntryDomainServiceServer
// with trivial successes, purely to let Execute reach (and prove) the
// authorization check without needing a real adapter.
type fakeEntryCRUDRepo struct {
	pb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
}

func (r *fakeEntryCRUDRepo) CreateRatingDescriptionSetEntry(_ context.Context, req *pb.CreateRatingDescriptionSetEntryRequest) (*pb.CreateRatingDescriptionSetEntryResponse, error) {
	return &pb.CreateRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{req.Data}, Success: true}, nil
}

func (r *fakeEntryCRUDRepo) ReadRatingDescriptionSetEntry(_ context.Context, req *pb.ReadRatingDescriptionSetEntryRequest) (*pb.ReadRatingDescriptionSetEntryResponse, error) {
	return &pb.ReadRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{{Id: "entry-1"}}, Success: true}, nil
}

func (r *fakeEntryCRUDRepo) UpdateRatingDescriptionSetEntry(_ context.Context, req *pb.UpdateRatingDescriptionSetEntryRequest) (*pb.UpdateRatingDescriptionSetEntryResponse, error) {
	return &pb.UpdateRatingDescriptionSetEntryResponse{Data: []*pb.RatingDescriptionSetEntry{req.Data}, Success: true}, nil
}

func (r *fakeEntryCRUDRepo) DeleteRatingDescriptionSetEntry(_ context.Context, _ *pb.DeleteRatingDescriptionSetEntryRequest) (*pb.DeleteRatingDescriptionSetEntryResponse, error) {
	return &pb.DeleteRatingDescriptionSetEntryResponse{Success: true}, nil
}

func (r *fakeEntryCRUDRepo) ListRatingDescriptionSetEntries(_ context.Context, _ *pb.ListRatingDescriptionSetEntriesRequest) (*pb.ListRatingDescriptionSetEntriesResponse, error) {
	return &pb.ListRatingDescriptionSetEntriesResponse{Data: nil, Success: true}, nil
}

var (
	entryCreateUpdateDeletePermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionUpdate)
	entryReadListPermission           = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionRead)
	// wrongEntryResourcePermission is the WRONG resource the pre-fix code
	// checked — granting only this must still deny every use case.
	wrongEntryResourcePermission = entityid.EntityPermission(entityid.RatingDescriptionSetEntry, entityid.ActionCreate)
)

func TestCreateRatingDescriptionSetEntry_AuthorizesAgainstParentSetUpdate(t *testing.T) {
	authz := allowOnlyEntryPermission(entryCreateUpdateDeletePermission)
	tx := &entryAuthzTestTransactor{}
	uc := NewCreateRatingDescriptionSetEntryUseCase(
		CreateRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		CreateRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(),
			IDGenerator: nil, ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.CreateRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{
		RatingDescriptionSetId: "set-1", OutcomeCriteriaId: "crit-1", ScoreScaleBandId: "band-1", Description: "text",
	}})
	if err != nil {
		t.Fatalf("expected success when only %q is granted, got %v", entryCreateUpdateDeletePermission, err)
	}
	if len(authz.asked) != 1 || authz.asked[0] != entryCreateUpdateDeletePermission {
		t.Fatalf("expected exactly one check for %q, got %v", entryCreateUpdateDeletePermission, authz.asked)
	}
}

func TestCreateRatingDescriptionSetEntry_WrongResourceGrantStillDenies(t *testing.T) {
	authz := allowOnlyEntryPermission(wrongEntryResourcePermission)
	tx := &entryAuthzTestTransactor{}
	uc := NewCreateRatingDescriptionSetEntryUseCase(
		CreateRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		CreateRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.CreateRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{
		RatingDescriptionSetId: "set-1", OutcomeCriteriaId: "crit-1", ScoreScaleBandId: "band-1", Description: "text",
	}})
	if err == nil {
		t.Fatal("expected denial: granting rating_description_set_entry:create must not authorize entry creation")
	}
	if tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede any transaction, got %d", tx.executeCalls)
	}
}

func TestUpdateRatingDescriptionSetEntry_AuthorizesAgainstParentSetUpdate(t *testing.T) {
	authz := allowOnlyEntryPermission(entryCreateUpdateDeletePermission)
	tx := &entryAuthzTestTransactor{}
	uc := NewUpdateRatingDescriptionSetEntryUseCase(
		UpdateRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		UpdateRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.UpdateRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{Id: "entry-1", Description: "text"}})
	if err != nil {
		t.Fatalf("expected success when only %q is granted, got %v", entryCreateUpdateDeletePermission, err)
	}
}

func TestDeleteRatingDescriptionSetEntry_AuthorizesAgainstParentSetUpdate(t *testing.T) {
	authz := allowOnlyEntryPermission(entryCreateUpdateDeletePermission)
	tx := &entryAuthzTestTransactor{}
	uc := NewDeleteRatingDescriptionSetEntryUseCase(
		DeleteRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		DeleteRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.DeleteRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{Id: "entry-1"}})
	if err != nil {
		t.Fatalf("expected success when only %q is granted, got %v", entryCreateUpdateDeletePermission, err)
	}
}

func TestReadRatingDescriptionSetEntry_AuthorizesAgainstParentSetRead(t *testing.T) {
	authz := allowOnlyEntryPermission(entryReadListPermission)
	uc := NewReadRatingDescriptionSetEntryUseCase(
		ReadRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		ReadRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.ReadRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{Id: "entry-1"}})
	if err != nil {
		t.Fatalf("expected success when only %q is granted, got %v", entryReadListPermission, err)
	}
	if len(authz.asked) != 1 || authz.asked[0] != entryReadListPermission {
		t.Fatalf("expected exactly one check for %q, got %v", entryReadListPermission, authz.asked)
	}
}

func TestReadRatingDescriptionSetEntry_UpdateGrantAloneDenies(t *testing.T) {
	// Read must not accept the write permission either — it is scoped to
	// rating_description_set:read specifically.
	authz := allowOnlyEntryPermission(entryCreateUpdateDeletePermission)
	uc := NewReadRatingDescriptionSetEntryUseCase(
		ReadRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		ReadRatingDescriptionSetEntryServices{
			Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.ReadRatingDescriptionSetEntryRequest{Data: &pb.RatingDescriptionSetEntry{Id: "entry-1"}})
	if err == nil {
		t.Fatal("expected denial: rating_description_set:update alone must not authorize a read")
	}
}

func TestListRatingDescriptionSetEntries_AuthorizesAgainstParentSetRead(t *testing.T) {
	authz := allowOnlyEntryPermission(entryReadListPermission)
	uc := NewListRatingDescriptionSetEntriesUseCase(
		ListRatingDescriptionSetEntriesRepositories{RatingDescriptionSetEntry: &fakeEntryCRUDRepo{}},
		ListRatingDescriptionSetEntriesServices{
			Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		})

	_, err := uc.Execute(newEntryAuthzContext(), &pb.ListRatingDescriptionSetEntriesRequest{})
	if err != nil {
		t.Fatalf("expected success when only %q is granted, got %v", entryReadListPermission, err)
	}
}
