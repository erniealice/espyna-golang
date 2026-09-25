package rating_description_set

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

// --- shared test doubles ---------------------------------------------------

type lifecycleTestAuthorizer struct {
	enabled bool
	perms   map[string]bool
	asked   []string
}

func (a *lifecycleTestAuthorizer) IsEnabled() bool { return a.enabled }

func (a *lifecycleTestAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	if a.perms == nil {
		return false, nil
	}
	return a.perms[permission], nil
}

func newLifecycleContext() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func lifecycleServices(authz *lifecycleTestAuthorizer, tx ports.Transactor) PublishRatingDescriptionSetServices {
	return PublishRatingDescriptionSetServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func deprecateServices(authz *lifecycleTestAuthorizer, tx ports.Transactor) DeprecateRatingDescriptionSetServices {
	return DeprecateRatingDescriptionSetServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func updateSetServices(authz *lifecycleTestAuthorizer, tx ports.Transactor) UpdateRatingDescriptionSetServices {
	return UpdateRatingDescriptionSetServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

type lifecycleTransactor struct {
	supports     bool
	executeCalls int
	committed    bool
	rolledBack   bool
}

func (t *lifecycleTransactor) SupportsTransactions() bool               { return t.supports }
func (t *lifecycleTransactor) IsTransactionActive(context.Context) bool { return false }
func (t *lifecycleTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.executeCalls++
	if err := fn(ctx); err != nil {
		t.rolledBack = true
		return err
	}
	t.committed = true
	return nil
}

// fakeSetRepo implements pb.RatingDescriptionSetDomainServiceServer plus the
// four conditional lifecycle methods (domainports.RatingDescriptionSetLifecycleRepository).
type fakeSetRepo struct {
	pb.UnimplementedRatingDescriptionSetDomainServiceServer
	publishCalls   int
	deprecateCalls int
	lockCalls      int
	updateCalls    int
	publishErr     error
	deprecateErr   error
	lockErr        error
	updateErr      error
	// lockStatus is returned by LockRatingDescriptionSetForUpdate; defaults to
	// DRAFT (the common case exercised by these tests) unless overridden.
	lockStatus enumspb.VersionStatus
	result     *pb.RatingDescriptionSet
	// lastUpdateReq captures the request UpdateRatingDescriptionSetIfDraft
	// actually received, so tests can assert lifecycle-owned fields were
	// stripped/cleared before the write reached the repository.
	lastUpdateReq *pb.UpdateRatingDescriptionSetRequest
}

func (r *fakeSetRepo) PublishRatingDescriptionSetIfDraft(_ context.Context, _ string) (*pb.RatingDescriptionSet, error) {
	r.publishCalls++
	if r.publishErr != nil {
		return nil, r.publishErr
	}
	return r.result, nil
}

func (r *fakeSetRepo) DeprecateRatingDescriptionSetIfPublished(_ context.Context, _ string) (*pb.RatingDescriptionSet, error) {
	r.deprecateCalls++
	if r.deprecateErr != nil {
		return nil, r.deprecateErr
	}
	return r.result, nil
}

func (r *fakeSetRepo) LockRatingDescriptionSetForUpdate(_ context.Context, _ string) (enumspb.VersionStatus, error) {
	r.lockCalls++
	if r.lockErr != nil {
		return enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED, r.lockErr
	}
	if r.lockStatus == enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED {
		return enumspb.VersionStatus_VERSION_STATUS_DRAFT, nil
	}
	return r.lockStatus, nil
}

func (r *fakeSetRepo) UpdateRatingDescriptionSetIfDraft(_ context.Context, req *pb.UpdateRatingDescriptionSetRequest) (*pb.RatingDescriptionSet, error) {
	r.updateCalls++
	r.lastUpdateReq = req
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return r.result, nil
}

// fakeSetRepoNoLifecycle implements ONLY the generic CRUD interface — used to
// exercise the repositoryError precondition path.
type fakeSetRepoNoLifecycle struct {
	pb.UnimplementedRatingDescriptionSetDomainServiceServer
}

type fakeEntryRepo struct {
	entrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
	entries []*entrypb.RatingDescriptionSetEntry
	err     error
}

func (r *fakeEntryRepo) ListRatingDescriptionSetEntries(_ context.Context, _ *entrypb.ListRatingDescriptionSetEntriesRequest) (*entrypb.ListRatingDescriptionSetEntriesResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &entrypb.ListRatingDescriptionSetEntriesResponse{Data: r.entries, Success: true}, nil
}

var (
	publishPermission   = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionPublish)
	deprecatePermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionDeprecate)
	updateSetPermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionUpdate)
)

