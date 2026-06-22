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

type CreateLineWorkspaceUserRepositories struct {
	LineWorkspaceUser pb.LineWorkspaceUserDomainServiceServer
}

type CreateLineWorkspaceUserServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateLineWorkspaceUserUseCase struct {
	repositories CreateLineWorkspaceUserRepositories
	services     CreateLineWorkspaceUserServices
}

func NewCreateLineWorkspaceUserUseCase(r CreateLineWorkspaceUserRepositories, s CreateLineWorkspaceUserServices) *CreateLineWorkspaceUserUseCase {
	return &CreateLineWorkspaceUserUseCase{repositories: r, services: s}
}

func (uc *CreateLineWorkspaceUserUseCase) Execute(ctx context.Context, req *pb.CreateLineWorkspaceUserRequest) (*pb.CreateLineWorkspaceUserResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.LineWorkspaceUser, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line_workspace_user.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.LineWorkspaceUser.CreateLineWorkspaceUser(ctx, req)
}

func (uc *CreateLineWorkspaceUserUseCase) enrich(data *pb.LineWorkspaceUser) {
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
