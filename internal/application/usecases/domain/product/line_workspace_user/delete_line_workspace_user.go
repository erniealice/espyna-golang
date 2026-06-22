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

type DeleteLineWorkspaceUserRepositories struct {
	LineWorkspaceUser pb.LineWorkspaceUserDomainServiceServer
}

type DeleteLineWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteLineWorkspaceUserUseCase struct {
	repositories DeleteLineWorkspaceUserRepositories
	services     DeleteLineWorkspaceUserServices
}

func NewDeleteLineWorkspaceUserUseCase(r DeleteLineWorkspaceUserRepositories, s DeleteLineWorkspaceUserServices) *DeleteLineWorkspaceUserUseCase {
	return &DeleteLineWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *DeleteLineWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.DeleteLineWorkspaceUserRequest) (*pb.DeleteLineWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.LineWorkspaceUser, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line_workspace_user.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.LineWorkspaceUser.DeleteLineWorkspaceUser(ctx, req)
}
