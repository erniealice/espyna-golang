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

type UpdatePriceScheduleWorkspaceUserRepositories struct {
	PriceScheduleWorkspaceUser pb.PriceScheduleWorkspaceUserDomainServiceServer
}

type UpdatePriceScheduleWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdatePriceScheduleWorkspaceUserUseCase struct {
	repositories UpdatePriceScheduleWorkspaceUserRepositories
	services     UpdatePriceScheduleWorkspaceUserServices
}

func NewUpdatePriceScheduleWorkspaceUserUseCase(r UpdatePriceScheduleWorkspaceUserRepositories, s UpdatePriceScheduleWorkspaceUserServices) *UpdatePriceScheduleWorkspaceUserUseCase {
	return &UpdatePriceScheduleWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *UpdatePriceScheduleWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.UpdatePriceScheduleWorkspaceUserRequest) (*pb.UpdatePriceScheduleWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PriceScheduleWorkspaceUser, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.PriceScheduleWorkspaceUser.UpdatePriceScheduleWorkspaceUser(ctx, req)
}
