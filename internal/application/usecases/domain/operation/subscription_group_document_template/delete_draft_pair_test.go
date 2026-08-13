package subscription_group_document_template

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

var (
	deletePairDocumentPermission = entityid.EntityPermission(entityid.DocumentTemplate, entityid.ActionDelete)
	deletePairBindingPermission  = entityid.EntityPermission(entityid.SubscriptionGroupDocumentTemplate, entityid.ActionDelete)
)

type deletePairRepo struct {
	pb.UnimplementedSubscriptionGroupDocumentTemplateDomainServiceServer
	deleteCalls int
	contexts    []context.Context
	artifact    *documenttemplatepb.DocumentTemplate
	err         error
}

func (r *deletePairRepo) DeleteDraftPair(ctx context.Context, _ string) (*documenttemplatepb.DocumentTemplate, error) {
	r.deleteCalls++
	r.contexts = append(r.contexts, ctx)
	if r.err != nil {
		return nil, r.err
	}
	return r.artifact, nil
}

type deletePairTransactor struct {
	supports      bool
	executeCalls  int
	committed     bool
	rolledBack    bool
	callbackError error
}

func (t *deletePairTransactor) SupportsTransactions() bool { return t.supports }

func (t *deletePairTransactor) IsTransactionActive(context.Context) bool { return false }

func (t *deletePairTransactor) ExecuteInTransaction(_ context.Context, fn func(context.Context) error) error {
	t.executeCalls++
	ctx := context.WithValue(context.Background(), deletePairTransactionKey{}, true)
	err := fn(ctx)
	if err != nil {
		t.rolledBack = true
		return err
	}
	t.committed = true
	return t.callbackError
}

type deletePairTransactionKey struct{}

func deletePairServices(authz *testAuthorizer, tx ports.Transactor) Services {
	return Services{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func deletePairAuthorizer(missing string) *testAuthorizer {
	a := &testAuthorizer{
		enabled: true,
		perms: map[string]bool{
			deletePairDocumentPermission: true,
			deletePairBindingPermission:  true,
		},
	}
	if missing != "" {
		a.perms[missing] = false
	}
	return a
}

func deletePairArtifact() *documenttemplatepb.DocumentTemplate {
	container, key := "grade-templates", "grade-templates/draft.docx"
	return &documenttemplatepb.DocumentTemplate{
		Id:               "artifact-delete-1",
		StorageContainer: &container,
		StorageKey:       &key,
		Active:           false,
	}
}

func newDeletePairUseCase(repo pb.SubscriptionGroupDocumentTemplateDomainServiceServer, authz *testAuthorizer, tx ports.Transactor) *DeleteDraftPairUseCase {
	return NewDeleteDraftPairUseCase(repo, deletePairServices(authz, tx))
}

func TestDeleteDraftPair_MissingPermissionsLeaveNoRepoOrTxCalls(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		missing string
		checks  []string
	}{
		{name: "missing_binding_delete", missing: deletePairBindingPermission, checks: []string{deletePairBindingPermission}},
		{name: "missing_document_delete", missing: deletePairDocumentPermission, checks: []string{deletePairBindingPermission, deletePairDocumentPermission}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &deletePairRepo{artifact: deletePairArtifact()}
			tx := &deletePairTransactor{supports: true}
			authz := deletePairAuthorizer(tc.missing)
			_, err := newDeletePairUseCase(repo, authz, tx).Execute(newContext(), &DeleteDraftPairRequest{BindingID: "binding-delete-1"})
			if err == nil {
				t.Fatal("expected permission denial")
			}
			if repo.deleteCalls != 0 || tx.executeCalls != 0 {
				t.Fatalf("permission denial must precede repo/transaction calls: repo=%d tx=%d", repo.deleteCalls, tx.executeCalls)
			}
			if len(authz.asked) != len(tc.checks) {
				t.Fatalf("checked permissions %v, want %v", authz.asked, tc.checks)
			}
			for i := range tc.checks {
				if authz.asked[i] != tc.checks[i] {
					t.Fatalf("checked permissions %v, want %v", authz.asked, tc.checks)
				}
			}
		})
	}
}

