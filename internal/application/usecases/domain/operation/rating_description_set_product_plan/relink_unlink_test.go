package rating_description_set_product_plan

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type relinkTestAuthorizer struct {
	enabled bool
	perms   map[string]bool
	asked   []string
}

func (a *relinkTestAuthorizer) IsEnabled() bool { return a.enabled }

func (a *relinkTestAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	if a.perms == nil {
		return false, nil
	}
	return a.perms[permission], nil
}

func newRelinkContext() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func relinkServices(authz *relinkTestAuthorizer, tx ports.Transactor, idGen ports.IDGenerator) RelinkRatingDescriptionSetProductPlanServices {
	return RelinkRatingDescriptionSetProductPlanServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		IDGenerator:      idGen,
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func unlinkServices(authz *relinkTestAuthorizer, tx ports.Transactor) UnlinkRatingDescriptionSetProductPlanServices {
	return UnlinkRatingDescriptionSetProductPlanServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

type relinkTransactor struct {
	supports     bool
	executeCalls int
	committed    bool
	rolledBack   bool
}

func (t *relinkTransactor) SupportsTransactions() bool               { return t.supports }
func (t *relinkTransactor) IsTransactionActive(context.Context) bool { return false }
func (t *relinkTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.executeCalls++
	if err := fn(ctx); err != nil {
		t.rolledBack = true
		return err
	}
	t.committed = true
	return nil
}

type stubIDGenerator struct{ id string }

func (g *stubIDGenerator) GenerateID() string                        { return g.id }
func (g *stubIDGenerator) GenerateIDWithPrefix(prefix string) string { return prefix + g.id }
func (g *stubIDGenerator) IsEnabled() bool                           { return true }
func (g *stubIDGenerator) GetProviderInfo() string                   { return "stub" }

// fakeLinkRepo implements pb.RatingDescriptionSetProductPlanDomainServiceServer
// plus RelinkLocked/UnlinkLocked (domainports.RatingDescriptionSetProductPlanRelinker).
type fakeLinkRepo struct {
	pb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer
	relinkCalls int
	unlinkCalls int
	lastNewLink *pb.RatingDescriptionSetProductPlan
	lastExpect  *string
	lastReason  string
	relinkErr   error
	unlinkErr   error
	result      *pb.RatingDescriptionSetProductPlan
}

func (r *fakeLinkRepo) RelinkLocked(_ context.Context, newLink *pb.RatingDescriptionSetProductPlan, expected *string, reason string) (*pb.RatingDescriptionSetProductPlan, error) {
	r.relinkCalls++
	r.lastNewLink = newLink
	r.lastExpect = expected
	r.lastReason = reason
	if r.relinkErr != nil {
		return nil, r.relinkErr
	}
	return r.result, nil
}

func (r *fakeLinkRepo) UnlinkLocked(_ context.Context, _ string, reason string) (*pb.RatingDescriptionSetProductPlan, error) {
	r.unlinkCalls++
	r.lastReason = reason
	if r.unlinkErr != nil {
		return nil, r.unlinkErr
	}
	return r.result, nil
}

type fakeLinkRepoNoRelinker struct {
	pb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer
}

var (
	relinkCreatePermission = entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionCreate)
	relinkUpdatePermission = entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionUpdate)
	unlinkPermission       = entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionDelete)
)

func relinkAllowAuthorizer(perms ...string) *relinkTestAuthorizer {
	m := map[string]bool{}
	for _, p := range perms {
		m[p] = true
	}
	return &relinkTestAuthorizer{enabled: true, perms: m}
}

func relinkDenyAuthorizer(perm string) *relinkTestAuthorizer {
	return &relinkTestAuthorizer{enabled: true, perms: map[string]bool{perm: false}}
}

// --- Relink --------------------------------------------------------------

