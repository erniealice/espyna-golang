package subscription_group_document_template

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

type publishOnlyRepo struct {
	pb.UnimplementedSubscriptionGroupDocumentTemplateDomainServiceServer
	response *pb.PublishSubscriptionGroupDocumentTemplateResponse
	err      error
	calls    int
}

func (r *publishOnlyRepo) PublishSubscriptionGroupDocumentTemplate(_ context.Context, _ *pb.PublishSubscriptionGroupDocumentTemplateRequest) (*pb.PublishSubscriptionGroupDocumentTemplateResponse, error) {
	r.calls++
	return r.response, r.err
}

func newPublishUseCase(repo *publishOnlyRepo, tx *testTransactor) *PublishUseCase {
	return &PublishUseCase{
		repo: repo,
		svc: Services{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       tx,
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(allowedAuthorizer(), ports.NewNoOpTranslator()),
		},
	}
}

func TestPublishSubscriptionGroupDocumentTemplateRequiresTransactionAndValidResponse(t *testing.T) {
	t.Parallel()

	t.Run("success commits only after hydrated response", func(t *testing.T) {
		repo := &publishOnlyRepo{response: &pb.PublishSubscriptionGroupDocumentTemplateResponse{
			Success: true,
			Data:    &pb.SubscriptionGroupDocumentTemplate{Id: "binding-1"},
		}}
		tx := &testTransactor{supports: true}
		response, err := newPublishUseCase(repo, tx).Execute(newContext(), &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "binding-1"})
		if err != nil {
			t.Fatalf("publish: %v", err)
		}
		if response.GetData().GetId() != "binding-1" || repo.calls != 1 || tx.executeCalls != 1 {
			t.Fatalf("unexpected success state: response=%v calls=%d tx=%d", response, repo.calls, tx.executeCalls)
		}
	})

	t.Run("transaction unavailable", func(t *testing.T) {
		repo := &publishOnlyRepo{}
		_, err := newPublishUseCase(repo, &testTransactor{supports: false}).Execute(newContext(), &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "binding-1"})
		if err == nil || repo.calls != 0 {
			t.Fatalf("expected pre-repository transaction failure; err=%v calls=%d", err, repo.calls)
		}
	})

	t.Run("repository error rolls back", func(t *testing.T) {
		repo := &publishOnlyRepo{err: errors.New("publish failed")}
		tx := &testTransactor{supports: true, rollbackExpected: true}
		_, err := newPublishUseCase(repo, tx).Execute(newContext(), &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "binding-1"})
		if err == nil || !tx.rolledBack {
			t.Fatalf("expected rollback; err=%v rolledBack=%t", err, tx.rolledBack)
		}
	})

	t.Run("invalid response rolls back", func(t *testing.T) {
		repo := &publishOnlyRepo{response: &pb.PublishSubscriptionGroupDocumentTemplateResponse{Success: true}}
		tx := &testTransactor{supports: true, rollbackExpected: true}
		_, err := newPublishUseCase(repo, tx).Execute(newContext(), &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "binding-1"})
		if err == nil || !tx.rolledBack {
			t.Fatalf("expected invalid response rollback; err=%v rolledBack=%t", err, tx.rolledBack)
		}
	})
}
