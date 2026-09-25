package rating_description_set

import (
	"context"
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	scalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// pagedScoreScaleRepo honors the offset-pagination request (page number x
// limit) exactly like the real postgres adapter, so a test can prove the
// Add-set drawer's score_scale picker pages past a single 100-row response
// instead of silently truncating (codex-review-impl4.out.md round-3/round-4
// disposition #2 — "any other page-data picker reads (scales...)").
type pagedScoreScaleRepo struct {
	scalepb.UnimplementedScoreScaleDomainServiceServer
	items []*scalepb.ScoreScale
	calls int
}

func (r *pagedScoreScaleRepo) ListScoreScales(_ context.Context, req *scalepb.ListScoreScalesRequest) (*scalepb.ListScoreScalesResponse, error) {
	r.calls++
	page := int(req.GetPagination().GetOffset().GetPage())
	limit := int(req.GetPagination().GetLimit())
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(r.items) {
		start = len(r.items)
	}
	end := start + limit
	if end > len(r.items) {
		end = len(r.items)
	}
	return &scalepb.ListScoreScalesResponse{Data: r.items[start:end], Success: true}, nil
}

// TestGetRatingDescriptionSetFormPageData_ScalePickerPagesPastSingleAdapterPage
// proves the Add-set drawer's score_scale picker returns ALL active scales,
// not just the adapter's first 100-row page (codex-review-impl4.out.md
// round-3/round-4 disposition #2).
func TestGetRatingDescriptionSetFormPageData_ScalePickerPagesPastSingleAdapterPage(t *testing.T) {
	const n = 137
	scales := make([]*scalepb.ScoreScale, 0, n)
	for i := 0; i < n; i++ {
		scales = append(scales, &scalepb.ScoreScale{Id: fmt.Sprintf("scale-%03d", i), Name: fmt.Sprintf("Scale %03d", i), Active: true})
	}
	repo := &pagedScoreScaleRepo{items: scales}

	authz := allowAuthorizer(entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionCreate))
	uc := NewGetRatingDescriptionSetFormPageDataUseCase(
		GetRatingDescriptionSetFormPageDataRepositories{ScoreScale: repo},
		GetRatingDescriptionSetFormPageDataServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		},
	)

	resp, err := uc.Execute(newLifecycleContext(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ScoreScales) != n {
		t.Fatalf("expected all %d scales, got %d", n, len(resp.ScoreScales))
	}
	if repo.calls < 2 {
		t.Fatalf("expected the scale picker to page (>1 call for %d rows at 100/page), got %d calls", n, repo.calls)
	}
}