func TestDeleteDraftPair_TransactionUnavailableOrCapabilityMissingFailsBeforeRepositoryMutation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		repo pb.SubscriptionGroupDocumentTemplateDomainServiceServer
		tx   *deletePairTransactor
	}{
		{name: "transaction_unsupported", repo: &deletePairRepo{artifact: deletePairArtifact()}, tx: &deletePairTransactor{supports: false}},
		{name: "repository_capability_missing", repo: &fakeBindingRepo{}, tx: &deletePairTransactor{supports: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newDeletePairUseCase(tc.repo, deletePairAuthorizer(""), tc.tx).Execute(newContext(), &DeleteDraftPairRequest{BindingID: "binding-delete-1"})
			if err == nil {
				t.Fatal("expected precondition failure")
			}
			if tc.tx.executeCalls != 0 {
				t.Fatalf("precondition failure must not execute transaction, got %d", tc.tx.executeCalls)
			}
			if strings.Contains(err.Error(), "repository") && tc.name != "repository_capability_missing" {
				t.Fatalf("unexpected repository error for transaction case: %v", err)
			}
		})
	}
}

func TestDeleteDraftPair_SuccessCallsRepositoryInsideTransactionAndReturnsArtifactAfterCommit(t *testing.T) {
	t.Parallel()

	repo := &deletePairRepo{artifact: deletePairArtifact()}
	tx := &deletePairTransactor{supports: true}
	artifact, err := newDeletePairUseCase(repo, deletePairAuthorizer(""), tx).ExecutePair(newContext(), " binding-delete-1 ")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !tx.committed || tx.rolledBack || tx.executeCalls != 1 {
		t.Fatalf("expected one committed transaction, got executes=%d committed=%v rolledBack=%v", tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if repo.deleteCalls != 1 || len(repo.contexts) != 1 {
		t.Fatalf("expected one repository call in transaction, calls=%d contexts=%d", repo.deleteCalls, len(repo.contexts))
	}
	if !repo.contexts[0].Value(deletePairTransactionKey{}).(bool) {
		t.Fatal("repository call did not receive transaction context")
	}
	if artifact.GetId() != "artifact-delete-1" || artifact.GetStorageContainer() != "grade-templates" || artifact.GetStorageKey() != "grade-templates/draft.docx" {
		t.Fatalf("unexpected committed artifact locator: %v", artifact)
	}
}

func TestDeleteDraftPair_RepositoryErrorRollsBackAndReturnsNoResponse(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository delete failed")
	repo := &deletePairRepo{err: repoErr}
	tx := &deletePairTransactor{supports: true}
	artifact, err := newDeletePairUseCase(repo, deletePairAuthorizer(""), tx).ExecutePair(newContext(), "binding-delete-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error, got artifact=%v err=%v", artifact, err)
	}
	if artifact != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("repository error must rollback with no response: artifact=%v committed=%v rolledBack=%v", artifact, tx.committed, tx.rolledBack)
	}
}

func TestDeleteDraftPair_IncompleteLocatorRollsBackAndReturnsNoResponse(t *testing.T) {
	t.Parallel()

	artifact := deletePairArtifact()
	artifact.StorageKey = nil
	repo := &deletePairRepo{artifact: artifact}
	tx := &deletePairTransactor{supports: true}
	got, err := newDeletePairUseCase(repo, deletePairAuthorizer(""), tx).ExecutePair(newContext(), "binding-delete-1")
	if err == nil || !strings.Contains(err.Error(), "incomplete storage locator") {
		t.Fatalf("expected incomplete locator error, got artifact=%v err=%v", got, err)
	}
	if got != nil || !tx.rolledBack || tx.committed {
		t.Fatalf("incomplete locator must rollback with no response: artifact=%v committed=%v rolledBack=%v", got, tx.committed, tx.rolledBack)
	}
}

func TestDeleteDraftPair_RepositoryErrorPreservesExactError(t *testing.T) {
	t.Parallel()

	sentinel := fmt.Errorf("exact sentinel")
	repo := &deletePairRepo{err: sentinel}
	tx := &deletePairTransactor{supports: true}
	_, err := newDeletePairUseCase(repo, deletePairAuthorizer(""), tx).Execute(newContext(), &DeleteDraftPairRequest{BindingID: "binding-delete-1"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel propagation, got %v", err)
	}
}
