package tenant_invoice

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	tenantinvoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_invoice"
)

const entityTenantInvoice = "tenant_invoice"

// TenantInvoiceRepositories groups repository dependencies.
type TenantInvoiceRepositories struct {
	TenantInvoice tenantinvoicepb.TenantInvoiceDomainServiceServer
}

// TenantInvoiceServices groups service dependencies.
type TenantInvoiceServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all tenant_invoice use cases.
type UseCases struct {
	Create *CreateTenantInvoiceUseCase
	Read   *ReadTenantInvoiceUseCase
	Update *UpdateTenantInvoiceUseCase
	Delete *DeleteTenantInvoiceUseCase
	List   *ListTenantInvoicesUseCase
}

// NewUseCases creates a new collection of tenant_invoice use cases.
func NewUseCases(repos TenantInvoiceRepositories, services TenantInvoiceServices) *UseCases {
	return &UseCases{
		Create: &CreateTenantInvoiceUseCase{repo: repos.TenantInvoice, services: services},
		Read:   &ReadTenantInvoiceUseCase{repo: repos.TenantInvoice, services: services},
		Update: &UpdateTenantInvoiceUseCase{repo: repos.TenantInvoice, services: services},
		Delete: &DeleteTenantInvoiceUseCase{repo: repos.TenantInvoice, services: services},
		List:   &ListTenantInvoicesUseCase{repo: repos.TenantInvoice, services: services},
	}
}

// CreateTenantInvoiceUseCase handles creating a tenant invoice.
type CreateTenantInvoiceUseCase struct {
	repo     tenantinvoicepb.TenantInvoiceDomainServiceServer
	services TenantInvoiceServices
}

func (uc *CreateTenantInvoiceUseCase) Execute(ctx context.Context, req *tenantinvoicepb.CreateTenantInvoiceRequest) (*tenantinvoicepb.CreateTenantInvoiceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTenantInvoice, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("tenant_invoice data is required")
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true
	return uc.repo.CreateTenantInvoice(ctx, req)
}

// ReadTenantInvoiceUseCase handles reading a tenant invoice.
type ReadTenantInvoiceUseCase struct {
	repo     tenantinvoicepb.TenantInvoiceDomainServiceServer
	services TenantInvoiceServices
}

func (uc *ReadTenantInvoiceUseCase) Execute(ctx context.Context, req *tenantinvoicepb.ReadTenantInvoiceRequest) (*tenantinvoicepb.ReadTenantInvoiceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTenantInvoice, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ReadTenantInvoice(ctx, req)
}

// UpdateTenantInvoiceUseCase handles updating a tenant invoice.
type UpdateTenantInvoiceUseCase struct {
	repo     tenantinvoicepb.TenantInvoiceDomainServiceServer
	services TenantInvoiceServices
}

func (uc *UpdateTenantInvoiceUseCase) Execute(ctx context.Context, req *tenantinvoicepb.UpdateTenantInvoiceRequest) (*tenantinvoicepb.UpdateTenantInvoiceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTenantInvoice, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("tenant_invoice ID is required")
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	return uc.repo.UpdateTenantInvoice(ctx, req)
}

// DeleteTenantInvoiceUseCase handles deleting a tenant invoice.
type DeleteTenantInvoiceUseCase struct {
	repo     tenantinvoicepb.TenantInvoiceDomainServiceServer
	services TenantInvoiceServices
}

func (uc *DeleteTenantInvoiceUseCase) Execute(ctx context.Context, req *tenantinvoicepb.DeleteTenantInvoiceRequest) (*tenantinvoicepb.DeleteTenantInvoiceResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTenantInvoice, entityid.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repo.DeleteTenantInvoice(ctx, req)
}

// ListTenantInvoicesUseCase handles listing tenant invoices.
type ListTenantInvoicesUseCase struct {
	repo     tenantinvoicepb.TenantInvoiceDomainServiceServer
	services TenantInvoiceServices
}

func (uc *ListTenantInvoicesUseCase) Execute(ctx context.Context, req *tenantinvoicepb.ListTenantInvoicesRequest) (*tenantinvoicepb.ListTenantInvoicesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTenantInvoice, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.repo.ListTenantInvoices(ctx, req)
}
