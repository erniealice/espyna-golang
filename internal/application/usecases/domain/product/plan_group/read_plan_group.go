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

type ReadPlanGroupRepositories struct {
	PlanGroup pb.PlanGroupDomainServiceServer
}

type ReadPlanGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadPlanGroupUseCase struct {
	repositories ReadPlanGroupRepositories
	services     ReadPlanGroupServices
}

func NewReadPlanGroupUseCase(r ReadPlanGroupRepositories, s ReadPlanGroupServices) *ReadPlanGroupUseCase {
	return &ReadPlanGroupUseCase{repositories: r, services: s}
}

func (uc *ReadPlanGroupUseCase) Execute(ctx context.Context, req *pb.ReadPlanGroupRequest) (*pb.ReadPlanGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroup, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PlanGroup.ReadPlanGroup(ctx, req)
}
