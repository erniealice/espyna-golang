// Package evaluation_template holds the CRUD + lifecycle (Activate/Deprecate) use
// cases for the workspace-scoped evaluation_template aggregate.
package evaluation_template

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
	templateitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template_item"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// Repositories groups all repository dependencies. EvaluationTemplateItem +
// OutcomeCriteria back the Activate non-numeric-weight guard.
type Repositories struct {
	EvaluationTemplate     pb.EvaluationTemplateDomainServiceServer
	EvaluationTemplateItem templateitempb.EvaluationTemplateItemDomainServiceServer
	OutcomeCriteria        outcomecriteriapb.OutcomeCriteriaDomainServiceServer
}

// Services groups all business service dependencies.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the evaluation_template use cases.
type UseCases struct {
	Create      *CreateUseCase
	Read        *ReadUseCase
	Update      *UpdateUseCase
	Delete      *DeleteUseCase
	List        *ListUseCase
	GetListPage *GetListPageDataUseCase
	GetItemPage *GetItemPageDataUseCase
	Activate    *ActivateUseCase
	Deprecate   *DeprecateUseCase
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
		Activate:    &ActivateUseCase{r: r, s: s},
		Deprecate:   &DeprecateUseCase{r: r, s: s},
	}
}

type CreateUseCase struct {
	r Repositories
	s Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateEvaluationTemplateRequest) (*pb.CreateEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.name_required", "Name is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.s.IDGenerator.GenerateID()
	}
	// New templates start as DRAFT (never NULL — single lifecycle source).
	if req.Data.Status == pb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_UNSPECIFIED {
		req.Data.Status = pb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DRAFT
	}
	if req.Data.Version == 0 {
		req.Data.Version = 1
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationTemplate.CreateEvaluationTemplate(ctx, req)
}

type ReadUseCase struct {
	r Repositories
	s Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadEvaluationTemplateRequest) (*pb.ReadEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationTemplate.ReadEvaluationTemplate(ctx, req)
}

type UpdateUseCase struct {
	r Repositories
	s Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateEvaluationTemplateRequest) (*pb.UpdateEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.id_required", "ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationTemplate.UpdateEvaluationTemplate(ctx, req)
}

type DeleteUseCase struct {
	r Repositories
	s Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteEvaluationTemplateRequest) (*pb.DeleteEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationTemplate.DeleteEvaluationTemplate(ctx, req)
}

type ListUseCase struct {
	r Repositories
	s Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListEvaluationTemplatesRequest) (*pb.ListEvaluationTemplatesResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplate.ListEvaluationTemplates(ctx, req)
}

type GetListPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetListPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationTemplateListPageDataRequest) (*pb.GetEvaluationTemplateListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplate.GetEvaluationTemplateListPageData(ctx, req)
}

type GetItemPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationTemplateItemPageDataRequest) (*pb.GetEvaluationTemplateItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.r.EvaluationTemplate.GetEvaluationTemplateItemPageData(ctx, req)
}

// ActivateRequest / DeprecateRequest are Go-shaped state-transition inputs.
type ActivateRequest struct{ TemplateID string }
type DeprecateRequest struct{ TemplateID string }

// ActivateUseCase moves a template DRAFT→ACTIVE. It REJECTS any item whose linked
// OutcomeCriteria.criteria_type ∉ {numeric_range, numeric_score} while carrying a
// non-zero weight (weight_override OR the criterion's own weight) — so a weighted
// non-numeric dimension can never enter a scorable rubric (belt-and-suspenders
// with ComputeEvaluationScore excluding non-numeric types).
type ActivateUseCase struct {
	r Repositories
	s Services
}

