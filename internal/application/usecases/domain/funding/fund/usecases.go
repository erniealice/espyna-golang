// Package fund contains use cases for the Fund entity in the funding domain.
// Fund is a global entity (no workspace_id) — the physical/financial instrument.
// Access to a Fund from a workspace context must be mediated by an active FundAllocation.
package fund

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	fundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund"
)

const entityFund = "fund"

// FundRepositories groups repository dependencies.
type FundRepositories struct {
	Fund fundpb.FundDomainServiceServer
}

// FundServices groups service dependencies.
type FundServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all fund use cases.
type UseCases struct {
	Create *CreateFundUseCase
	Read   *ReadFundUseCase
	Update *UpdateFundUseCase
	Delete *DeleteFundUseCase
	List   *ListFundsUseCase
}

// NewUseCases creates a new collection of fund use cases.
func NewUseCases(repos FundRepositories, services FundServices) *UseCases {
	return &UseCases{
		Create: &CreateFundUseCase{repo: repos.Fund, services: services},
		Read:   &ReadFundUseCase{repo: repos.Fund, services: services},
		Update: &UpdateFundUseCase{repo: repos.Fund, services: services},
		Delete: &DeleteFundUseCase{repo: repos.Fund, services: services},
		List:   &ListFundsUseCase{repo: repos.Fund, services: services},
	}
}

// CreateFundUseCase handles creating a fund.
type CreateFundUseCase struct {
	repo     fundpb.FundDomainServiceServer
	services FundServices
}

func (uc *CreateFundUseCase) Execute(ctx context.Context, req *fundpb.CreateFundRequest) (*fundpb.CreateFundResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityFund,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("fund data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateFund(ctx, req)
}

// ReadFundUseCase handles reading a fund.
type ReadFundUseCase struct {
	repo     fundpb.FundDomainServiceServer
	services FundServices
}

func (uc *ReadFundUseCase) Execute(ctx context.Context, req *fundpb.ReadFundRequest) (*fundpb.ReadFundResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityFund,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ReadFund(ctx, req)
}

// UpdateFundUseCase handles updating a fund.
type UpdateFundUseCase struct {
	repo     fundpb.FundDomainServiceServer
	services FundServices
}

func (uc *UpdateFundUseCase) Execute(ctx context.Context, req *fundpb.UpdateFundRequest) (*fundpb.UpdateFundResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityFund,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateFund(ctx, req)
}

// DeleteFundUseCase handles deleting a fund.
type DeleteFundUseCase struct {
	repo     fundpb.FundDomainServiceServer
	services FundServices
}

func (uc *DeleteFundUseCase) Execute(ctx context.Context, req *fundpb.DeleteFundRequest) (*fundpb.DeleteFundResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityFund,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	return uc.repo.DeleteFund(ctx, req)
}

// ListFundsUseCase handles listing funds.
type ListFundsUseCase struct {
	repo     fundpb.FundDomainServiceServer
	services FundServices
}

func (uc *ListFundsUseCase) Execute(ctx context.Context, req *fundpb.ListFundsRequest) (*fundpb.ListFundsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityFund,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ListFunds(ctx, req)
}
