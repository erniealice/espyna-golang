package line_workspace_user

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line_workspace_user"
)

type ListLineWorkspaceUsersRepositories struct {
	LineWorkspaceUser pb.LineWorkspaceUserDomainServiceServer
}

type ListLineWorkspaceUsersServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListLineWorkspaceUsersUseCase struct {
	repositories ListLineWorkspaceUsersRepositories
	services     ListLineWorkspaceUsersServices
}

func NewListLineWorkspaceUsersUseCase(r ListLineWorkspaceUsersRepositories, s ListLineWorkspaceUsersServices) *ListLineWorkspaceUsersUseCase {
	return &ListLineWorkspaceUsersUseCase{repositories: r, services: s}
}

func (uc *ListLineWorkspaceUsersUseCase) Execute(ctx context.Context, req *pb.ListLineWorkspaceUsersRequest) (*pb.ListLineWorkspaceUsersResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.LineWorkspaceUser, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.LineWorkspaceUser.ListLineWorkspaceUsers(ctx, req)
}
