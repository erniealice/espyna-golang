package rating_description_set_product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type UnlinkRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UnlinkRatingDescriptionSetProductPlanUseCase deactivates one link; the
// offering becomes NO_LINK for the resolver.
type UnlinkRatingDescriptionSetProductPlanUseCase struct {
	repo            domainports.RatingDescriptionSetProductPlanRelinker
	repositoryError error
	services        UnlinkRatingDescriptionSetProductPlanServices
}

func NewUnlinkRatingDescriptionSetProductPlanUseCase(
	repository pb.RatingDescriptionSetProductPlanDomainServiceServer,
	services UnlinkRatingDescriptionSetProductPlanServices,
) *UnlinkRatingDescriptionSetProductPlanUseCase {
	relinker, ok := repository.(domainports.RatingDescriptionSetProductPlanRelinker)
	uc := &UnlinkRatingDescriptionSetProductPlanUseCase{repo: relinker, services: services}
	if !ok {
		uc.repositoryError = fmt.Errorf("rating description set product plan repository %T does not support the conditional relink/unlink transaction", repository)
	}
	return uc
}

func (uc *UnlinkRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.UnlinkRatingDescriptionSetProductPlanRequest) (*pb.UnlinkRatingDescriptionSetProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if req == nil || req.LinkId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.link_id_required", "[ERR-DEFAULT] link_id is required"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.errors.transactor_unavailable", "[ERR-DEFAULT] Unlink requires transaction support"))
	}
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		_, txErr := uc.repo.UnlinkLocked(txCtx, req.LinkId, req.Reason)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return &pb.UnlinkRatingDescriptionSetProductPlanResponse{Success: true}, nil
}
