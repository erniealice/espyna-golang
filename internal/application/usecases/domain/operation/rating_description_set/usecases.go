// Package rating_description_set is the entity use-case group for
// RatingDescriptionSet (docs/plan/20260925-criterion-descriptors-by-program-year,
// modeled on the score_scale package). Plain CRUD (Create/Read/List/GetListPageData
// /GetItemPageData) mirrors score_scale exactly; Update/Delete additionally
// enforce the DRAFT-only lifecycle rule (schema-proposal.md §9.2); Publish/Deprecate
// are the two status-transition use cases.
//
// Deferred (logged, not implemented this wave — see W3-ESPYNA.done):
// CreateRatingDescriptionSetVersion, the audited-ops wrapper.
package rating_description_set

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	scalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// UseCases aggregates the rating_description_set CRUD + lifecycle + page-data
// use cases.
type UseCases struct {
	CreateRatingDescriptionSet          *CreateRatingDescriptionSetUseCase
	ReadRatingDescriptionSet            *ReadRatingDescriptionSetUseCase
	UpdateRatingDescriptionSet          *UpdateRatingDescriptionSetUseCase
	DeleteRatingDescriptionSet          *DeleteRatingDescriptionSetUseCase
	ListRatingDescriptionSets           *ListRatingDescriptionSetsUseCase
	GetRatingDescriptionSetListPageData *GetRatingDescriptionSetListPageDataUseCase
	GetRatingDescriptionSetItemPageData *GetRatingDescriptionSetItemPageDataUseCase
	PublishRatingDescriptionSet         *PublishRatingDescriptionSetUseCase
	DeprecateRatingDescriptionSet       *DeprecateRatingDescriptionSetUseCase
	// GetRatingDescriptionSetFormPageData — the Add-set drawer's score_scale
	// picker, authorized under rating_description_set:create (no separate
	// score_scale:list grant — codex-review-impl2.out.md finding #6,
	// schema-proposal.md §9.4).
	GetRatingDescriptionSetFormPageData *GetRatingDescriptionSetFormPageDataUseCase
	// GetRatingDescriptionSetListSummaryPageData — the LIST page's rows,
	// scale name, and entry/link counts, all authorized ONCE under
	// rating_description_set:list (codex-review-impl3.out.md finding #1 —
	// the list page must not separately require score_scale:list /
	// rating_description_set_product_plan:list).
	GetRatingDescriptionSetListSummaryPageData *GetRatingDescriptionSetListSummaryPageDataUseCase
}

// Repositories groups the primary repository dependency plus the two
// cross-entity reads needed by the lifecycle guards (Publish's "≥1 entry"
// check; Delete's "no links, any status" check).
type Repositories struct {
	RatingDescriptionSet            pb.RatingDescriptionSetDomainServiceServer
	RatingDescriptionSetEntry       entrypb.RatingDescriptionSetEntryDomainServiceServer
	RatingDescriptionSetProductPlan linkpb.RatingDescriptionSetProductPlanDomainServiceServer
	// ScoreScale — picker-only cross-domain read for
	// GetRatingDescriptionSetFormPageData (finding #6). Optional: nil-safe,
	// the picker use case degrades to an empty option list.
	ScoreScale scalepb.ScoreScaleDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the rating_description_set use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.RatingDescriptionSet
	return &UseCases{
		CreateRatingDescriptionSet: NewCreateRatingDescriptionSetUseCase(
			CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, CreateRatingDescriptionSetServices(s)),
		ReadRatingDescriptionSet: NewReadRatingDescriptionSetUseCase(
			ReadRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, ReadRatingDescriptionSetServices(s)),
		UpdateRatingDescriptionSet: NewUpdateRatingDescriptionSetUseCase(
			UpdateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, UpdateRatingDescriptionSetServices(s)),
		DeleteRatingDescriptionSet: NewDeleteRatingDescriptionSetUseCase(
			DeleteRatingDescriptionSetRepositories{RatingDescriptionSet: repo, RatingDescriptionSetProductPlan: r.RatingDescriptionSetProductPlan},
			DeleteRatingDescriptionSetServices(s)),
		ListRatingDescriptionSets: NewListRatingDescriptionSetsUseCase(
			ListRatingDescriptionSetsRepositories{RatingDescriptionSet: repo}, ListRatingDescriptionSetsServices(s)),
		GetRatingDescriptionSetListPageData: NewGetRatingDescriptionSetListPageDataUseCase(
			GetRatingDescriptionSetListPageDataRepositories{RatingDescriptionSet: repo}, GetRatingDescriptionSetListPageDataServices(s)),
		GetRatingDescriptionSetItemPageData: NewGetRatingDescriptionSetItemPageDataUseCase(
			GetRatingDescriptionSetItemPageDataRepositories{RatingDescriptionSet: repo}, GetRatingDescriptionSetItemPageDataServices(s)),
		PublishRatingDescriptionSet: NewPublishRatingDescriptionSetUseCase(
			repo, r.RatingDescriptionSetEntry, PublishRatingDescriptionSetServices(s)),
		DeprecateRatingDescriptionSet: NewDeprecateRatingDescriptionSetUseCase(
			repo, DeprecateRatingDescriptionSetServices(s)),
		GetRatingDescriptionSetFormPageData: NewGetRatingDescriptionSetFormPageDataUseCase(
			GetRatingDescriptionSetFormPageDataRepositories{ScoreScale: r.ScoreScale},
			GetRatingDescriptionSetFormPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetRatingDescriptionSetListSummaryPageData: NewGetRatingDescriptionSetListSummaryPageDataUseCase(
			GetRatingDescriptionSetListSummaryPageDataRepositories{
				RatingDescriptionSet:            repo,
				ScoreScale:                      r.ScoreScale,
				RatingDescriptionSetEntry:       r.RatingDescriptionSetEntry,
				RatingDescriptionSetProductPlan: r.RatingDescriptionSetProductPlan,
			},
			GetRatingDescriptionSetListSummaryPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
	}
}