func (uc *ActivateUseCase) Execute(ctx context.Context, req *ActivateRequest) (*pb.UpdateEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.TemplateID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.id_required", "ID is required [DEFAULT]"))
	}
	read, err := uc.r.EvaluationTemplate.ReadEvaluationTemplate(ctx, &pb.ReadEvaluationTemplateRequest{Data: &pb.EvaluationTemplate{Id: req.TemplateID}})
	if err != nil {
		return nil, err
	}
	if read == nil || len(read.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.errors.not_found", "Template not found [DEFAULT]"))
	}
	tmpl := read.Data[0]
	if tmpl.Status != pb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DRAFT {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.errors.not_draft", "Only a draft template can be activated [DEFAULT]"))
	}

	if err := uc.assertNoWeightedNonNumeric(ctx, req.TemplateID); err != nil {
		return nil, err
	}

	now := time.Now()
	tmpl.Status = pb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_ACTIVE
	tmpl.Active = true
	tmpl.DateModified = &[]int64{now.UnixMilli()}[0]
	tmpl.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationTemplate.UpdateEvaluationTemplate(ctx, &pb.UpdateEvaluationTemplateRequest{Data: tmpl})
}

// assertNoWeightedNonNumeric rejects activation if any template item is weighted
// AND its linked OutcomeCriteria is non-numeric.
func (uc *ActivateUseCase) assertNoWeightedNonNumeric(ctx context.Context, templateID string) error {
	if uc.r.EvaluationTemplateItem == nil || uc.r.OutcomeCriteria == nil {
		return nil
	}
	itemsResp, err := uc.r.EvaluationTemplateItem.ListEvaluationTemplateItems(ctx, &templateitempb.ListEvaluationTemplateItemsRequest{})
	if err != nil {
		return err
	}
	if itemsResp == nil {
		return nil
	}
	for _, item := range itemsResp.Data {
		if item.EvaluationTemplateId != templateID {
			continue
		}
		ocResp, err := uc.r.OutcomeCriteria.ReadOutcomeCriteria(ctx, &outcomecriteriapb.ReadOutcomeCriteriaRequest{Data: &outcomecriteriapb.OutcomeCriteria{Id: item.OutcomeCriteriaId}})
		if err != nil {
			return err
		}
		if ocResp == nil || len(ocResp.Data) == 0 {
			continue
		}
		oc := ocResp.Data[0]
		// Effective weight: weight_override if set, else the criterion's weight.
		weight := oc.Weight
		if item.WeightOverride != nil {
			weight = *item.WeightOverride
		}
		if weight != 0 && !isNumericCriteriaType(oc.CriteriaType) {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.errors.weighted_non_numeric", "A weighted dimension must be numeric to activate this template [DEFAULT]"))
		}
	}
	return nil
}

func isNumericCriteriaType(t enumspb.CriteriaType) bool {
	return t == enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE ||
		t == enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE
}

// DeprecateUseCase moves a template ACTIVE→DEPRECATED (no longer pickable;
// existing DRAFTs still submittable).
type DeprecateUseCase struct {
	r Repositories
	s Services
}

func (uc *DeprecateUseCase) Execute(ctx context.Context, req *DeprecateRequest) (*pb.UpdateEvaluationTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationTemplate, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.TemplateID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.validation.id_required", "ID is required [DEFAULT]"))
	}
	read, err := uc.r.EvaluationTemplate.ReadEvaluationTemplate(ctx, &pb.ReadEvaluationTemplateRequest{Data: &pb.EvaluationTemplate{Id: req.TemplateID}})
	if err != nil {
		return nil, err
	}
	if read == nil || len(read.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_template.errors.not_found", "Template not found [DEFAULT]"))
	}
	tmpl := read.Data[0]
	now := time.Now()
	tmpl.Status = pb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DEPRECATED
	// active = (status NOT IN {DEPRECATED}); a deprecated template is inactive.
	tmpl.Active = false
	tmpl.DateModified = &[]int64{now.UnixMilli()}[0]
	tmpl.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationTemplate.UpdateEvaluationTemplate(ctx, &pb.UpdateEvaluationTemplateRequest{Data: tmpl})
}
