package line

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
)

// LineRepositories groups all repository dependencies for line use cases.
type LineRepositories struct {
	Line linepb.LineDomainServiceServer
}

// LineServices groups all business service dependencies for line use cases.
type LineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all line-related use cases.
type UseCases struct {
	CreateLine *CreateLineUseCase
	ReadLine   *ReadLineUseCase
	UpdateLine *UpdateLineUseCase
	DeleteLine *DeleteLineUseCase
	ListLines  *ListLinesUseCase
}

// NewUseCases creates a new collection of line use cases.
func NewUseCases(repositories LineRepositories, services LineServices) *UseCases {
	createRepos := CreateLineRepositories{Line: repositories.Line}
	createServices := CreateLineServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLineRepositories{Line: repositories.Line}
	readServices := ReadLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateLineRepositories{Line: repositories.Line}
	updateServices := UpdateLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteLineRepositories{Line: repositories.Line}
	deleteServices := DeleteLineServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListLinesRepositories{Line: repositories.Line}
	listServices := ListLinesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateLine: NewCreateLineUseCase(createRepos, createServices),
		ReadLine:   NewReadLineUseCase(readRepos, readServices),
		UpdateLine: NewUpdateLineUseCase(updateRepos, updateServices),
		DeleteLine: NewDeleteLineUseCase(deleteRepos, deleteServices),
		ListLines:  NewListLinesUseCase(listRepos, listServices),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

type CreateLineRepositories struct {
	Line linepb.LineDomainServiceServer
}

type CreateLineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

type CreateLineUseCase struct {
	repositories CreateLineRepositories
	services     CreateLineServices
}

func NewCreateLineUseCase(repositories CreateLineRepositories, services CreateLineServices) *CreateLineUseCase {
	return &CreateLineUseCase{repositories: repositories, services: services}
}

func (uc *CreateLineUseCase) Execute(ctx context.Context, req *linepb.CreateLineRequest) (*linepb.CreateLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Line, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.validation.data_required", "Line data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.validation.name_required", "Name is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	resp, err := uc.repositories.Line.CreateLine(ctx, req)
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.errors.creation_failed", "Line creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

type ReadLineRepositories struct {
	Line linepb.LineDomainServiceServer
}

type ReadLineServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadLineUseCase struct {
	repositories ReadLineRepositories
	services     ReadLineServices
}

func NewReadLineUseCase(repositories ReadLineRepositories, services ReadLineServices) *ReadLineUseCase {
	return &ReadLineUseCase{repositories: repositories, services: services}
}

func (uc *ReadLineUseCase) Execute(ctx context.Context, req *linepb.ReadLineRequest) (*linepb.ReadLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Line, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.validation.id_required", "Line ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.Line.ReadLine(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.errors.not_found", "Line not found [DEFAULT]"))
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

type UpdateLineRepositories struct {
	Line linepb.LineDomainServiceServer
}

type UpdateLineServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateLineUseCase struct {
	repositories UpdateLineRepositories
	services     UpdateLineServices
}

func NewUpdateLineUseCase(repositories UpdateLineRepositories, services UpdateLineServices) *UpdateLineUseCase {
	return &UpdateLineUseCase{repositories: repositories, services: services}
}

func (uc *UpdateLineUseCase) Execute(ctx context.Context, req *linepb.UpdateLineRequest) (*linepb.UpdateLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Line, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.validation.id_required", "Line ID is required [DEFAULT]"))
	}
	readResp, err := uc.repositories.Line.ReadLine(ctx, &linepb.ReadLineRequest{Data: &linepb.Line{Id: req.Data.Id}})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.errors.not_found", "Line not found [DEFAULT]"))
	}
	existing := readResp.GetData()[0]
	if req.Data.Name == "" {
		req.Data.Name = existing.GetName()
	}
	if req.Data.Description == "" {
		req.Data.Description = existing.GetDescription()
	}
	req.Data.Active = existing.GetActive()
	now := time.Now()
	req.Data.DateCreated = existing.DateCreated
	req.Data.DateCreatedString = existing.DateCreatedString
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.Line.UpdateLine(ctx, req)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

type DeleteLineRepositories struct {
	Line linepb.LineDomainServiceServer
}

type DeleteLineServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteLineUseCase struct {
	repositories DeleteLineRepositories
	services     DeleteLineServices
}

func NewDeleteLineUseCase(repositories DeleteLineRepositories, services DeleteLineServices) *DeleteLineUseCase {
	return &DeleteLineUseCase{repositories: repositories, services: services}
}

func (uc *DeleteLineUseCase) Execute(ctx context.Context, req *linepb.DeleteLineRequest) (*linepb.DeleteLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Line, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "line.validation.id_required", "Line ID is required [DEFAULT]"))
	}
	return uc.repositories.Line.DeleteLine(ctx, req)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

type ListLinesRepositories struct {
	Line linepb.LineDomainServiceServer
}

type ListLinesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListLinesUseCase struct {
	repositories ListLinesRepositories
	services     ListLinesServices
}

func NewListLinesUseCase(repositories ListLinesRepositories, services ListLinesServices) *ListLinesUseCase {
	return &ListLinesUseCase{repositories: repositories, services: services}
}

func (uc *ListLinesUseCase) Execute(ctx context.Context, req *linepb.ListLinesRequest) (*linepb.ListLinesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Line, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		req = &linepb.ListLinesRequest{}
	}
	return uc.repositories.Line.ListLines(ctx, req)
}
