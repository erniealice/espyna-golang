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

type CreatePlanGroupRepositories struct {
	PlanGroup pb.PlanGroupDomainServiceServer
}

type CreatePlanGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreatePlanGroupUseCase struct {
	repositories CreatePlanGroupRepositories
	services     CreatePlanGroupServices
}

func NewCreatePlanGroupUseCase(r CreatePlanGroupRepositories, s CreatePlanGroupServices) *CreatePlanGroupUseCase {
	return &CreatePlanGroupUseCase{repositories: r, services: s}
}

func (uc *CreatePlanGroupUseCase) Execute(ctx context.Context, req *pb.CreatePlanGroupRequest) (*pb.CreatePlanGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanGroup, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_group.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.PlanGroup.CreatePlanGroup(ctx, req)
}

func (uc *CreatePlanGroupUseCase) enrich(data *pb.PlanGroup) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
