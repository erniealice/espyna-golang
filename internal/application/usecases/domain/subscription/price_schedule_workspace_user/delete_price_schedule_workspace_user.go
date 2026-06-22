package price_schedule_workspace_user

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
)

type DeletePriceScheduleWorkspaceUserRepositories struct {
	PriceScheduleWorkspaceUser pb.PriceScheduleWorkspaceUserDomainServiceServer
}

type DeletePriceScheduleWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeletePriceScheduleWorkspaceUserUseCase struct {
	repositories DeletePriceScheduleWorkspaceUserRepositories
	services     DeletePriceScheduleWorkspaceUserServices
}

func NewDeletePriceScheduleWorkspaceUserUseCase(r DeletePriceScheduleWorkspaceUserRepositories, s DeletePriceScheduleWorkspaceUserServices) *DeletePriceScheduleWorkspaceUserUseCase {
	return &DeletePriceScheduleWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *DeletePriceScheduleWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.DeletePriceScheduleWorkspaceUserRequest) (*pb.DeletePriceScheduleWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PriceScheduleWorkspaceUser, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PriceScheduleWorkspaceUser.DeletePriceScheduleWorkspaceUser(ctx, req)
}
