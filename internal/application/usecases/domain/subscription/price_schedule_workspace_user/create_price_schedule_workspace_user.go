package price_schedule_workspace_user

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
)

type CreatePriceScheduleWorkspaceUserRepositories struct {
	PriceScheduleWorkspaceUser pb.PriceScheduleWorkspaceUserDomainServiceServer
}

type CreatePriceScheduleWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreatePriceScheduleWorkspaceUserUseCase struct {
	repositories CreatePriceScheduleWorkspaceUserRepositories
	services     CreatePriceScheduleWorkspaceUserServices
}

func NewCreatePriceScheduleWorkspaceUserUseCase(r CreatePriceScheduleWorkspaceUserRepositories, s CreatePriceScheduleWorkspaceUserServices) *CreatePriceScheduleWorkspaceUserUseCase {
	return &CreatePriceScheduleWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *CreatePriceScheduleWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.CreatePriceScheduleWorkspaceUserRequest) (*pb.CreatePriceScheduleWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PriceScheduleWorkspaceUser, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.PriceScheduleWorkspaceUser.CreatePriceScheduleWorkspaceUser(ctx, req)
}

func (uc *CreatePriceScheduleWorkspaceUserUseCase) enrich(data *pb.PriceScheduleWorkspaceUser) {
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
