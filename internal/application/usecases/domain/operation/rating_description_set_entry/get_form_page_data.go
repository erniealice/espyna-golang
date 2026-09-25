package rating_description_set_entry

import (
	"context"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	scalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// entryPickerPageLimit/entryPickerMaxPages bound the offset-pagination loops
// below (codex-review-impl4.out.md round-3 disposition #2: a single unpaged
// call silently truncated criteria/bands at the adapter's 100-row default
// cap; mirrors the identical loop in
// rating_description_set/get_list_summary_page_data.go).
const (
	entryPickerPageLimit = 100
	entryPickerMaxPages  = 50
)

func entryPickerPagination(page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  entryPickerPageLimit,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}

func entryPickerIDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// FormPickerOption is one criterion <select> entry for a
// rating_description_set_entry drawer.
type FormPickerOption struct {
	ID   string
	Name string
}

// BandFormPickerOption is one level (score_scale_band) row/entry. See the
// package doc comment on GetRatingDescriptionSetEntryFormPageDataRequest for
// why ScoreScaleId/BandRole are carried through unfiltered.
type BandFormPickerOption struct {
	ID            string
	Name          string
	ScoreScaleId  string
	BandRole      string
	SequenceOrder int32
}

// GetRatingDescriptionSetEntryFormPageDataRequest has no fields — every
// picker below is workspace-wide; scale-scoping of the returned bands is
// left to the caller (BandFormPickerOption.ScoreScaleId).
type GetRatingDescriptionSetEntryFormPageDataRequest struct{}

// GetRatingDescriptionSetEntryFormPageDataResponse carries every picker
// option the entry drawer needs.
type GetRatingDescriptionSetEntryFormPageDataResponse struct {
	Criteria []FormPickerOption
	Bands    []BandFormPickerOption
}

type GetRatingDescriptionSetEntryFormPageDataRepositories struct {
	// OutcomeCriteria/ScoreScaleBand — picker-only cross-domain reads, read
	// directly here (not through their own gated List*UseCase) so this page
	// data is authorized once, under rating_description_set:update — not
	// under separate outcome_criteria:list / score_scale_band:list grants
	// (codex-review-impl2.out.md finding #6, schema-proposal.md §9.4).
	OutcomeCriteria criteriapb.OutcomeCriteriaDomainServiceServer
	ScoreScaleBand  scalebandpb.ScoreScaleBandDomainServiceServer
}

type GetRatingDescriptionSetEntryFormPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetEntryFormPageDataUseCase struct {
	repositories GetRatingDescriptionSetEntryFormPageDataRepositories
	services     GetRatingDescriptionSetEntryFormPageDataServices
}

func NewGetRatingDescriptionSetEntryFormPageDataUseCase(r GetRatingDescriptionSetEntryFormPageDataRepositories, s GetRatingDescriptionSetEntryFormPageDataServices) *GetRatingDescriptionSetEntryFormPageDataUseCase {
	return &GetRatingDescriptionSetEntryFormPageDataUseCase{repositories: r, services: s}
}

// Execute authorizes on rating_description_set:read. Used ONLY by the set
// detail page's READ-ONLY Descriptors matrix (codex-review-impl2.out.md
// finding #6's live confirmation: "the rating_description_set detail
// matrix's band ROWS never render"). The Add/Edit entry DRAWER uses the
// SIBLING GetRatingDescriptionSetEntryDrawerFormPageDataUseCase below,
// gated on :update instead — codex-review-impl3.out.md round-2 disposition
// #11: gating the drawer's picker on :read let a read-only-only viewer
// (Section Template Manager: set list/read, nothing else) reach the
// mutating drawer, while an update-only editor without a separate :read
// grant could not.
func (uc *GetRatingDescriptionSetEntryFormPageDataUseCase) Execute(ctx context.Context, _ *GetRatingDescriptionSetEntryFormPageDataRequest) (*GetRatingDescriptionSetEntryFormPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	return fetchEntryFormPickerData(ctx, uc.repositories)
}

// EntrySnapshot is the pre-fill shape for an existing entry, returned by the
// Drawer page-data variant's Edit path (codex-review-impl4.out.md
// round-3/round-4 "Update-only drawer" finding).
type EntrySnapshot struct {
	ID                     string
	RatingDescriptionSetId string
	OutcomeCriteriaId      string
	ScoreScaleBandId       string
	Description            string
}

// GetRatingDescriptionSetEntryDrawerFormPageDataRequest carries the Add
// drawer's parent set id and/or the Edit drawer's entry id. Both are
// optional and either may resolve the other: RatingDescriptionSetId alone
// scopes the Level picker (Add); EntryId alone loads the entry AND (via its
// own RatingDescriptionSetId) scopes the Level picker (Edit); both together
// is also valid (Edit passing its known parent set explicitly).
type GetRatingDescriptionSetEntryDrawerFormPageDataRequest struct {
	RatingDescriptionSetId string
	EntryId                string
}

// GetRatingDescriptionSetEntryDrawerFormPageDataResponse carries the entry
// drawer's picker options plus, when resolvable, the parent set's
// score_scale_id and (for Edit) the entry itself — so the fayna Add/Edit GET
// handlers need no separate rating_description_set:read-gated
// ReadRatingDescriptionSet / ReadRatingDescriptionSetEntry call
// (codex-review-impl4.out.md "Update-only drawer": an update-only editor
// without a separate :read grant must still be able to open both Add and
// Edit; a read-only-only viewer must not reach either).
type GetRatingDescriptionSetEntryDrawerFormPageDataResponse struct {
	Criteria     []FormPickerOption
	Bands        []BandFormPickerOption
	ScoreScaleId string
	// Entry is nil unless EntryId was supplied AND found. A supplied EntryId
	// that does not resolve to a row is reported as a nil Entry with a nil
	// error (absence, not a technical failure) — the caller treats a nil
	// Entry as Not Found. A genuine read failure is still returned as an
	// error.
	Entry *EntrySnapshot
}

// GetRatingDescriptionSetEntryDrawerFormPageDataUseCase backs the Add/Edit
// entry DRAWER specifically. Same picker data as the sibling use case
// above, but authorized on rating_description_set:update — NOT :read — so
// an update-only editor (who may not separately hold :read) can still open
// the drawer, and a read-only viewer (list/read only, no update) can no
// longer open it even though the read-only matrix keeps working via the
// sibling use case (codex-review-impl3.out.md round-2 disposition #11).
//
// The entry read (Edit) and the parent-set scale lookup (Add + Edit) are
// ALSO performed here, directly via their repository interfaces, bypassing
// rating_description_set_entry's own :read gate and rating_description_set's
// own :read gate respectively (codex-review-impl4.out.md round-3/round-4
// "Update-only drawer" finding: those separate :read-gated reads left an
// update-only editor unable to open Edit at all, and unable to see the Level
// picker on Add).
type GetRatingDescriptionSetEntryDrawerFormPageDataUseCase struct {
	repositories GetRatingDescriptionSetEntryFormPageDataRepositories
	// RatingDescriptionSet/RatingDescriptionSetEntry — parent-authorized
	// direct reads, optional/nil-safe (a nil repository degrades that one
	// enrichment to its zero value rather than failing the whole call).
	ratingDescriptionSet      setpb.RatingDescriptionSetDomainServiceServer
	ratingDescriptionSetEntry entrypb.RatingDescriptionSetEntryDomainServiceServer
	services                  GetRatingDescriptionSetEntryFormPageDataServices
}

func NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(r GetRatingDescriptionSetEntryFormPageDataRepositories, s GetRatingDescriptionSetEntryFormPageDataServices) *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase {
	return &GetRatingDescriptionSetEntryDrawerFormPageDataUseCase{repositories: r, services: s}
}

// WithParentReads attaches the parent rating_description_set and sibling
// rating_description_set_entry repositories used to resolve the Edit
// drawer's entry and the Level picker's scale scoping — both under this use
// case's own :update authorization, never their own entities' :read gate.
// Returns the same use case (builder-style) so wiring stays a one-liner in
// usecases.go.
func (uc *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase) WithParentReads(set setpb.RatingDescriptionSetDomainServiceServer, entry entrypb.RatingDescriptionSetEntryDomainServiceServer) *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase {
	uc.ratingDescriptionSet = set
	uc.ratingDescriptionSetEntry = entry
	return uc
}

func (uc *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase) Execute(ctx context.Context, req *GetRatingDescriptionSetEntryDrawerFormPageDataRequest) (*GetRatingDescriptionSetEntryDrawerFormPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	picker, err := fetchEntryFormPickerData(ctx, uc.repositories)
	if err != nil {
		return nil, err
	}
	resp := &GetRatingDescriptionSetEntryDrawerFormPageDataResponse{Criteria: picker.Criteria, Bands: picker.Bands}

	setID := ""
	entryID := ""
	if req != nil {
		setID = req.RatingDescriptionSetId
		entryID = req.EntryId
	}

	if entryID != "" && uc.ratingDescriptionSetEntry != nil {
		entryResp, eErr := uc.ratingDescriptionSetEntry.ReadRatingDescriptionSetEntry(ctx, &entrypb.ReadRatingDescriptionSetEntryRequest{
			Data: &entrypb.RatingDescriptionSetEntry{Id: entryID},
		})
		if eErr != nil {
			return nil, eErr
		}
		rows := entryResp.GetData()
		if len(rows) > 0 && rows[0] != nil {
			rec := rows[0]
			resp.Entry = &EntrySnapshot{
				ID:                     rec.GetId(),
				RatingDescriptionSetId: rec.GetRatingDescriptionSetId(),
				OutcomeCriteriaId:      rec.GetOutcomeCriteriaId(),
				ScoreScaleBandId:       rec.GetScoreScaleBandId(),
				Description:            rec.GetDescription(),
			}
			if setID == "" {
				setID = resp.Entry.RatingDescriptionSetId
			}
		}
		// len(rows) == 0: resp.Entry stays nil, no error — absence, not a
		// technical failure (caller treats a nil Entry as Not Found).
	}

	if setID != "" && uc.ratingDescriptionSet != nil {
		setResp, sErr := uc.ratingDescriptionSet.ReadRatingDescriptionSet(ctx, &setpb.ReadRatingDescriptionSetRequest{
			Data: &setpb.RatingDescriptionSet{Id: setID},
		})
		if sErr == nil && setResp != nil && len(setResp.GetData()) > 0 && setResp.GetData()[0] != nil {
			resp.ScoreScaleId = setResp.GetData()[0].GetScoreScaleId()
		}
		// A failed/absent parent-set lookup degrades ScoreScaleId to "" (the
		// Level picker then falls back to the raw-id text input) — it never
		// fails the whole drawer, same convention as the pre-existing
		// scoreScaleIDForSet fallback it replaces.
	}

	return resp, nil
}

// ExecuteForIDs is a plain-string wrapper around Execute so fayna's
// engineblock.go can call it without naming this package's request type
// across the module boundary (mirrors the established
// rating_description_set_product_plan.ExecuteForScheduleID workaround).
func (uc *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase) ExecuteForIDs(ctx context.Context, ratingDescriptionSetID, entryID string) (*GetRatingDescriptionSetEntryDrawerFormPageDataResponse, error) {
	return uc.Execute(ctx, &GetRatingDescriptionSetEntryDrawerFormPageDataRequest{RatingDescriptionSetId: ratingDescriptionSetID, EntryId: entryID})
}

// fetchEntryFormPickerData reads criteria and bands (workspace-wide, all
// scales) directly via their repository interfaces — shared by both
// authorization variants above. Pages through ALL results (no silent cap —
// codex-review-impl4.out.md round-3 disposition #2), since a workspace can
// hold more than one adapter page (100 rows) of active criteria or bands.
func fetchEntryFormPickerData(ctx context.Context, repos GetRatingDescriptionSetEntryFormPageDataRepositories) (*GetRatingDescriptionSetEntryFormPageDataResponse, error) {
	resp := &GetRatingDescriptionSetEntryFormPageDataResponse{}

	if repos.OutcomeCriteria != nil {
		for page := int32(1); page <= entryPickerMaxPages; page++ {
			criteriaResp, err := repos.OutcomeCriteria.ListOutcomeCriterias(ctx, &criteriapb.ListOutcomeCriteriasRequest{Sort: entryPickerIDSort(), Pagination: entryPickerPagination(page)})
			if err != nil {
				return nil, err
			}
			data := criteriaResp.GetData()
			for _, c := range data {
				if c == nil || !c.GetActive() {
					continue
				}
				resp.Criteria = append(resp.Criteria, FormPickerOption{ID: c.GetId(), Name: c.GetName()})
			}
			if len(data) < entryPickerPageLimit {
				break
			}
		}
		sort.SliceStable(resp.Criteria, func(i, j int) bool { return resp.Criteria[i].Name < resp.Criteria[j].Name })
	}

	if repos.ScoreScaleBand != nil {
		for page := int32(1); page <= entryPickerMaxPages; page++ {
			bandsResp, err := repos.ScoreScaleBand.ListScoreScaleBands(ctx, &scalebandpb.ListScoreScaleBandsRequest{Sort: entryPickerIDSort(), Pagination: entryPickerPagination(page)})
			if err != nil {
				return nil, err
			}
			data := bandsResp.GetData()
			for _, b := range data {
				if b == nil || !b.GetActive() {
					continue
				}
				resp.Bands = append(resp.Bands, BandFormPickerOption{
					ID:            b.GetId(),
					Name:          b.GetOutputLabel(),
					ScoreScaleId:  b.GetScoreScaleId(),
					BandRole:      b.GetBandRole(),
					SequenceOrder: b.GetSequenceOrder(),
				})
			}
			if len(data) < entryPickerPageLimit {
				break
			}
		}
		sort.SliceStable(resp.Bands, func(i, j int) bool { return resp.Bands[i].SequenceOrder < resp.Bands[j].SequenceOrder })
	}

	return resp, nil
}
