package rating_description_set_product_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type RelinkRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// RelinkRatingDescriptionSetProductPlanUseCase performs the first-link /
// replacement transaction (schema-proposal.md §9.2, espyna-golang.md
// "Lifecycle and relink rules" — "one ExecuteInTransaction: authorize
// rating_description_set_product_plan:create/update; lock/compare current
// active link = expected_current_link_id; target set PUBLISHED + same
// workspace; deactivate current; insert new"). Authorization: create when no
// current link is expected, update when one is — both checked up front (the
// adapter's own compare is the authoritative race guard).
type RelinkRatingDescriptionSetProductPlanUseCase struct {
	repo            domainports.RatingDescriptionSetProductPlanRelinker
	repositoryError error
	services        RelinkRatingDescriptionSetProductPlanServices
}

func NewRelinkRatingDescriptionSetProductPlanUseCase(
	repository pb.RatingDescriptionSetProductPlanDomainServiceServer,
	services RelinkRatingDescriptionSetProductPlanServices,
) *RelinkRatingDescriptionSetProductPlanUseCase {
	relinker, ok := repository.(domainports.RatingDescriptionSetProductPlanRelinker)
	uc := &RelinkRatingDescriptionSetProductPlanUseCase{repo: relinker, services: services}
	if !ok {
		uc.repositoryError = fmt.Errorf("rating description set product plan repository %T does not support the conditional relink/unlink transaction", repository)
	}
	return uc
}

func (uc *RelinkRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.RelinkRatingDescriptionSetProductPlanRequest) (*pb.RelinkRatingDescriptionSetProductPlanResponse, error) {
	action := entityid.ActionCreate
	if req != nil && req.ExpectedCurrentLinkId != nil && *req.ExpectedCurrentLinkId != "" {
		action = entityid.ActionUpdate
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: action}); err != nil {
		return nil, err
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if req == nil || req.ProductPlanId == "" || req.PriceScheduleId == "" || req.RatingDescriptionSetId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.relink_fields_required", "[ERR-DEFAULT] product_plan_id, price_schedule_id and rating_description_set_id are required"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.errors.transactor_unavailable", "[ERR-DEFAULT] Relink requires transaction support"))
	}

	newLink := &pb.RatingDescriptionSetProductPlan{
		ProductPlanId:          req.ProductPlanId,
		PriceScheduleId:        req.PriceScheduleId,
		RatingDescriptionSetId: req.RatingDescriptionSetId,
	}
	if uc.services.IDGenerator != nil {
		newLink.Id = uc.services.IDGenerator.GenerateID()
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	newLink.DateCreated = &ms
	newLink.DateCreatedString = &s
	newLink.DateModified = &ms
	newLink.DateModifiedString = &s

	var result *pb.RatingDescriptionSetProductPlan
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		item, txErr := uc.repo.RelinkLocked(txCtx, newLink, req.ExpectedCurrentLinkId, req.Reason)
		if txErr != nil {
			return txErr
		}
		result = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.RelinkRatingDescriptionSetProductPlanResponse{Data: []*pb.RatingDescriptionSetProductPlan{result}, Success: true}, nil
}
