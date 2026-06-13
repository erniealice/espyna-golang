// Package evaluation_cycle holds the CRUD and orchestration use cases for the
// evaluation_cycle aggregate. The status-transition orchestration
// (Open / Close) spans evaluation_cycle + evaluation_cycle_member +
// subscription_seat — these are FK-neighbor aggregates within the evaluation
// family, so the orchestration lives here in the domain layer (same pattern
// as create_evaluation.go importing subscription_seat for IDOR checks).
package evaluation_cycle

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle"
	evaluationcyclememberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle_member"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// Repositories groups all repository dependencies. EvaluationCycleMember and
// SubscriptionSeat are FK-neighbor reads required by the Open use case
// (freezeDenominator). They follow the same pattern as create_evaluation.go
// importing subscription_seat for anchor-ownership IDOR checks.
type Repositories struct {
	EvaluationCycle       pb.EvaluationCycleDomainServiceServer
	EvaluationCycleMember evaluationcyclememberpb.EvaluationCycleMemberDomainServiceServer
	SubscriptionSeat      subscriptionseatpb.SubscriptionSeatDomainServiceServer
}

// Services groups all business service dependencies.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the evaluation_cycle CRUD and orchestration use cases.
type UseCases struct {
	Create      *CreateUseCase
	Read        *ReadUseCase
	Update      *UpdateUseCase
	Delete      *DeleteUseCase
	List        *ListUseCase
	GetListPage *GetListPageDataUseCase
	GetItemPage *GetItemPageDataUseCase
	Open        *OpenUseCase
	Close       *CloseUseCase
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
		Open:        &OpenUseCase{r: r, s: s},
		Close:       &CloseUseCase{r: r, s: s},
	}
}

type CreateUseCase struct {
	r Repositories
	s Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateEvaluationCycleRequest) (*pb.CreateEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.SubscriptionId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.subscription_id_required", "Subscription ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.name_required", "Name is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.s.IDGenerator.GenerateID()
	}
	// New cycles start OPEN (never NULL — single lifecycle source).
	if req.Data.Status == pb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_UNSPECIFIED {
		req.Data.Status = pb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_OPEN
	}
	req.Data.Active = true
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationCycle.CreateEvaluationCycle(ctx, req)
}

type ReadUseCase struct {
	r Repositories
	s Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadEvaluationCycleRequest) (*pb.ReadEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationCycle.ReadEvaluationCycle(ctx, req)
}

type UpdateUseCase struct {
	r Repositories
	s Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateEvaluationCycleRequest) (*pb.UpdateEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.id_required", "ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.r.EvaluationCycle.UpdateEvaluationCycle(ctx, req)
}

type DeleteUseCase struct {
	r Repositories
	s Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteEvaluationCycleRequest) (*pb.DeleteEvaluationCycleResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.s.Translator, "evaluation_cycle.validation.id_required", "ID is required [DEFAULT]"))
	}
	return uc.r.EvaluationCycle.DeleteEvaluationCycle(ctx, req)
}

type ListUseCase struct {
	r Repositories
	s Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListEvaluationCyclesRequest) (*pb.ListEvaluationCyclesResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycle.ListEvaluationCycles(ctx, req)
}

type GetListPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetListPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationCycleListPageDataRequest) (*pb.GetEvaluationCycleListPageDataResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycle.GetEvaluationCycleListPageData(ctx, req)
}

type GetItemPageDataUseCase struct {
	r Repositories
	s Services
}

func (uc *GetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetEvaluationCycleItemPageDataRequest) (*pb.GetEvaluationCycleItemPageDataResponse, error) {
	if err := uc.s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.EvaluationCycle, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	return uc.r.EvaluationCycle.GetEvaluationCycleItemPageData(ctx, req)
}
