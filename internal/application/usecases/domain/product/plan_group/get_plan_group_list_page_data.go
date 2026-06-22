package plan_group

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group"
)

type GetPlanGroupListPageDataRepositories struct {
	PlanGroup pb.PlanGroupDomainServiceServer
}

type GetPlanGroupListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetPlanGroupListPageDataUseCase struct {
	repositories GetPlanGroupListPageDataRepositories
	services     GetPlanGroupListPageDataServices
}

func NewGetPlanGroupListPageDataUseCase(r GetPlanGroupListPageDataRepositories, s GetPlanGroupListPageDataServices) *GetPlanGroupListPageDataUseCase {
	return &GetPlanGroupListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetPlanGroupListPageDataUseCase) Execute(ctx context.Context, req *pb.GetPlanGroupListPageDataRequest) (*pb.GetPlanGroupListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroup, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroup.GetPlanGroupListPageData(ctx, req)
}
