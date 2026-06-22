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

type GetPriceScheduleWorkspaceUserListPageDataRepositories struct {
	PriceScheduleWorkspaceUser pb.PriceScheduleWorkspaceUserDomainServiceServer
}

type GetPriceScheduleWorkspaceUserListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetPriceScheduleWorkspaceUserListPageDataUseCase struct {
	repositories GetPriceScheduleWorkspaceUserListPageDataRepositories
	services     GetPriceScheduleWorkspaceUserListPageDataServices
}

func NewGetPriceScheduleWorkspaceUserListPageDataUseCase(r GetPriceScheduleWorkspaceUserListPageDataRepositories, s GetPriceScheduleWorkspaceUserListPageDataServices) *GetPriceScheduleWorkspaceUserListPageDataUseCase {
	return &GetPriceScheduleWorkspaceUserListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetPriceScheduleWorkspaceUserListPageDataUseCase) Execute(ctx context.Context, req *pb.GetPriceScheduleWorkspaceUserListPageDataRequest) (*pb.GetPriceScheduleWorkspaceUserListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PriceScheduleWorkspaceUser, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.PriceScheduleWorkspaceUser.GetPriceScheduleWorkspaceUserListPageData(ctx, req)
}
