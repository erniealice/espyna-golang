// Package evaluation_response holds the CRUD use cases for the evaluation_response
// child aggregate. Rows are primarily written by SubmitEvaluation (snapshotting)
// and read for the Scores tab; Create/Update support partial-save drafts.
package evaluation_response

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_response"
)

// Repositories groups all repository dependencies. Evaluation is the parent for
// the workspace_id copy + validation (children copy workspace_id from parent).
type Repositories struct {
	EvaluationResponse pb.EvaluationResponseDomainServiceServer
	Evaluation         evaluationpb.EvaluationDomainServiceServer
}

// Services groups all business service dependencies.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the evaluation_response use cases.
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

// CreateUseCase creates an evaluation_response, copying workspace_id from the
// parent evaluation (single-write boundary; never from the form).
type CreateUseCase struct {
	r Repositories
	s Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateEvaluationResponseRequest) (*pb.CreateEvaluationResponseResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_response.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.EvaluationId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_response.validation.evaluation_id_required", "Evaluation ID is required [DEFAULT]"))
	}
	if err := copyWorkspaceFromParent(ctx, uc.r, uc.s, req.Data); err != nil {
		return nil, err
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.s.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationResponse.CreateEvaluationResponse(ctx, req)
}

// copyWorkspaceFromParent reads the parent evaluation and copies + validates its
// workspace_id onto the child (children copy workspace_id from the parent).
func copyWorkspaceFromParent(ctx context.Context, r Repositories, s Services, child *pb.EvaluationResponse) error {
	if r.Evaluation == nil {
		return nil
	}
	parent, err := r.Evaluation.ReadEvaluation(ctx, &evaluationpb.ReadEvaluationRequest{Data: &evaluationpb.Evaluation{Id: child.EvaluationId}})
	if err != nil {
		return err
	}
	if parent == nil || len(parent.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, s.Translator, "evaluation_response.errors.parent_not_found", "Parent evaluation not found [DEFAULT]"))
	}
	child.WorkspaceId = parent.Data[0].WorkspaceId
	return nil
}

type ReadUseCase struct {
	r Repositories
	s Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadEvaluationResponseRequest) (*pb.ReadEvaluationResponseResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_response.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationResponse.ReadEvaluationResponse(ctx, req)
}

type UpdateUseCase struct {
	r Repositories
	s Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateEvaluationResponseRequest) (*pb.UpdateEvaluationResponseResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_response.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationResponse.UpdateEvaluationResponse(ctx, req)
}

type DeleteUseCase struct {
	r Repositories
	s Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteEvaluationResponseRequest) (*pb.DeleteEvaluationResponseResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_response.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationResponse.DeleteEvaluationResponse(ctx, req)
}

type ListUseCase struct {
	r Repositories
	s Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListEvaluationResponsesRequest) (*pb.ListEvaluationResponsesResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationResponse.ListEvaluationResponses(ctx, req)
}

type GetListPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetListPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationResponseListPageDataRequest) (*pb.GetEvaluationResponseListPageDataResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationResponse.GetEvaluationResponseListPageData(ctx, req)
}

type GetItemPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationResponseItemPageDataRequest) (*pb.GetEvaluationResponseItemPageDataResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationResponse, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationResponse.GetEvaluationResponseItemPageData(ctx, req)
}
