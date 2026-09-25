// Package rating_description_set_entry is the entity use-case group for
// RatingDescriptionSetEntry (docs/plan/20260925-criterion-descriptors-by-program-year).
// Plain CRUD + List — the generated RatingDescriptionSetEntryDomainServiceServer
// interface has no page-data RPCs (unlike RatingDescriptionSet /
// RatingDescriptionSetProductPlan). The DRAFT-parent + band-role guards live in
// the postgres adapter itself (contrib/postgres/internal/adapter/operation/
// rating_description_set_entry.go), enforced regardless of caller — these use
// cases add only the standard authorization + request-shape checks, mirroring
// score_scale_band.
//
// Deferred (logged, not implemented this wave — see W3-ESPYNA.done):
// ImportRatingDescriptionSetEntries.
package rating_description_set_entry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	scalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// UseCases aggregates the rating_description_set_entry CRUD use cases.
type UseCases struct {
	CreateRatingDescriptionSetEntry *CreateRatingDescriptionSetEntryUseCase
	ReadRatingDescriptionSetEntry   *ReadRatingDescriptionSetEntryUseCase
	UpdateRatingDescriptionSetEntry *UpdateRatingDescriptionSetEntryUseCase
	DeleteRatingDescriptionSetEntry *DeleteRatingDescriptionSetEntryUseCase
	ListRatingDescriptionSetEntries *ListRatingDescriptionSetEntriesUseCase
	// GetRatingDescriptionSetEntryFormPageData — the set detail page's
	// READ-ONLY Descriptors matrix criteria/band source, authorized under
	// rating_description_set:read (no separate outcome_criteria:list /
	// score_scale_band:list grant — codex-review-impl2.out.md finding #6,
	// schema-proposal.md §9.4).
	GetRatingDescriptionSetEntryFormPageData *GetRatingDescriptionSetEntryFormPageDataUseCase
	// GetRatingDescriptionSetEntryDrawerFormPageData — the Add/Edit entry
	// DRAWER's criteria/band pickers specifically, authorized under
	// rating_description_set:update (codex-review-impl3.out.md round-2
	// disposition #11).
	GetRatingDescriptionSetEntryDrawerFormPageData *GetRatingDescriptionSetEntryDrawerFormPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	RatingDescriptionSetEntry pb.RatingDescriptionSetEntryDomainServiceServer
	// OutcomeCriteria/ScoreScaleBand — picker-only cross-domain reads for
	// GetRatingDescriptionSetEntryFormPageData (finding #6). Optional:
	// nil-safe, the picker use case degrades to empty option lists.
	OutcomeCriteria criteriapb.OutcomeCriteriaDomainServiceServer
	ScoreScaleBand  scalebandpb.ScoreScaleBandDomainServiceServer
	// RatingDescriptionSet — parent-authorized read used ONLY by
	// GetRatingDescriptionSetEntryDrawerFormPageData to resolve the Level
	// picker's scale scoping (codex-review-impl4.out.md "Update-only
	// drawer"). Optional: nil-safe, degrades ScoreScaleId to "".
	RatingDescriptionSet setpb.RatingDescriptionSetDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the rating_description_set_entry use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.RatingDescriptionSetEntry
	return &UseCases{
		CreateRatingDescriptionSetEntry: NewCreateRatingDescriptionSetEntryUseCase(
			CreateRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: repo}, CreateRatingDescriptionSetEntryServices(s)),
		ReadRatingDescriptionSetEntry: NewReadRatingDescriptionSetEntryUseCase(
			ReadRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: repo}, ReadRatingDescriptionSetEntryServices(s)),
		UpdateRatingDescriptionSetEntry: NewUpdateRatingDescriptionSetEntryUseCase(
			UpdateRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: repo}, UpdateRatingDescriptionSetEntryServices(s)),
		DeleteRatingDescriptionSetEntry: NewDeleteRatingDescriptionSetEntryUseCase(
			DeleteRatingDescriptionSetEntryRepositories{RatingDescriptionSetEntry: repo}, DeleteRatingDescriptionSetEntryServices(s)),
		ListRatingDescriptionSetEntries: NewListRatingDescriptionSetEntriesUseCase(
			ListRatingDescriptionSetEntriesRepositories{RatingDescriptionSetEntry: repo}, ListRatingDescriptionSetEntriesServices(s)),
		GetRatingDescriptionSetEntryFormPageData: NewGetRatingDescriptionSetEntryFormPageDataUseCase(
			GetRatingDescriptionSetEntryFormPageDataRepositories{
				OutcomeCriteria: r.OutcomeCriteria,
				ScoreScaleBand:  r.ScoreScaleBand,
			},
			GetRatingDescriptionSetEntryFormPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetRatingDescriptionSetEntryDrawerFormPageData: NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(
			GetRatingDescriptionSetEntryFormPageDataRepositories{
				OutcomeCriteria: r.OutcomeCriteria,
				ScoreScaleBand:  r.ScoreScaleBand,
			},
			GetRatingDescriptionSetEntryFormPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper},
		).WithParentReads(r.RatingDescriptionSet, repo),
	}
}
