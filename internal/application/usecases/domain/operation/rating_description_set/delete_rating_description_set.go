package rating_description_set

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type DeleteRatingDescriptionSetRepositories struct {
	RatingDescriptionSet            pb.RatingDescriptionSetDomainServiceServer
	RatingDescriptionSetProductPlan linkpb.RatingDescriptionSetProductPlanDomainServiceServer
}

type DeleteRatingDescriptionSetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteRatingDescriptionSetUseCase struct {
	repositories DeleteRatingDescriptionSetRepositories
	services     DeleteRatingDescriptionSetServices
}

func NewDeleteRatingDescriptionSetUseCase(r DeleteRatingDescriptionSetRepositories, s DeleteRatingDescriptionSetServices) *DeleteRatingDescriptionSetUseCase {
	return &DeleteRatingDescriptionSetUseCase{repositories: r, services: s}
}

// Execute deletes a rating_description_set only while DRAFT AND only when it
// has no rating_description_set_product_plan links of ANY status (schema-
// proposal.md §9.2 "Update / delete set... delete only with no links (any
// status)", interfaces.md §4 "DeleteRatingDescriptionSet (DRAFT with no links
// only)"). A DRAFT set can never legitimately have a link (Relink requires
// the target PUBLISHED), so the no-links check also guards against deleting a
// set that was published, linked, then somehow reverted — belt-and-suspenders,
// since generic status transitions other than Publish/Deprecate are rejected.
func (uc *DeleteRatingDescriptionSetUseCase) Execute(ctx context.Context, req *pb.DeleteRatingDescriptionSetRequest) (*pb.DeleteRatingDescriptionSetResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.request_required", "Request is required [DEFAULT]"))
	}

	existingResp, err := uc.repositories.RatingDescriptionSet.ReadRatingDescriptionSet(ctx, &pb.ReadRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{Id: req.Data.Id}})
	if err != nil || existingResp == nil || len(existingResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.not_found", "[ERR-DEFAULT] Rating description set not found"))
	}
	existing := existingResp.Data[0]
	if existing.VersionStatus != enumspb.VersionStatus_VERSION_STATUS_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.set_not_draft", "SET_NOT_DRAFT: rating description set can only be deleted while DRAFT"))
	}

	if uc.repositories.RatingDescriptionSetProductPlan != nil {
		linkResp, err := uc.repositories.RatingDescriptionSetProductPlan.ListRatingDescriptionSetProductPlans(ctx, &linkpb.ListRatingDescriptionSetProductPlansRequest{
			Filters: equalsFilter("rating_description_set_id", req.Data.Id),
		})
		if err != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.link_check_failed", "[ERR-DEFAULT] Failed to check rating description set links"))
		}
		if linkResp != nil && len(linkResp.Data) > 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.has_links", "[ERR-DEFAULT] Rating description set has links (active or inactive) and cannot be deleted"))
		}
	}

	return uc.repositories.RatingDescriptionSet.DeleteRatingDescriptionSet(ctx, req)
}

// equalsFilter builds a single-field exact-match FilterRequest (case-sensitive
// so opaque ids match verbatim) — mirrors grade_compute's helper of the same
// name (package-local copy; not exported cross-package by that package).
func equalsFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{{
			Field: field,
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:         value,
					Operator:      commonpb.StringOperator_STRING_EQUALS,
					CaseSensitive: true,
				},
			},
		}},
	}
}
