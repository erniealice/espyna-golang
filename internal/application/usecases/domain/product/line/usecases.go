package line

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
)

// LineRepositories groups all repository dependencies for line use cases.
type LineRepositories struct {
	Line linepb.LineDomainServiceServer
}

// LineServices groups all business service dependencies for line use cases.
type LineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadLineRepositories{Line: repositories.Line}
	readServices := ReadLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateLineRepositories{Line: repositories.Line}
	updateServices := UpdateLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteLineRepositories{Line: repositories.Line}
	deleteServices := DeleteLineServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListLinesRepositories{Line: repositories.Line}
	listServices := ListLinesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type CreateLineUseCase struct {
	repositories CreateLineRepositories
	services     CreateLineServices
}

func NewCreateLineUseCase(repositories CreateLineRepositories, services CreateLineServices) *CreateLineUseCase {
	return &CreateLineUseCase{repositories: repositories, services: services}
}

func (uc *CreateLineUseCase) Execute(ctx context.Context, req *linepb.CreateLineRequest) (*linepb.CreateLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, ports.EntityLine, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.validation.data_required", "Line data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.validation.name_required", "Name is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	resp, err := uc.repositories.Line.CreateLine(ctx, req)
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.errors.creation_failed", "Line creation failed [DEFAULT]")
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadLineUseCase struct {
	repositories ReadLineRepositories
	services     ReadLineServices
}

func NewReadLineUseCase(repositories ReadLineRepositories, services ReadLineServices) *ReadLineUseCase {
	return &ReadLineUseCase{repositories: repositories, services: services}
}

func (uc *ReadLineUseCase) Execute(ctx context.Context, req *linepb.ReadLineRequest) (*linepb.ReadLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, ports.EntityLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.validation.id_required", "Line ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.Line.ReadLine(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.errors.not_found", "Line not found [DEFAULT]"))
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateLineUseCase struct {
	repositories UpdateLineRepositories
	services     UpdateLineServices
}

func NewUpdateLineUseCase(repositories UpdateLineRepositories, services UpdateLineServices) *UpdateLineUseCase {
	return &UpdateLineUseCase{repositories: repositories, services: services}
}

func (uc *UpdateLineUseCase) Execute(ctx context.Context, req *linepb.UpdateLineRequest) (*linepb.UpdateLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, ports.EntityLine, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.validation.id_required", "Line ID is required [DEFAULT]"))
	}
	readResp, err := uc.repositories.Line.ReadLine(ctx, &linepb.ReadLineRequest{Data: &linepb.Line{Id: req.Data.Id}})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.errors.not_found", "Line not found [DEFAULT]"))
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type DeleteLineUseCase struct {
	repositories DeleteLineRepositories
	services     DeleteLineServices
}

func NewDeleteLineUseCase(repositories DeleteLineRepositories, services DeleteLineServices) *DeleteLineUseCase {
	return &DeleteLineUseCase{repositories: repositories, services: services}
}

func (uc *DeleteLineUseCase) Execute(ctx context.Context, req *linepb.DeleteLineRequest) (*linepb.DeleteLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, ports.EntityLine, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "line.validation.id_required", "Line ID is required [DEFAULT]"))
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ListLinesUseCase struct {
	repositories ListLinesRepositories
	services     ListLinesServices
}

func NewListLinesUseCase(repositories ListLinesRepositories, services ListLinesServices) *ListLinesUseCase {
	return &ListLinesUseCase{repositories: repositories, services: services}
}

func (uc *ListLinesUseCase) Execute(ctx context.Context, req *linepb.ListLinesRequest) (*linepb.ListLinesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, ports.EntityLine, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		req = &linepb.ListLinesRequest{}
	}
	return uc.repositories.Line.ListLines(ctx, req)
}
