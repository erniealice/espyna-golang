package rating_description_set

import (
	"context"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	scalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// formPickerPageLimit/formPickerMaxPages bound the offset-pagination loop
// below (codex-review-impl4.out.md round-3 disposition #2 — "any other
// page-data picker reads (scales...)" — a single unpaged call silently
// truncated at the adapter's 100-row default cap).
const (
	formPickerPageLimit = 100
	formPickerMaxPages  = 50
)

func formPickerPagination(page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  formPickerPageLimit,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}

func formPickerIDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// FormPickerOption is one <select> entry for a rating_description_set drawer.
type FormPickerOption struct {
	ID   string
	Name string
}

// GetRatingDescriptionSetFormPageDataRequest has no fields today — the
// picker is workspace-wide (the Add-set drawer's score_scale picker is not
// scoped by any other selection). A plain (non-protobuf) request/response
// pair, like the Relink/Unlink use cases on the sibling
// rating_description_set_product_plan package (proto comment: "PLAIN
// use-case request/response messages — not repository RPCs").
type GetRatingDescriptionSetFormPageDataRequest struct{}

// GetRatingDescriptionSetFormPageDataResponse carries every picker option the
// rating_description_set drawer needs.
type GetRatingDescriptionSetFormPageDataResponse struct {
	ScoreScales []FormPickerOption
}

type GetRatingDescriptionSetFormPageDataRepositories struct {
	// ScoreScale backs the score_scale picker. Read directly here (not
	// through score_scale's own gated ListScoreScalesUseCase) so this page
	// data is authorized once, under rating_description_set:create — not
	// under a second, separate score_scale:list grant (schema-proposal.md
	// §9.4: "Pickers: page-data use cases return every picker option the
	// drawers need ... scoped to the workspace — no extra score_scale:list /
	// outcome_criteria:list grants." codex-review-impl2.out.md finding #6).
	ScoreScale scalepb.ScoreScaleDomainServiceServer
}

type GetRatingDescriptionSetFormPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetFormPageDataUseCase struct {
	repositories GetRatingDescriptionSetFormPageDataRepositories
	services     GetRatingDescriptionSetFormPageDataServices
}

func NewGetRatingDescriptionSetFormPageDataUseCase(r GetRatingDescriptionSetFormPageDataRepositories, s GetRatingDescriptionSetFormPageDataServices) *GetRatingDescriptionSetFormPageDataUseCase {
	return &GetRatingDescriptionSetFormPageDataUseCase{repositories: r, services: s}
}

// Execute authorizes on rating_description_set:create — the only action that
// currently opens a rating_description_set drawer needing this picker
// (Add-set; there is no Edit-header action yet, W3-FAYNA-VIEWS TODO) — then
// reads score_scale directly via the repository interface, bypassing
// score_scale's own ActionGatekeeper check entirely (this use case's own
// check IS the authorization for this read; a management-permission holder
// is not additionally required to hold score_scale:list).
func (uc *GetRatingDescriptionSetFormPageDataUseCase) Execute(ctx context.Context, _ *GetRatingDescriptionSetFormPageDataRequest) (*GetRatingDescriptionSetFormPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	resp := &GetRatingDescriptionSetFormPageDataResponse{}
	if uc.repositories.ScoreScale != nil {
		for page := int32(1); page <= formPickerMaxPages; page++ {
			scalesResp, err := uc.repositories.ScoreScale.ListScoreScales(ctx, &scalepb.ListScoreScalesRequest{Sort: formPickerIDSort(), Pagination: formPickerPagination(page)})
			if err != nil {
				return nil, err
			}
			data := scalesResp.GetData()
			for _, s := range data {
				if s == nil || !s.GetActive() {
					continue
				}
				resp.ScoreScales = append(resp.ScoreScales, FormPickerOption{ID: s.GetId(), Name: s.GetName()})
			}
			if len(data) < formPickerPageLimit {
				break
			}
		}
		sort.SliceStable(resp.ScoreScales, func(i, j int) bool { return resp.ScoreScales[i].Name < resp.ScoreScales[j].Name })
	}
	return resp, nil
}
