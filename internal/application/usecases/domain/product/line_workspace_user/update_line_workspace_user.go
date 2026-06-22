package line_workspace_user

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line_workspace_user"
)

type UpdateLineWorkspaceUserRepositories struct {
	LineWorkspaceUser pb.LineWorkspaceUserDomainServiceServer
}

type UpdateLineWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateLineWorkspaceUserUseCase struct {
	repositories UpdateLineWorkspaceUserRepositories
	services     UpdateLineWorkspaceUserServices
}

func NewUpdateLineWorkspaceUserUseCase(r UpdateLineWorkspaceUserRepositories, s UpdateLineWorkspaceUserServices) *UpdateLineWorkspaceUserUseCase {
	return &UpdateLineWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *UpdateLineWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.UpdateLineWorkspaceUserRequest) (*pb.UpdateLineWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.LineWorkspaceUser, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.LineWorkspaceUser.UpdateLineWorkspaceUser(ctx, req)
}
