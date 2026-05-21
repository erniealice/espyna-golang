package delegate_supplier

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
)

const entityDelegateSupplier = "delegate_supplier"

// DelegateSupplierRepositories groups repository dependencies.
type DelegateSupplierRepositories struct {
	DelegateSupplier delegatesupplierpb.DelegateSupplierDomainServiceServer
}

// DelegateSupplierServices groups service dependencies.
type DelegateSupplierServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all delegate_supplier use cases.
type UseCases struct {
	Create *CreateDelegateSupplierUseCase
	Read   *ReadDelegateSupplierUseCase
	Update *UpdateDelegateSupplierUseCase
	Delete *DeleteDelegateSupplierUseCase
	List   *ListDelegateSuppliersUseCase
}

// NewUseCases creates a new collection of delegate_supplier use cases.
func NewUseCases(repos DelegateSupplierRepositories, services DelegateSupplierServices) *UseCases {
	return &UseCases{
		Create: &CreateDelegateSupplierUseCase{repo: repos.DelegateSupplier, services: services},
		Read:   &ReadDelegateSupplierUseCase{repo: repos.DelegateSupplier, services: services},
		Update: &UpdateDelegateSupplierUseCase{repo: repos.DelegateSupplier, services: services},
		Delete: &DeleteDelegateSupplierUseCase{repo: repos.DelegateSupplier, services: services},
		List:   &ListDelegateSuppliersUseCase{repo: repos.DelegateSupplier, services: services},
	}
}

// CreateDelegateSupplierUseCase handles creating a delegate supplier.
type CreateDelegateSupplierUseCase struct {
	repo     delegatesupplierpb.DelegateSupplierDomainServiceServer
	services DelegateSupplierServices
}

func (uc *CreateDelegateSupplierUseCase) Execute(ctx context.Context, req *delegatesupplierpb.CreateDelegateSupplierRequest) (*delegatesupplierpb.CreateDelegateSupplierResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDelegateSupplier, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("delegate_supplier data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateDelegateSupplier(ctx, req)
}

// ReadDelegateSupplierUseCase handles reading a delegate supplier.
type ReadDelegateSupplierUseCase struct {
	repo     delegatesupplierpb.DelegateSupplierDomainServiceServer
	services DelegateSupplierServices
}

func (uc *ReadDelegateSupplierUseCase) Execute(ctx context.Context, req *delegatesupplierpb.ReadDelegateSupplierRequest) (*delegatesupplierpb.ReadDelegateSupplierResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDelegateSupplier, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadDelegateSupplier(ctx, req)
}

// UpdateDelegateSupplierUseCase handles updating a delegate supplier.
type UpdateDelegateSupplierUseCase struct {
	repo     delegatesupplierpb.DelegateSupplierDomainServiceServer
	services DelegateSupplierServices
}

func (uc *UpdateDelegateSupplierUseCase) Execute(ctx context.Context, req *delegatesupplierpb.UpdateDelegateSupplierRequest) (*delegatesupplierpb.UpdateDelegateSupplierResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDelegateSupplier, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("delegate_supplier ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateDelegateSupplier(ctx, req)
}

// DeleteDelegateSupplierUseCase handles deleting a delegate supplier.
type DeleteDelegateSupplierUseCase struct {
	repo     delegatesupplierpb.DelegateSupplierDomainServiceServer
	services DelegateSupplierServices
}

func (uc *DeleteDelegateSupplierUseCase) Execute(ctx context.Context, req *delegatesupplierpb.DeleteDelegateSupplierRequest) (*delegatesupplierpb.DeleteDelegateSupplierResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDelegateSupplier, ports.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteDelegateSupplier(ctx, req)
}

// ListDelegateSuppliersUseCase handles listing delegate suppliers.
type ListDelegateSuppliersUseCase struct {
	repo     delegatesupplierpb.DelegateSupplierDomainServiceServer
	services DelegateSupplierServices
}

func (uc *ListDelegateSuppliersUseCase) Execute(ctx context.Context, req *delegatesupplierpb.ListDelegateSuppliersRequest) (*delegatesupplierpb.ListDelegateSuppliersResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDelegateSupplier, ports.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListDelegateSuppliers(ctx, req)
}
