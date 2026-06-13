package tenant_payment_method

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	tenantpaymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_payment_method"
)

const entityTenantPaymentMethod = "tenant_payment_method"

// TenantPaymentMethodRepositories groups repository dependencies.
type TenantPaymentMethodRepositories struct {
	TenantPaymentMethod tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
}

// TenantPaymentMethodServices groups service dependencies.
type TenantPaymentMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all tenant_payment_method use cases.
type UseCases struct {
	Create *CreateTenantPaymentMethodUseCase
	Read   *ReadTenantPaymentMethodUseCase
	Update *UpdateTenantPaymentMethodUseCase
	Delete *DeleteTenantPaymentMethodUseCase
	List   *ListTenantPaymentMethodsUseCase
}

// NewUseCases creates a new collection of tenant_payment_method use cases.
func NewUseCases(repos TenantPaymentMethodRepositories, services TenantPaymentMethodServices) *UseCases {
	return &UseCases{
		Create: &CreateTenantPaymentMethodUseCase{repo: repos.TenantPaymentMethod, services: services},
		Read:   &ReadTenantPaymentMethodUseCase{repo: repos.TenantPaymentMethod, services: services},
		Update: &UpdateTenantPaymentMethodUseCase{repo: repos.TenantPaymentMethod, services: services},
		Delete: &DeleteTenantPaymentMethodUseCase{repo: repos.TenantPaymentMethod, services: services},
		List:   &ListTenantPaymentMethodsUseCase{repo: repos.TenantPaymentMethod, services: services},
	}
}

// CreateTenantPaymentMethodUseCase handles creating a tenant payment method.
type CreateTenantPaymentMethodUseCase struct {
	repo     tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	services TenantPaymentMethodServices
}

func (uc *CreateTenantPaymentMethodUseCase) Execute(ctx context.Context, req *tenantpaymentmethodpb.CreateTenantPaymentMethodRequest) (*tenantpaymentmethodpb.CreateTenantPaymentMethodResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTenantPaymentMethod,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("tenant_payment_method data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateTenantPaymentMethod(ctx, req)
}

// ReadTenantPaymentMethodUseCase handles reading a tenant payment method.
type ReadTenantPaymentMethodUseCase struct {
	repo     tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	services TenantPaymentMethodServices
}

func (uc *ReadTenantPaymentMethodUseCase) Execute(ctx context.Context, req *tenantpaymentmethodpb.ReadTenantPaymentMethodRequest) (*tenantpaymentmethodpb.ReadTenantPaymentMethodResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTenantPaymentMethod,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ReadTenantPaymentMethod(ctx, req)
}

// UpdateTenantPaymentMethodUseCase handles updating a tenant payment method.
type UpdateTenantPaymentMethodUseCase struct {
	repo     tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	services TenantPaymentMethodServices
}

func (uc *UpdateTenantPaymentMethodUseCase) Execute(ctx context.Context, req *tenantpaymentmethodpb.UpdateTenantPaymentMethodRequest) (*tenantpaymentmethodpb.UpdateTenantPaymentMethodResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTenantPaymentMethod,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_payment_method ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateTenantPaymentMethod(ctx, req)
}

// DeleteTenantPaymentMethodUseCase handles deleting a tenant payment method.
type DeleteTenantPaymentMethodUseCase struct {
	repo     tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	services TenantPaymentMethodServices
}

func (uc *DeleteTenantPaymentMethodUseCase) Execute(ctx context.Context, req *tenantpaymentmethodpb.DeleteTenantPaymentMethodRequest) (*tenantpaymentmethodpb.DeleteTenantPaymentMethodResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTenantPaymentMethod,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}
	return uc.repo.DeleteTenantPaymentMethod(ctx, req)
}

// ListTenantPaymentMethodsUseCase handles listing tenant payment methods.
type ListTenantPaymentMethodsUseCase struct {
	repo     tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	services TenantPaymentMethodServices
}

func (uc *ListTenantPaymentMethodsUseCase) Execute(ctx context.Context, req *tenantpaymentmethodpb.ListTenantPaymentMethodsRequest) (*tenantpaymentmethodpb.ListTenantPaymentMethodsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTenantPaymentMethod,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repo.ListTenantPaymentMethods(ctx, req)
}