func allowAuthorizer(perm string) *lifecycleTestAuthorizer {
	return &lifecycleTestAuthorizer{enabled: true, perms: map[string]bool{perm: true}}
}

func denyAuthorizer(perm string) *lifecycleTestAuthorizer {
	return &lifecycleTestAuthorizer{enabled: true, perms: map[string]bool{perm: false}}
}

// --- Publish -----------------------------------------------------------

func TestPublishRatingDescriptionSet_MissingPermissionDeniesBeforeAnyCall(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED}}
	entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(denyAuthorizer(publishPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if err == nil {
		t.Fatal("expected permission denial")
	}
	if repo.publishCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: publish=%d tx=%d", repo.publishCalls, tx.executeCalls)
	}
}

// W3 follow-up finding (2026-09-25): the entry count is now taken AFTER the
// row lock, INSIDE the transaction (so the lock actually serializes against a
// concurrent entry delete) — the guard no longer precedes tx.executeCalls;
// instead it locks (lockCalls=1), lists with zero entries, and rolls back
// without ever calling PublishRatingDescriptionSetIfDraft.
func TestPublishRatingDescriptionSet_RequiresAtLeastOneEntry(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}
	entryRepo := &fakeEntryRepo{entries: nil}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if err == nil {
		t.Fatal("expected publish-requires-entry error")
	}
	if repo.lockCalls != 1 {
		t.Fatalf("expected the parent row lock to be taken before the entry count, got lockCalls=%d", repo.lockCalls)
	}
	if repo.publishCalls != 0 {
		t.Fatalf("entry-count guard (post-lock) must precede the conditional publish write: publish=%d", repo.publishCalls)
	}
	if tx.executeCalls != 1 || tx.committed || !tx.rolledBack {
		t.Fatalf("expected the guard to run and roll back INSIDE one transaction: executeCalls=%d committed=%v rolledBack=%v",
			tx.executeCalls, tx.committed, tx.rolledBack)
	}
}

// TestPublishRatingDescriptionSet_EntryCountReadAfterLock proves the ordering
// contract directly: LockRatingDescriptionSetForUpdate must be called before
// the entry repository is consulted (a spy entry repo records whether the
// lock had already happened by the time it was called).
func TestPublishRatingDescriptionSet_EntryCountReadAfterLock(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED}}
	entryRepo := &lockOrderSpyEntryRepo{setRepo: repo, entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !entryRepo.lockedWhenCalled {
		t.Fatal("expected the parent row lock to already be held when the entry list was read")
	}
}

type lockOrderSpyEntryRepo struct {
	entrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer
	setRepo          *fakeSetRepo
	entries          []*entrypb.RatingDescriptionSetEntry
	lockedWhenCalled bool
}

func (r *lockOrderSpyEntryRepo) ListRatingDescriptionSetEntries(_ context.Context, _ *entrypb.ListRatingDescriptionSetEntriesRequest) (*entrypb.ListRatingDescriptionSetEntriesResponse, error) {
	r.lockedWhenCalled = r.setRepo.lockCalls >= 1
	return &entrypb.ListRatingDescriptionSetEntriesResponse{Data: r.entries, Success: true}, nil
}

func TestPublishRatingDescriptionSet_SuccessCommitsInsideTransaction(t *testing.T) {
	published := &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED}
	repo := &fakeSetRepo{result: published}
	entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))

	resp, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.publishCalls != 1 || tx.executeCalls != 1 || !tx.committed || tx.rolledBack {
		t.Fatalf("expected exactly one committed publish call: calls=%d executeCalls=%d committed=%v rolledBack=%v",
			repo.publishCalls, tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if len(resp.Data) != 1 || resp.Data[0].VersionStatus != enumspb.VersionStatus_VERSION_STATUS_PUBLISHED {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestPublishRatingDescriptionSet_ConflictRollsBackAndReturnsNoResponse(t *testing.T) {
	sentinel := errors.New("SET_NOT_DRAFT: not draft")
	repo := &fakeSetRepo{publishErr: sentinel}
	entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))

	resp, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got resp=%v err=%v", resp, err)
	}
	if resp != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("conflict must rollback with no response: resp=%v committed=%v rolledBack=%v", resp, tx.committed, tx.rolledBack)
	}
}