func TestRelinkRatingDescriptionSetProductPlan_FirstLinkChecksCreatePermission(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-1"}}
	tx := &relinkTransactor{supports: true}
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkDenyAuthorizer(relinkCreatePermission), tx, &stubIDGenerator{id: "link-1"}))

	_, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-1",
	})
	if err == nil {
		t.Fatal("expected permission denial for first-link (create)")
	}
	if repo.relinkCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: relink=%d tx=%d", repo.relinkCalls, tx.executeCalls)
	}
}

func TestRelinkRatingDescriptionSetProductPlan_ReplacementChecksUpdatePermission(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-2"}}
	tx := &relinkTransactor{supports: true}
	expected := "link-1"
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkDenyAuthorizer(relinkUpdatePermission), tx, &stubIDGenerator{id: "link-2"}))

	_, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-2", ExpectedCurrentLinkId: &expected,
	})
	if err == nil {
		t.Fatal("expected permission denial for replacement (update)")
	}
	if repo.relinkCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: relink=%d tx=%d", repo.relinkCalls, tx.executeCalls)
	}
}

func TestRelinkRatingDescriptionSetProductPlan_SuccessCommitsInsideTransactionWithGeneratedID(t *testing.T) {
	result := &pb.RatingDescriptionSetProductPlan{Id: "link-1", RatingDescriptionSetId: "set-1", Active: true}
	repo := &fakeLinkRepo{result: result}
	tx := &relinkTransactor{supports: true}
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkAllowAuthorizer(relinkCreatePermission), tx, &stubIDGenerator{id: "link-1"}))

	resp, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-1",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.relinkCalls != 1 || tx.executeCalls != 1 || !tx.committed || tx.rolledBack {
		t.Fatalf("expected exactly one committed relink call: calls=%d executeCalls=%d committed=%v rolledBack=%v",
			repo.relinkCalls, tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if repo.lastNewLink == nil || repo.lastNewLink.Id != "link-1" || repo.lastNewLink.ProductPlanId != "pp-1" {
		t.Fatalf("expected enriched newLink with generated id, got %v", repo.lastNewLink)
	}
	if repo.lastExpect != nil {
		t.Fatalf("expected nil expectedCurrentLinkID for first link, got %v", *repo.lastExpect)
	}
	if len(resp.Data) != 1 || resp.Data[0].Id != "link-1" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// TestRelinkRatingDescriptionSetProductPlan_ReasonReachesTheRepository proves
// the use case threads Request.Reason through to RelinkLocked, which is the
// only way the adapter's semantic audit event (writeLifecycleAudit,
// rating_description_set_product_plan.go) gets the caller's stated reason
// (schema-proposal.md §9.4, codex-review-impl2.out.md finding 9).
func TestRelinkRatingDescriptionSetProductPlan_ReasonReachesTheRepository(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-1"}}
	tx := &relinkTransactor{supports: true}
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkAllowAuthorizer(relinkCreatePermission), tx, &stubIDGenerator{id: "link-1"}))

	_, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-1", Reason: "AY rollover",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.lastReason != "AY rollover" {
		t.Fatalf("expected the request reason to reach RelinkLocked, got %q", repo.lastReason)
	}
}

func TestRelinkRatingDescriptionSetProductPlan_StaleLinkRollsBackAndReturnsNoResponse(t *testing.T) {
	sentinel := errors.New("STALE_LINK: expected current link \"link-0\" but found \"link-9\"")
	repo := &fakeLinkRepo{relinkErr: sentinel}
	tx := &relinkTransactor{supports: true}
	expected := "link-0"
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkAllowAuthorizer(relinkUpdatePermission), tx, &stubIDGenerator{id: "link-2"}))

	resp, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-2", ExpectedCurrentLinkId: &expected,
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got resp=%v err=%v", resp, err)
	}
	if resp != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("stale link must rollback with no response: resp=%v committed=%v rolledBack=%v", resp, tx.committed, tx.rolledBack)
	}
}

