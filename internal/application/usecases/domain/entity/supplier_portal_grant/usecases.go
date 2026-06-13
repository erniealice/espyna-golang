package supplier_portal_grant

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
)

const entitySupplierPortalGrant = "supplier_portal_grant"

// SupplierPortalGrantRepositories groups repository dependencies.
type SupplierPortalGrantRepositories struct {
	SupplierPortalGrant supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
}

// SupplierPortalGrantServices groups service dependencies.
type SupplierPortalGrantServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all supplier_portal_grant use cases.
type UseCases struct {
	Create *CreateSupplierPortalGrantUseCase
	Read   *ReadSupplierPortalGrantUseCase
	Update *UpdateSupplierPortalGrantUseCase
	Delete *DeleteSupplierPortalGrantUseCase
	List   *ListSupplierPortalGrantsUseCase
}

// NewUseCases creates a new collection of supplier_portal_grant use cases.
func NewUseCases(repos SupplierPortalGrantRepositories, services SupplierPortalGrantServices) *UseCases {
	return &UseCases{
		Create: &CreateSupplierPortalGrantUseCase{repo: repos.SupplierPortalGrant, services: services},
		Read:   &ReadSupplierPortalGrantUseCase{repo: repos.SupplierPortalGrant, services: services},
		Update: &UpdateSupplierPortalGrantUseCase{repo: repos.SupplierPortalGrant, services: services},
		Delete: &DeleteSupplierPortalGrantUseCase{repo: repos.SupplierPortalGrant, services: services},
		List:   &ListSupplierPortalGrantsUseCase{repo: repos.SupplierPortalGrant, services: services},
	}
}

// CreateSupplierPortalGrantUseCase handles creating a supplier portal grant.
type CreateSupplierPortalGrantUseCase struct {
	repo     supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	services SupplierPortalGrantServices
}

func (uc *CreateSupplierPortalGrantUseCase) Execute(ctx context.Context, req *supplierportalgrantpb.CreateSupplierPortalGrantRequest) (*supplierportalgrantpb.CreateSupplierPortalGrantResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierPortalGrant,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("supplier_portal_grant data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateSupplierPortalGrant(ctx, req)
}

// ReadSupplierPortalGrantUseCase handles reading a supplier portal grant.
type ReadSupplierPortalGrantUseCase struct {
	repo     supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	services SupplierPortalGrantServices
}

func (uc *ReadSupplierPortalGrantUseCase) Execute(ctx context.Context, req *supplierportalgrantpb.ReadSupplierPortalGrantRequest) (*supplierportalgrantpb.ReadSupplierPortalGrantResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierPortalGrant,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ReadSupplierPortalGrant(ctx, req)
}

// UpdateSupplierPortalGrantUseCase handles updating a supplier portal grant.
type UpdateSupplierPortalGrantUseCase struct {
	repo     supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	services SupplierPortalGrantServices
}

func (uc *UpdateSupplierPortalGrantUseCase) Execute(ctx context.Context, req *supplierportalgrantpb.UpdateSupplierPortalGrantRequest) (*supplierportalgrantpb.UpdateSupplierPortalGrantResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierPortalGrant,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier_portal_grant ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateSupplierPortalGrant(ctx, req)
}

// DeleteSupplierPortalGrantUseCase handles deleting a supplier portal grant.
type DeleteSupplierPortalGrantUseCase struct {
	repo     supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	services SupplierPortalGrantServices
}

func (uc *DeleteSupplierPortalGrantUseCase) Execute(ctx context.Context, req *supplierportalgrantpb.DeleteSupplierPortalGrantRequest) (*supplierportalgrantpb.DeleteSupplierPortalGrantResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierPortalGrant,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	return uc.repo.DeleteSupplierPortalGrant(ctx, req)
}

// ListSupplierPortalGrantsUseCase handles listing supplier portal grants.
type ListSupplierPortalGrantsUseCase struct {
	repo     supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	services SupplierPortalGrantServices
}

func (uc *ListSupplierPortalGrantsUseCase) Execute(ctx context.Context, req *supplierportalgrantpb.ListSupplierPortalGrantsRequest) (*supplierportalgrantpb.ListSupplierPortalGrantsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierPortalGrant,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ListSupplierPortalGrants(ctx, req)
}
