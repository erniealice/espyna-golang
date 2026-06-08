// Package evaluation_cycle_member holds the CRUD use cases for the
// evaluation_cycle_member child aggregate (SR-1 frozen-denominator snapshot).
// The idempotent INSERT-on-open is driven by the domain-layer
// evaluation_cycle.OpenUseCase; this package provides the plain CRUD surface.
package evaluation_cycle_member

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle_member"
)

// Repositories groups all repository dependencies.
type Repositories struct {
	EvaluationCycleMember pb.EvaluationCycleMemberDomainServiceServer
}

// Services groups all business service dependencies.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the evaluation_cycle_member use cases.
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

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateEvaluationCycleMemberRequest) (*pb.CreateEvaluationCycleMemberResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle_member.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.s.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	if req.Data.DateAdded == nil {
		req.Data.DateAdded = &[]int64{now.UnixMilli()}[0]
	}
	return uc.r.EvaluationCycleMember.CreateEvaluationCycleMember(ctx, req)
}

type ReadUseCase struct {
	r Repositories
	s Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadEvaluationCycleMemberRequest) (*pb.ReadEvaluationCycleMemberResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle_member.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationCycleMember.ReadEvaluationCycleMember(ctx, req)
}

type UpdateUseCase struct {
	r Repositories
	s Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateEvaluationCycleMemberRequest) (*pb.UpdateEvaluationCycleMemberResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle_member.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationCycleMember.UpdateEvaluationCycleMember(ctx, req)
}

type DeleteUseCase struct {
	r Repositories
	s Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteEvaluationCycleMemberRequest) (*pb.DeleteEvaluationCycleMemberResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle_member.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationCycleMember.DeleteEvaluationCycleMember(ctx, req)
}

type ListUseCase struct {
	r Repositories
	s Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListEvaluationCycleMembersRequest) (*pb.ListEvaluationCycleMembersResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycleMember.ListEvaluationCycleMembers(ctx, req)
}

type GetListPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetListPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationCycleMemberListPageDataRequest) (*pb.GetEvaluationCycleMemberListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycleMember.GetEvaluationCycleMemberListPageData(ctx, req)
}

type GetItemPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationCycleMemberItemPageDataRequest) (*pb.GetEvaluationCycleMemberItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.s.Authorizer, uc.s.Translator, entityid.EvaluationCycleMember, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycleMember.GetEvaluationCycleMemberItemPageData(ctx, req)
}
