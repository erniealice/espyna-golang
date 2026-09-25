package rating_description_set

import (
	"context"
	"errors"
	"strings"
	"testing"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

// fix2-backend (codex impl2 #1): fakeSetRepo satisfies the publish lock
// protocol port with no-op defaults so the pre-existing publish tests keep
// exercising the happy path.
func (r *fakeSetRepo) LockScoreScaleForPublish(_ context.Context, _ string) (string, error) {
	return "scale-1", nil
}

func (r *fakeSetRepo) VerifyRatingDescriptionSetPublishScale(_ context.Context, _, _ string) error {
	return nil
}

// orderedSetRepo records the lock-protocol call order.
type orderedSetRepo struct {
	*fakeSetRepo
	calls     []string
	scaleErr  error
	verifyErr error
}

func (r *orderedSetRepo) LockScoreScaleForPublish(_ context.Context, _ string) (string, error) {
	r.calls = append(r.calls, "scale")
	if r.scaleErr != nil {
		return "", r.scaleErr
	}
	return "scale-1", nil
}

func (r *orderedSetRepo) LockRatingDescriptionSetForUpdate(ctx context.Context, id string) (enumspb.VersionStatus, error) {
	r.calls = append(r.calls, "set")
	return r.fakeSetRepo.LockRatingDescriptionSetForUpdate(ctx, id)
}

func (r *orderedSetRepo) VerifyRatingDescriptionSetPublishScale(_ context.Context, _, scaleID string) error {
	r.calls = append(r.calls, "verify:"+scaleID)
	return r.verifyErr
}

func (r *orderedSetRepo) PublishRatingDescriptionSetIfDraft(ctx context.Context, id string) (*pb.RatingDescriptionSet, error) {
	r.calls = append(r.calls, "publish")
	return r.fakeSetRepo.PublishRatingDescriptionSetIfDraft(ctx, id)
}

func TestPublishRatingDescriptionSet_LockOrderScaleThenSet(t *testing.T) {
	repo := &orderedSetRepo{fakeSetRepo: &fakeSetRepo{result: &pb.RatingDescriptionSet{Id: "set-1", VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED}}}
	entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
	tx := &lifecycleTransactor{supports: true}
	uc := NewPublishRatingDescriptionSetUseCase(repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))
	if _, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	got := strings.Join(repo.calls, ",")
	if got != "scale,set,verify:scale-1,publish" {
		t.Fatalf("lock order = %s, want scale,set,verify:scale-1,publish", got)
	}
}

func TestPublishRatingDescriptionSet_InvalidScaleRejectsWithoutPublish(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo *orderedSetRepo
	}{
		{"scale lock fails", &orderedSetRepo{fakeSetRepo: &fakeSetRepo{}, scaleErr: errors.New("INVALID_CONFIG: foreign scale")}},
		{"verify fails", &orderedSetRepo{fakeSetRepo: &fakeSetRepo{}, verifyErr: errors.New("INVALID_CONFIG: foreign band")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &lifecycleTransactor{supports: true}
			entryRepo := &fakeEntryRepo{entries: []*entrypb.RatingDescriptionSetEntry{{Id: "entry-1"}}}
			uc := NewPublishRatingDescriptionSetUseCase(tc.repo, entryRepo, lifecycleServices(allowAuthorizer(publishPermission), tx))
			_, err := uc.Execute(newLifecycleContext(), &pb.PublishRatingDescriptionSetRequest{Id: "set-1"})
			if err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
				t.Fatalf("expected INVALID_CONFIG, got %v", err)
			}
			if tc.repo.publishCalls != 0 || tx.committed {
				t.Fatalf("publish must not run/commit: publishCalls=%d committed=%v", tc.repo.publishCalls, tx.committed)
			}
		})
	}
}
