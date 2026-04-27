package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"

	// Protobuf domain services - Expenditure domain
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// ExpenditureRepositories contains all expenditure domain repositories
type ExpenditureRepositories struct {
	Expenditure            expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem    expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	ExpenditureCategory    expenditurecategorypb.ExpenditureCategoryDomainServiceServer
	ExpenditureAttribute   expenditureattributepb.ExpenditureAttributeDomainServiceServer
	Prepayment             prepaymentpb.PrepaymentDomainServiceServer
	PurchaseOrder          purchaseorderpb.PurchaseOrderDomainServiceServer
	PurchaseOrderLineItem  purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
	SupplierContract       suppliercontractpb.SupplierContractDomainServiceServer
	SupplierContractLine   suppliercontractlinepb.SupplierContractLineDomainServiceServer
	ProcurementRequest     procurementrequestpb.ProcurementRequestDomainServiceServer
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// NewExpenditureRepositories creates and returns a new set of ExpenditureRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewExpenditureRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*ExpenditureRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &ExpenditureRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	if r := tryCreate(entityid.Expenditure); r != nil {
		repos.Expenditure = r.(expenditurepb.ExpenditureDomainServiceServer)
	}
	if r := tryCreate(entityid.ExpenditureLineItem); r != nil {
		repos.ExpenditureLineItem = r.(expenditurelineitempb.ExpenditureLineItemDomainServiceServer)
	}
	if r := tryCreate(entityid.ExpenditureCategory); r != nil {
		repos.ExpenditureCategory = r.(expenditurecategorypb.ExpenditureCategoryDomainServiceServer)
	}
	if r := tryCreate(entityid.ExpenditureAttribute); r != nil {
		repos.ExpenditureAttribute = r.(expenditureattributepb.ExpenditureAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.Prepayment); r != nil {
		repos.Prepayment = r.(prepaymentpb.PrepaymentDomainServiceServer)
	}
	if r := tryCreate(entityid.PurchaseOrder); r != nil {
		repos.PurchaseOrder = r.(purchaseorderpb.PurchaseOrderDomainServiceServer)
	}
	if r := tryCreate(entityid.PurchaseOrderLineItem); r != nil {
		repos.PurchaseOrderLineItem = r.(purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer)
	}
	if r := tryCreate(entityid.SupplierContract); r != nil {
		repos.SupplierContract = r.(suppliercontractpb.SupplierContractDomainServiceServer)
	}
	if r := tryCreate(entityid.SupplierContractLine); r != nil {
		repos.SupplierContractLine = r.(suppliercontractlinepb.SupplierContractLineDomainServiceServer)
	}
	if r := tryCreate(entityid.ProcurementRequest); r != nil {
		repos.ProcurementRequest = r.(procurementrequestpb.ProcurementRequestDomainServiceServer)
	}
	if r := tryCreate(entityid.ProcurementRequestLine); r != nil {
		repos.ProcurementRequestLine = r.(procurementrequestlinepb.ProcurementRequestLineDomainServiceServer)
	}
	if r := tryCreate(entityid.PaymentTerm); r != nil {
		repos.PaymentTerm = r.(paymenttermpb.PaymentTermDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Expenditure repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
