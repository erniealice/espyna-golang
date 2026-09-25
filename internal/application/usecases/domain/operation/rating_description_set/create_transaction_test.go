package rating_description_set

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
)

// fix3-backend (codex-review-impl3 #4): set creation + its audit event run in
// ONE transaction; an audit (create) failure rolls back; no transactor → fail
// closed before any write.

type txTrackingSetRepo struct {
	*validatingSetRepo
	createErr   error
	sawTxCtx    bool
	validateCtx bool
}

type txMarkerKey struct{}

type markingTransactor struct{ lifecycleTransactor }

func (t *markingTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return t.lifecycleTransactor.ExecuteInTransaction(context.WithValue(ctx, txMarkerKey{}, true), fn)
}

func (r *txTrackingSetRepo) ValidateRatingDescriptionSetScoreScale(ctx context.Context, id string) error {
	r.validateCtx = ctx.Value(txMarkerKey{}) == true
	return r.validatingSetRepo.ValidateRatingDescriptionSetScoreScale(ctx, id)
}

func (r *txTrackingSetRepo) CreateRatingDescriptionSet(ctx context.Context, req *pb.CreateRatingDescriptionSetRequest) (*pb.CreateRatingDescriptionSetResponse, error) {
	r.sawTxCtx = ctx.Value(txMarkerKey{}) == true
	r.createCalls++
	if r.createErr != nil {
		return nil, r.createErr
	}
	return &pb.CreateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{req.Data}, Success: true}, nil
}

func createServicesWithTx(authz *lifecycleTestAuthorizer, tx ports.Transactor) CreateRatingDescriptionSetServices {
	return CreateRatingDescriptionSetServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       tx,
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func TestCreateRatingDescriptionSet_CreateAndValidateRunInsideTransaction(t *testing.T) {
	repo := &txTrackingSetRepo{validatingSetRepo: &validatingSetRepo{fakeSetRepo: &fakeSetRepo{}}}
	tx := &markingTransactor{lifecycleTransactor{supports: true}}
	uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createServicesWithTx(allowAuthorizer(createSetPermission), tx))
	resp, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "own"}})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("create: resp=%v err=%v", resp, err)
	}
	if tx.executeCalls != 1 || !tx.committed || tx.rolledBack {
		t.Fatalf("want one committed tx: calls=%d committed=%v rolledBack=%v", tx.executeCalls, tx.committed, tx.rolledBack)
	}
	if !repo.sawTxCtx || !repo.validateCtx {
		t.Fatalf("validate and create must run on the tx context: validate=%v create=%v", repo.validateCtx, repo.sawTxCtx)
	}
}

func TestCreateRatingDescriptionSet_AuditFailureRollsBack(t *testing.T) {
	auditErr := errors.New("audit: insert audit_entry: boom")
	repo := &txTrackingSetRepo{validatingSetRepo: &validatingSetRepo{fakeSetRepo: &fakeSetRepo{}}, createErr: auditErr}
	tx := &markingTransactor{lifecycleTransactor{supports: true}}
	uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createServicesWithTx(allowAuthorizer(createSetPermission), tx))
	resp, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "own"}})
	if !errors.Is(err, auditErr) || resp != nil {
		t.Fatalf("want the audit error and no response, got resp=%v err=%v", resp, err)
	}
	if !tx.rolledBack || tx.committed {
		t.Fatalf("audit failure must roll back: committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
}

func TestCreateRatingDescriptionSet_TransactorUnsupportedFailsClosed(t *testing.T) {
	for name, tx := range map[string]ports.Transactor{"nil": nil, "unsupported": &lifecycleTransactor{supports: false}} {
		t.Run(name, func(t *testing.T) {
			repo := &txTrackingSetRepo{validatingSetRepo: &validatingSetRepo{fakeSetRepo: &fakeSetRepo{}}}
			uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createServicesWithTx(allowAuthorizer(createSetPermission), tx))
			if _, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "own"}}); err == nil {
				t.Fatal("create without transaction support must fail closed")
			}
			if repo.createCalls != 0 {
				t.Fatalf("no write without a transaction: createCalls=%d", repo.createCalls)
			}
		})
	}
}