func TestRelinkRatingDescriptionSetProductPlan_RepositoryWithoutRelinkerCapabilityFailsClosed(t *testing.T) {
	repo := &fakeLinkRepoNoRelinker{}
	tx := &relinkTransactor{supports: true}
	uc := NewRelinkRatingDescriptionSetProductPlanUseCase(repo, relinkServices(relinkAllowAuthorizer(relinkCreatePermission), tx, &stubIDGenerator{id: "link-1"}))

	_, err := uc.Execute(newRelinkContext(), &pb.RelinkRatingDescriptionSetProductPlanRequest{
		ProductPlanId: "pp-1", PriceScheduleId: "ps-1", RatingDescriptionSetId: "set-1",
	})
	if err == nil {
		t.Fatal("expected repository-capability error")
	}
	if tx.executeCalls != 0 {
		t.Fatalf("capability precondition must not execute a transaction, got %d", tx.executeCalls)
	}
}

// --- Unlink ----------------------------------------------------------------

func TestUnlinkRatingDescriptionSetProductPlan_MissingPermissionDeniesBeforeAnyCall(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-1", Active: false}}
	tx := &relinkTransactor{supports: true}
	uc := NewUnlinkRatingDescriptionSetProductPlanUseCase(repo, unlinkServices(relinkDenyAuthorizer(unlinkPermission), tx))

	_, err := uc.Execute(newRelinkContext(), &pb.UnlinkRatingDescriptionSetProductPlanRequest{LinkId: "link-1"})
	if err == nil {
		t.Fatal("expected permission denial")
	}
	if repo.unlinkCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: unlink=%d tx=%d", repo.unlinkCalls, tx.executeCalls)
	}
}

func TestUnlinkRatingDescriptionSetProductPlan_SuccessCommitsInsideTransaction(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-1", Active: false}}
	tx := &relinkTransactor{supports: true}
	uc := NewUnlinkRatingDescriptionSetProductPlanUseCase(repo, unlinkServices(relinkAllowAuthorizer(unlinkPermission), tx))

	resp, err := uc.Execute(newRelinkContext(), &pb.UnlinkRatingDescriptionSetProductPlanRequest{LinkId: "link-1"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %v", resp)
	}
	if repo.unlinkCalls != 1 || tx.executeCalls != 1 || !tx.committed || tx.rolledBack {
		t.Fatalf("expected exactly one committed unlink call: calls=%d executeCalls=%d committed=%v rolledBack=%v",
			repo.unlinkCalls, tx.executeCalls, tx.committed, tx.rolledBack)
	}
}

// TestUnlinkRatingDescriptionSetProductPlan_ReasonReachesTheRepository is
// Unlink's twin of the Relink reason-propagation proof above.
func TestUnlinkRatingDescriptionSetProductPlan_ReasonReachesTheRepository(t *testing.T) {
	repo := &fakeLinkRepo{result: &pb.RatingDescriptionSetProductPlan{Id: "link-1", Active: false}}
	tx := &relinkTransactor{supports: true}
	uc := NewUnlinkRatingDescriptionSetProductPlanUseCase(repo, unlinkServices(relinkAllowAuthorizer(unlinkPermission), tx))

	_, err := uc.Execute(newRelinkContext(), &pb.UnlinkRatingDescriptionSetProductPlanRequest{LinkId: "link-1", Reason: "superseded by Mathematics v2"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.lastReason != "superseded by Mathematics v2" {
		t.Fatalf("expected the request reason to reach UnlinkLocked, got %q", repo.lastReason)
	}
}

func TestUnlinkRatingDescriptionSetProductPlan_AlreadyInactiveRollsBack(t *testing.T) {
	sentinel := errors.New("STALE_LINK: link is already inactive")
	repo := &fakeLinkRepo{unlinkErr: sentinel}
	tx := &relinkTransactor{supports: true}
	uc := NewUnlinkRatingDescriptionSetProductPlanUseCase(repo, unlinkServices(relinkAllowAuthorizer(unlinkPermission), tx))

	resp, err := uc.Execute(newRelinkContext(), &pb.UnlinkRatingDescriptionSetProductPlanRequest{LinkId: "link-1"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got resp=%v err=%v", resp, err)
	}
	if resp != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("already-inactive must rollback with no response: resp=%v committed=%v rolledBack=%v", resp, tx.committed, tx.rolledBack)
	}
}
