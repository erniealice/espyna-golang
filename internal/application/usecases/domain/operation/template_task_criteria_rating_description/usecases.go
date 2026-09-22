package template_task_criteria_rating_description

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	templatetaskcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria_rating_description"
)

type Repositories struct {
	TemplateTaskCriteriaRatingDescription pb.TemplateTaskCriteriaRatingDescriptionDomainServiceServer
	TemplateTaskCriteria                  templatetaskcriteriapb.TemplateTaskCriteriaDomainServiceServer
	OutcomeCriteria                       outcomecriteriapb.OutcomeCriteriaDomainServiceServer
	ScoreScaleBand                        scorescalebandpb.ScoreScaleBandDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases exposes the typed child persistence operations. The entity remains
// embedded configuration in the template-task-criteria authoring flow; it does
// not create a new top-level vertical route by itself.
type UseCases struct {
	CreateTemplateTaskCriteriaRatingDescription          *CreateUseCase
	ReadTemplateTaskCriteriaRatingDescription            *ReadUseCase
	UpdateTemplateTaskCriteriaRatingDescription          *UpdateUseCase
	DeleteTemplateTaskCriteriaRatingDescription          *DeleteUseCase
	ListTemplateTaskCriteriaRatingDescriptions           *ListUseCase
	GetTemplateTaskCriteriaRatingDescriptionListPageData *ListPageDataUseCase
	GetTemplateTaskCriteriaRatingDescriptionItemPageData *ItemPageDataUseCase
	ListByTemplateTaskCriteria                           *ListByTemplateTaskCriteriaUseCase
}

func NewUseCases(r Repositories, s Services) *UseCases {
	shared := shared{
		repo:                 r.TemplateTaskCriteriaRatingDescription,
		templateTaskCriteria: r.TemplateTaskCriteria,
		outcomeCriteria:      r.OutcomeCriteria,
		scoreScaleBand:       r.ScoreScaleBand,
		services:             s,
	}
	return &UseCases{
		CreateTemplateTaskCriteriaRatingDescription:          &CreateUseCase{shared: shared},
		ReadTemplateTaskCriteriaRatingDescription:            &ReadUseCase{shared: shared},
		UpdateTemplateTaskCriteriaRatingDescription:          &UpdateUseCase{shared: shared},
		DeleteTemplateTaskCriteriaRatingDescription:          &DeleteUseCase{shared: shared},
		ListTemplateTaskCriteriaRatingDescriptions:           &ListUseCase{shared: shared},
		GetTemplateTaskCriteriaRatingDescriptionListPageData: &ListPageDataUseCase{shared: shared},
		GetTemplateTaskCriteriaRatingDescriptionItemPageData: &ItemPageDataUseCase{shared: shared},
		ListByTemplateTaskCriteria:                           &ListByTemplateTaskCriteriaUseCase{shared: shared},
	}
}

type shared struct {
	repo                 pb.TemplateTaskCriteriaRatingDescriptionDomainServiceServer
	templateTaskCriteria templatetaskcriteriapb.TemplateTaskCriteriaDomainServiceServer
	outcomeCriteria      outcomecriteriapb.OutcomeCriteriaDomainServiceServer
	scoreScaleBand       scorescalebandpb.ScoreScaleBandDomainServiceServer
	services             Services
}

func (u shared) check(ctx context.Context, action string) error {
	if u.services.ActionGatekeeper == nil {
		return nil
	}
	// This entity is an embedded configuration child, not a standalone
	// permission surface. Reuse the parent binding's read/update grants so the
	// Standards drawer can persist descriptions without requiring operators to
	// receive an otherwise invisible child permission. The child still enforces
	// its own workspace, parent-mode, criterion, and band ownership checks below.
	parentAction := entityid.ActionRead
	switch action {
	case entityid.ActionCreate, entityid.ActionUpdate, entityid.ActionDelete:
		parentAction = entityid.ActionUpdate
	}
	return u.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.TemplateTaskCriteria, Action: parentAction})
}

