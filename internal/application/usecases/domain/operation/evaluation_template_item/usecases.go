// Package evaluation_template_item holds the CRUD use cases for the
// evaluation_template_item child aggregate (workspace_id copied from the parent
// template at create).
package evaluation_template_item

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template_item"
)

// Repositories groups all repository dependencies. EvaluationTemplate is the
// parent for the workspace_id copy + validation.
type Repositories struct {
	EvaluationTemplateItem pb.EvaluationTemplateItemDomainServiceServer
	EvaluationTemplate     templatepb.EvaluationTemplateDomainServiceServer
}

// Services groups all business service dependencies.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the evaluation_template_item use cases.
type UseCases struct {
	Create      *CreateUseCase
	Read        *ReadUseCase
	Update      *UpdateUseCase
	Delete      *DeleteUseCase
	List        *ListUseCase
	GetListPage *GetListPageDataUseCase
	GetItemPage *GetItemPageDataUseCase
}

func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		Create:      &CreateUseCase{r: r, s: s},
		Read:        &ReadUseCase{r: r, s: s},
		Update:      &UpdateUseCase{r: r, s: s},
		Delete:      &DeleteUseCase{r: r, s: s},
		List:        &ListUseCase{r: r, s: s},
		GetListPage: &GetListPageDataUseCase{r: r, s: s},
		GetItemPage: &GetItemPageDataUseCase{r: r, s: s},
	}
}

type CreateUseCase struct {
	r Repositories
	s Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateEvaluationTemplateItemRequest) (*pb.CreateEvaluationTemplateItemResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.EvaluationTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.template_id_required", "Template ID is required [DEFAULT]"))
	}
	if req.Data.OutcomeCriteriaId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.criteria_id_required", "Outcome criteria ID is required [DEFAULT]"))
	}
	// Copy workspace_id from the parent template (single-write boundary).
	if uc.r.EvaluationTemplate != nil {
		parent, err := uc.r.EvaluationTemplate.ReadEvaluationTemplate(ctx, &templatepb.ReadEvaluationTemplateRequest{Data: &templatepb.EvaluationTemplate{Id: req.Data.EvaluationTemplateId}})
		if err != nil {
			return nil, err
		}
		if parent == nil || len(parent.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.errors.parent_not_found", "Parent template not found [DEFAULT]"))
		}
		req.Data.WorkspaceId = parent.Data[0].WorkspaceId
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.s.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationTemplateItem.CreateEvaluationTemplateItem(ctx, req)
}

type ReadUseCase struct {
	r Repositories
	s Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadEvaluationTemplateItemRequest) (*pb.ReadEvaluationTemplateItemResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationTemplateItem.ReadEvaluationTemplateItem(ctx, req)
}

type UpdateUseCase struct {
	r Repositories
	s Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateEvaluationTemplateItemRequest) (*pb.UpdateEvaluationTemplateItemResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationTemplateItem.UpdateEvaluationTemplateItem(ctx, req)
}

type DeleteUseCase struct {
	r Repositories
	s Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteEvaluationTemplateItemRequest) (*pb.DeleteEvaluationTemplateItemResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template_item.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationTemplateItem.DeleteEvaluationTemplateItem(ctx, req)
}

type ListUseCase struct {
	r Repositories
	s Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListEvaluationTemplateItemsRequest) (*pb.ListEvaluationTemplateItemsResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplateItem.ListEvaluationTemplateItems(ctx, req)
}

type GetListPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetListPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationTemplateItemListPageDataRequest) (*pb.GetEvaluationTemplateItemListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplateItem.GetEvaluationTemplateItemListPageData(ctx, req)
}

type GetItemPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationTemplateItemItemPageDataRequest) (*pb.GetEvaluationTemplateItemItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplateItem, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplateItem.GetEvaluationTemplateItemItemPageData(ctx, req)
}
