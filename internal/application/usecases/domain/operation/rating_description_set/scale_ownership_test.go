package rating_description_set

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
)

// fix2-backend (codex impl2 #4): set create/update validate the referenced
// score_scale's workspace ownership (or global) and fail closed.

type validatingSetRepo struct {
	*fakeSetRepo
	validateErr  error
	validatedIDs []string
	createCalls  int
}

func (r *validatingSetRepo) ValidateRatingDescriptionSetScoreScale(_ context.Context, id string) error {
	r.validatedIDs = append(r.validatedIDs, id)
	return r.validateErr
}

func (r *validatingSetRepo) CreateRatingDescriptionSet(_ context.Context, req *pb.CreateRatingDescriptionSetRequest) (*pb.CreateRatingDescriptionSetResponse, error) {
	r.createCalls++
	return &pb.CreateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{req.Data}, Success: true}, nil
}

var createSetPermission = entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionCreate)

func createSetServices(authz *lifecycleTestAuthorizer) CreateRatingDescriptionSetServices {
	return CreateRatingDescriptionSetServices{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       &lifecycleTransactor{supports: true},
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	}
}

func TestCreateRatingDescriptionSet_ForeignScaleRejected(t *testing.T) {
	repo := &validatingSetRepo{fakeSetRepo: &fakeSetRepo{}, validateErr: errors.New("INVALID_CONFIG: score scale owned by another workspace")}
	uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createSetServices(allowAuthorizer(createSetPermission)))
	_, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "foreign-scale"}})
	if err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("want INVALID_CONFIG, got %v", err)
	}
	if repo.createCalls != 0 || len(repo.validatedIDs) != 1 || repo.validatedIDs[0] != "foreign-scale" {
		t.Fatalf("create must not run: createCalls=%d validated=%v", repo.createCalls, repo.validatedIDs)
	}
}

func TestCreateRatingDescriptionSet_OwnedScaleCreates(t *testing.T) {
	repo := &validatingSetRepo{fakeSetRepo: &fakeSetRepo{}}
	uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createSetServices(allowAuthorizer(createSetPermission)))
	if _, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "own-scale"}}); err != nil {
		t.Fatalf("owned scale must create: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("createCalls=%d", repo.createCalls)
	}
}

func TestCreateRatingDescriptionSet_NoValidatorFailsClosed(t *testing.T) {
	repo := &fakeSetRepoNoLifecycle{}
	uc := NewCreateRatingDescriptionSetUseCase(CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, createSetServices(allowAuthorizer(createSetPermission)))
	if _, err := uc.Execute(newLifecycleContext(), &pb.CreateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Name: "n", ScoreScaleId: "s"}}); err == nil {
		t.Fatal("repository without ownership validation must fail closed")
	}
}

func TestUpdateRatingDescriptionSet_ForeignScaleRejectedWithoutWrite(t *testing.T) {
	repo := &validatingSetRepo{fakeSetRepo: &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}, validateErr: errors.New("INVALID_CONFIG: foreign")}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, updateSetServices(allowAuthorizer(updateSetPermission), tx))
	_, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", ScoreScaleId: "foreign-scale"}})
	if err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("want INVALID_CONFIG, got %v", err)
	}
	if repo.updateCalls != 0 || tx.committed {
		t.Fatalf("no write/commit allowed: updateCalls=%d committed=%v", repo.updateCalls, tx.committed)
	}
}

func TestUpdateRatingDescriptionSet_NoScaleChangeSkipsValidation(t *testing.T) {
	repo := &validatingSetRepo{fakeSetRepo: &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1"}}, validateErr: errors.New("should not be called")}
	tx := &lifecycleTransactor{supports: true}
	uc := NewUpdateRatingDescriptionSetUseCase(UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, updateSetServices(allowAuthorizer(updateSetPermission), tx))
	if _, err := uc.Execute(newLifecycleContext(), &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: "set-1", Name: "renamed"}}); err != nil {
		t.Fatalf("name-only update: %v", err)
	}
	if len(repo.validatedIDs) != 0 || repo.updateCalls != 1 {
		t.Fatalf("validated=%v updateCalls=%d", repo.validatedIDs, repo.updateCalls)
	}
}