func (u shared) validateConfiguration(ctx context.Context, data *pb.TemplateTaskCriteriaRatingDescription) error {
	if data == nil || data.GetTemplateTaskCriteriaId() == "" || data.GetScoreScaleBandId() == "" {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.parent_required", "Template task criteria and scale band are required")
	}
	if u.templateTaskCriteria == nil || u.outcomeCriteria == nil || u.scoreScaleBand == nil {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.dependencies_unavailable", "Rating configuration dependencies are unavailable")
	}

	parentResp, err := u.templateTaskCriteria.ReadTemplateTaskCriteria(ctx, &templatetaskcriteriapb.ReadTemplateTaskCriteriaRequest{
		Data: &templatetaskcriteriapb.TemplateTaskCriteria{Id: data.GetTemplateTaskCriteriaId()},
	})
	if err != nil || parentResp == nil || len(parentResp.GetData()) == 0 {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.parent_not_found", "Template task criteria was not found")
	}
	parent := parentResp.GetData()[0]
	if !parent.GetActive() || parent.GetRatingMode() != enums.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION || parent.GetRatingScaleId() == "" {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.parent_mode", "The binding is not configured for numeric descriptions")
	}

	criteriaResp, err := u.outcomeCriteria.ReadOutcomeCriteria(ctx, &outcomecriteriapb.ReadOutcomeCriteriaRequest{
		Data: &outcomecriteriapb.OutcomeCriteria{Id: parent.GetOutcomeCriteriaId()},
	})
	if err != nil || criteriaResp == nil || len(criteriaResp.GetData()) == 0 {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.criteria_not_found", "Outcome criteria was not found")
	}
	criteria := criteriaResp.GetData()[0]
	if criteria.GetCriteriaType() != enums.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE && criteria.GetCriteriaType() != enums.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.numeric_required", "Rating descriptions require a numeric criterion")
	}

	bandResp, err := u.scoreScaleBand.ReadScoreScaleBand(ctx, &scorescalebandpb.ReadScoreScaleBandRequest{
		Data: &scorescalebandpb.ScoreScaleBand{Id: data.GetScoreScaleBandId()},
	})
	if err != nil || bandResp == nil || len(bandResp.GetData()) == 0 {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.band_not_found", "Scale band was not found")
	}
	band := bandResp.GetData()[0]
	if !band.GetActive() || band.GetScoreScaleId() != parent.GetRatingScaleId() {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.band_scale_mismatch", "Scale band does not belong to the binding scale")
	}

	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if ws == "" {
		return validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.workspace_required", "Workspace is required")
	}
	data.WorkspaceId = &ws
	return nil
}

func validation(ctx context.Context, translator ports.Translator, key, fallback string) error {
	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator, key, fallback+" [DEFAULT]"))
}

type CreateUseCase struct{ shared }

func (u *CreateUseCase) Execute(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRatingDescriptionRequest) (*pb.CreateTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if err := u.check(ctx, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.data_required", "Data is required")
	}
	if strings.TrimSpace(req.Data.Description) == "" {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.description_required", "Description is required")
	}
	if err := u.validateConfiguration(ctx, req.Data); err != nil {
		return nil, err
	}
	if req.Data.Id == "" && u.services.IDGenerator != nil {
		req.Data.Id = u.services.IDGenerator.GenerateID()
	}
	now := time.Now()
	ms := now.UnixMilli()
	stamp := now.Format(time.RFC3339)
	req.Data.Active = true
	req.Data.DateCreated = &ms
	req.Data.DateCreatedString = &stamp
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &stamp
	return u.repo.CreateTemplateTaskCriteriaRatingDescription(ctx, req)
}

type ReadUseCase struct{ shared }

func (u *ReadUseCase) Execute(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRatingDescriptionRequest) (*pb.ReadTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if err := u.check(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.ReadTemplateTaskCriteriaRatingDescription(ctx, req)
}

type UpdateUseCase struct{ shared }

func (u *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRatingDescriptionRequest) (*pb.UpdateTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if err := u.check(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.data_required", "Data is required")
	}
	if req.Data.Id == "" {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.id_required", "ID is required")
	}
	if strings.TrimSpace(req.Data.Description) == "" {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.description_required", "Description is required")
	}
	if err := u.validateConfiguration(ctx, req.Data); err != nil {
		return nil, err
	}
	now := time.Now()
	ms := now.UnixMilli()
	stamp := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &stamp
	return u.repo.UpdateTemplateTaskCriteriaRatingDescription(ctx, req)
}

type DeleteUseCase struct{ shared }

func (u *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRatingDescriptionRequest) (*pb.DeleteTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if err := u.check(ctx, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.DeleteTemplateTaskCriteriaRatingDescription(ctx, req)
}

type ListUseCase struct{ shared }

func (u *ListUseCase) Execute(ctx context.Context, req *pb.ListTemplateTaskCriteriaRatingDescriptionsRequest) (*pb.ListTemplateTaskCriteriaRatingDescriptionsResponse, error) {
	if err := u.check(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.ListTemplateTaskCriteriaRatingDescriptions(ctx, req)
}

type ListPageDataUseCase struct{ shared }

func (u *ListPageDataUseCase) Execute(ctx context.Context, req *pb.GetTemplateTaskCriteriaRatingDescriptionListPageDataRequest) (*pb.GetTemplateTaskCriteriaRatingDescriptionListPageDataResponse, error) {
	if err := u.check(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.GetTemplateTaskCriteriaRatingDescriptionListPageData(ctx, req)
}

type ItemPageDataUseCase struct{ shared }

func (u *ItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetTemplateTaskCriteriaRatingDescriptionItemPageDataRequest) (*pb.GetTemplateTaskCriteriaRatingDescriptionItemPageDataResponse, error) {
	if err := u.check(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.GetTemplateTaskCriteriaRatingDescriptionItemPageData(ctx, req)
}

type ListByTemplateTaskCriteriaUseCase struct{ shared }

func (u *ListByTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.ListTemplateTaskCriteriaRatingDescriptionsByTemplateTaskCriteriaRequest) (*pb.ListTemplateTaskCriteriaRatingDescriptionsByTemplateTaskCriteriaResponse, error) {
	if err := u.check(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validation(ctx, u.services.Translator, "template_task_criteria_rating_description.validation.request_required", "Request is required")
	}
	return u.repo.ListByTemplateTaskCriteria(ctx, req)
}
