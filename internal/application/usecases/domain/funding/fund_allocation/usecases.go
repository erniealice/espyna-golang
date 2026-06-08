// Package fund_allocation contains use cases for the FundAllocation entity in the funding domain.
// FundAllocation is workspace-scoped and acts as the junction between a global Fund
// and a specific workspace. It carries workspace_id and defines the allocation limit.
package fund_allocation

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	fundallocationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_allocation"
)

const entityFundAllocation = "fund_allocation"

// FundAllocationRepositories groups repository dependencies.
type FundAllocationRepositories struct {
	FundAllocation fundallocationpb.FundAllocationDomainServiceServer
}

// FundAllocationServices groups service dependencies.
type FundAllocationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all fund_allocation use cases.
type UseCases struct {
	Create *CreateFundAllocationUseCase
	Read   *ReadFundAllocationUseCase
	Update *UpdateFundAllocationUseCase
	Delete *DeleteFundAllocationUseCase
	List   *ListFundAllocationsUseCase
}

// NewUseCases creates a new collection of fund_allocation use cases.
func NewUseCases(repos FundAllocationRepositories, services FundAllocationServices) *UseCases {
	return &UseCases{
		Create: &CreateFundAllocationUseCase{repo: repos.FundAllocation, services: services},
		Read:   &ReadFundAllocationUseCase{repo: repos.FundAllocation, services: services},
		Update: &UpdateFundAllocationUseCase{repo: repos.FundAllocation, services: services},
		Delete: &DeleteFundAllocationUseCase{repo: repos.FundAllocation, services: services},
		List:   &ListFundAllocationsUseCase{repo: repos.FundAllocation, services: services},
	}
}

// CreateFundAllocationUseCase handles creating a fund_allocation.
type CreateFundAllocationUseCase struct {
	repo     fundallocationpb.FundAllocationDomainServiceServer
	services FundAllocationServices
}

func (uc *CreateFundAllocationUseCase) Execute(ctx context.Context, req *fundallocationpb.CreateFundAllocationRequest) (*fundallocationpb.CreateFundAllocationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFundAllocation, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("fund_allocation data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateFundAllocation(ctx, req)
}

// ReadFundAllocationUseCase handles reading a fund_allocation.
type ReadFundAllocationUseCase struct {
	repo     fundallocationpb.FundAllocationDomainServiceServer
	services FundAllocationServices
}

func (uc *ReadFundAllocationUseCase) Execute(ctx context.Context, req *fundallocationpb.ReadFundAllocationRequest) (*fundallocationpb.ReadFundAllocationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFundAllocation, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadFundAllocation(ctx, req)
}

// UpdateFundAllocationUseCase handles updating a fund_allocation.
type UpdateFundAllocationUseCase struct {
	repo     fundallocationpb.FundAllocationDomainServiceServer
	services FundAllocationServices
}

func (uc *UpdateFundAllocationUseCase) Execute(ctx context.Context, req *fundallocationpb.UpdateFundAllocationRequest) (*fundallocationpb.UpdateFundAllocationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFundAllocation, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_allocation ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateFundAllocation(ctx, req)
}

// DeleteFundAllocationUseCase handles deleting a fund_allocation.
type DeleteFundAllocationUseCase struct {
	repo     fundallocationpb.FundAllocationDomainServiceServer
	services FundAllocationServices
}

func (uc *DeleteFundAllocationUseCase) Execute(ctx context.Context, req *fundallocationpb.DeleteFundAllocationRequest) (*fundallocationpb.DeleteFundAllocationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFundAllocation, entityid.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteFundAllocation(ctx, req)
}

// ListFundAllocationsUseCase handles listing fund_allocations.
type ListFundAllocationsUseCase struct {
	repo     fundallocationpb.FundAllocationDomainServiceServer
	services FundAllocationServices
}

func (uc *ListFundAllocationsUseCase) Execute(ctx context.Context, req *fundallocationpb.ListFundAllocationsRequest) (*fundallocationpb.ListFundAllocationsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityFundAllocation, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListFundAllocations(ctx, req)
}
