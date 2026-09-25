package rating_description_set

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
)

type DeprecateRatingDescriptionSetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeprecateRatingDescriptionSetUseCase transitions a set PUBLISHED ->
// DEPRECATED. Existing links keep resolving (Q19) — this use case never
// touches rating_description_set_product_plan.
type DeprecateRatingDescriptionSetUseCase struct {
	repo            domainports.RatingDescriptionSetLifecycleRepository
	repositoryError error
	services        DeprecateRatingDescriptionSetServices
}

func NewDeprecateRatingDescriptionSetUseCase(
	repository pb.RatingDescriptionSetDomainServiceServer,
	services DeprecateRatingDescriptionSetServices,
) *DeprecateRatingDescriptionSetUseCase {
	lifecycle, ok := repository.(domainports.RatingDescriptionSetLifecycleRepository)
	uc := &DeprecateRatingDescriptionSetUseCase{repo: lifecycle, services: services}
	if !ok {
		uc.repositoryError = fmt.Errorf("rating description set repository %T does not support the conditional publish/deprecate transitions", repository)
	}
	return uc
}

func (uc *DeprecateRatingDescriptionSetUseCase) Execute(ctx context.Context, req *pb.DeprecateRatingDescriptionSetRequest) (*pb.DeprecateRatingDescriptionSetResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionDeprecate}); err != nil {
		return nil, err
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.id_required", "[ERR-DEFAULT] Rating description set ID is required"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.transactor_unavailable", "[ERR-DEFAULT] Deprecate requires transaction support"))
	}
	var deprecated *pb.RatingDescriptionSet
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		item, txErr := uc.repo.DeprecateRatingDescriptionSetIfPublished(txCtx, req.Id)
		if txErr != nil {
			return txErr
		}
		deprecated = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeprecateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{deprecated}, Success: true}, nil
}