func TestPublishRatingDescriptionSet_RepositoryWithoutLifecycleCapabilityFailsClosed(t *testing.T) {
	repo := &fakeSetRepoNoLifecycle{}
	entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
	if err == nil {
		t.Fatal("expected repository-capability error")
	}
	if tx.executeCalls != 0 {
		t.Fatalf("capability precondition must not execute a transaction, got %d", tx.executeCalls)
	}
}

// --- Deprecate ---------------------------------------------------------

func TestDeprecateRatingDescriptionSet_MissingPermissionDeniesBeforeAnyCall(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_DEPRECATED}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewDeprecateRatingDescriptionSetUseCase(repo, deprecateServices(denyAuthorizer(deprecatePermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.DeprecateRatingDescriptionSetRequest{Id: "set-1"})
	if err == nil {
		t.Fatal("expected permission denial")
	}
	if repo.deprecateCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: deprecate=%d tx=%d", repo.deprecateCalls, tx.executeCalls)
	}
}

func TestDeprecateRatingDescriptionSet_SuccessCommitsInsideTransaction(t *testing.T) {
	deprecated := &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_DEPRECATED}
	repo := &fakeSetRepo{result: deprecated}
	tx := &lifecycleTransactor{supports: true}
	uc := NewDeprecateRatingDescriptionSetUseCase(repo, deprecateServices(allowAuthorizer(deprecatePermission), tx))

	resp, err := uc.Execute(newLifecycleContext(), &pb.DeprecateRatingDescriptionSetRequest{Id: "set-1"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.deprecateCalls != 1 || tx.executeCalls != 1 || !tx.committed || tx.rolledBack {
		t.Fatalf("expected exactly one committed deprecate call: calls=%d executeCalls=%d committed=%v rolledBack=%v",
			repo.deprecateCalls, tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if len(resp.Data) != 1 || resp.Data[0].VersionStatus != enumspb.VersionStatus_VERSION_STATUS_DEPRECATED {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestDeprecateRatingDescriptionSet_ConflictRollsBackAndReturnsNoResponse(t *testing.T) {
	sentinel := errors.New("SET_NOT_PUBLISHED: not published")
	repo := &fakeSetRepo{deprecateErr: sentinel}
	tx := &lifecycleTransactor{supports: true}
	uc := NewDeprecateRatingDescriptionSetUseCase(repo, deprecateServices(allowAuthorizer(deprecatePermission), tx))

	resp, err := uc.Execute(newLifecycleContext(), &pb.DeprecateRatingDescriptionSetRequest{Id: "set-1"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got resp=%v err=%v", resp, err)
	}
	if resp != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("conflict must rollback with no response: resp=%v committed=%v rolledBack=%v", resp, tx.committed, tx.rolledBack)
	}
}

func TestDeprecateRatingDescriptionSet_TransactorUnsupportedFailsBeforeRepoCall(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}
	tx := &lifecycleTransactor{supports: false}
	uc := NewDeprecateRatingDescriptionSetUseCase(repo, deprecateServices(allowAuthorizer(deprecatePermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.DeprecateRatingDescriptionSetRequest{Id: "set-1"})
	if err == nil {
		t.Fatal("expected transactor-unavailable error")
	}
	if repo.deprecateCalls != 0 {
		t.Fatalf("must not call repo without transaction support, got %d", repo.deprecateCalls)
	}
}

// --- Update (W3 follow-up finding, 2026-09-25) --------------------------
//
// codex-review-impl1.out.md: "Set update reads DRAFT outside a transaction
// and subsequently writes that status back: a concurrent publish can be
// reverted to DRAFT." These tests prove: (1) the DRAFT check and the write
// happen inside ONE transaction, lock-first; (2) a non-DRAFT lock result
// rejects without ever writing; (3) version_status/version/supersedes_id are
// always cleared on the request BEFORE the repository write, never copied
// from a pre-read snapshot.

func TestUpdateRatingDescriptionSet_MissingPermissionDeniesBeforeAnyCall(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(denyAuthorizer(updateSetPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed"}})
	if err == nil {
		t.Fatal("expected permission denial")
	}
	if repo.lockCalls != 0 || repo.updateCalls != 0 || tx.executeCalls != 0 {
		t.Fatalf("permission denial must precede repo/tx calls: lock=%d update=%d tx=%d", repo.lockCalls, repo.updateCalls, tx.executeCalls)
	}
}

func TestUpdateRatingDescriptionSet_RepositoryWithoutLifecycleCapabilityFailsClosed(t *testing.T) {
	repo := &fakeSetRepoNoLifecycle{}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(allowAuthorizer(updateSetPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed"}})
	if err == nil {
		t.Fatal("expected repository-capability error")
	}
	if tx.executeCalls != 0 {
		t.Fatalf("capability precondition must not execute a transaction, got %d", tx.executeCalls)
	}
}

func TestUpdateRatingDescriptionSet_TransactorUnsupportedFailsBeforeRepoCall(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}
	tx := &lifecycleTransactor{supports: false}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(allowAuthorizer(updateSetPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed"}})
	if err == nil {
		t.Fatal("expected transactor-unavailable error")
	}
	if repo.lockCalls != 0 || repo.updateCalls != 0 {
		t.Fatalf("must not call repo without transaction support: lock=%d update=%d", repo.lockCalls, repo.updateCalls)
	}
}

func TestUpdateRatingDescriptionSet_LockedNotDraftRejectsWithoutWriteInsideOneTransaction(t *testing.T) {
	repo := &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}, lockStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(allowAuthorizer(updateSetPermission), tx))

	_, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed"}})
	if err == nil {
		t.Fatal("expected SET_NOT_DRAFT rejection")
	}
	if repo.lockCalls != 1 {
		t.Fatalf("expected exactly one lock call, got %d", repo.lockCalls)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("a non-DRAFT lock result must never reach the write, got updateCalls=%d", repo.updateCalls)
	}
	if tx.executeCalls != 1 || tx.committed || !tx.rolledBack {
		t.Fatalf("expected the lock+check to run and roll back INSIDE one transaction: executeCalls=%d committed=%v rolledBack=%v",
			tx.executeCalls, tx.committed, tx.rolledBack)
	}
}

func TestUpdateRatingDescriptionSet_SuccessLocksBeforeWriteAndStripsLifecycleFields(t *testing.T) {
	repo := &fakeSetRepo{
		result:     &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_DRAFT},
		lockStatus: enumspb.VersionStatus_VERSION_STATUS_DRAFT,
	}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(allowAuthorizer(updateSetPermission), tx))

	// A caller (malicious or stale) supplies VersionStatus/Version/SupersedesId
	// on the request — these must never reach the repository write.
	staleSupersedes := "set-0"
	resp, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{
		Id:            "set-1",
		Name:          "Renamed",
		VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED,
		Version:       99,
		SupersedesId:  &staleSupersedes,
	}})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.lockCalls != 1 || repo.updateCalls != 1 {
		t.Fatalf("expected exactly one lock then one update, got lockCalls=%d updateCalls=%d", repo.lockCalls, repo.updateCalls)
	}
	if !tx.committed || tx.rolledBack || tx.executeCalls != 1 {
		t.Fatalf("expected exactly one committed transaction: executeCalls=%d committed=%v rolledBack=%v", tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if repo.lastUpdateReq == nil || repo.lastUpdateReq.Data == nil {
		t.Fatal("expected the repository to receive the update request")
	}
	got := repo.lastUpdateReq.Data
	if got.VersionStatus != enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED {
		t.Fatalf("expected version_status cleared before the write, got %v", got.VersionStatus)
	}
	if got.Version != 0 {
		t.Fatalf("expected version cleared before the write, got %d", got.Version)
	}
	if got.SupersedesId != nil {
		t.Fatalf("expected supersedes_id cleared before the write, got %v", *got.SupersedesId)
	}
	if len(resp.Data) != 1 || resp.Data[0].Name != "Renamed" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestUpdateRatingDescriptionSet_ConflictRollsBackAndReturnsNoResponse(t *testing.T) {
	sentinel := errors.New("boom: write failed")
	repo := &fakeSetRepo{lockStatus: enumspb.VersionStatus_VERSION_STATUS_DRAFT, updateErr: sentinel}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(
		UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo},
		updateSetServices(allowAuthorizer(updateSetPermission), tx))

	resp, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "Renamed"}})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got resp=%v err=%v", resp, err)
	}
	if resp != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("conflict must rollback with no response: resp=%v committed=%v rolledBack=%v", resp, tx.committed, tx.rolledBack)
	}
}
