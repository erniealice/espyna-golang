package plan_group

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group"
)

type UpdatePlanGroupRepositories struct {
	PlanGroup pb.PlanGroupDomainServiceServer
}

type UpdatePlanGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdatePlanGroupUseCase struct {
	repositories UpdatePlanGroupRepositories
	services     UpdatePlanGroupServices
}

func NewUpdatePlanGroupUseCase(r UpdatePlanGroupRepositories, s UpdatePlanGroupServices) *UpdatePlanGroupUseCase {
	return &UpdatePlanGroupUseCase{repositories: r, services: s}
}

func (uc *UpdatePlanGroupUseCase) Execute(ctx context.Context, req *pb.UpdatePlanGroupRequest) (*pb.UpdatePlanGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroup, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.PlanGroup.UpdatePlanGroup(ctx, req)
}
