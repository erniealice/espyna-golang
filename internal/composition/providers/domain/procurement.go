package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Procurement domain
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// ProcurementRepositories contains all procurement domain repositories
type ProcurementRepositories struct {
	CostSchedule            costschedulepb.CostScheduleDomainServiceServer
	SupplierPlan            supplierplanpb.SupplierPlanDomainServiceServer
	CostPlan                costplanpb.CostPlanDomainServiceServer
	SupplierProductPlan     supplierproductplanpb.SupplierProductPlanDomainServiceServer
	SupplierProductCostPlan supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer
	SupplierSubscription    suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
	Workspace               workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block on create
}

// NewProcurementRepositories creates and returns a new set of ProcurementRepositories
func NewProcurementRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*ProcurementRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	costScheduleRepo, err := repoCreator.CreateRepository(entityid.CostSchedule, conn, tableConfig.TableName(entityid.CostSchedule))
	if err != nil {
		return nil, fmt.Errorf("failed to create cost_schedule repository: %w", err)
	}

	supplierPlanRepo, err := repoCreator.CreateRepository(entityid.SupplierPlan, conn, tableConfig.TableName(entityid.SupplierPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_plan repository: %w", err)
	}

	costPlanRepo, err := repoCreator.CreateRepository(entityid.CostPlan, conn, tableConfig.TableName(entityid.CostPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create cost_plan repository: %w", err)
	}

	supplierProductPlanRepo, err := repoCreator.CreateRepository(entityid.SupplierProductPlan, conn, tableConfig.TableName(entityid.SupplierProductPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_product_plan repository: %w", err)
	}

	supplierProductCostPlanRepo, err := repoCreator.CreateRepository(entityid.SupplierProductCostPlan, conn, tableConfig.TableName(entityid.SupplierProductCostPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_product_cost_plan repository: %w", err)
	}

	supplierSubscriptionRepo, err := repoCreator.CreateRepository(entityid.SupplierSubscription, conn, tableConfig.TableName(entityid.SupplierSubscription))
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier_subscription repository: %w", err)
	}

	// Cross-domain: Workspace repository for currency hard-block on CreateSupplierSubscription
	workspaceRepo, err := repoCreator.CreateRepository(entityid.Workspace, conn, tableConfig.TableName(entityid.Workspace))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace repository: %w", err)
	}

	return &ProcurementRepositories{
		CostSchedule:            costScheduleRepo.(costschedulepb.CostScheduleDomainServiceServer),
		SupplierPlan:            supplierPlanRepo.(supplierplanpb.SupplierPlanDomainServiceServer),
		CostPlan:                costPlanRepo.(costplanpb.CostPlanDomainServiceServer),
		SupplierProductPlan:     supplierProductPlanRepo.(supplierproductplanpb.SupplierProductPlanDomainServiceServer),
		SupplierProductCostPlan: supplierProductCostPlanRepo.(supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer),
		SupplierSubscription:    supplierSubscriptionRepo.(suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer),
		Workspace:               workspaceRepo.(workspacepb.WorkspaceDomainServiceServer),
	}, nil
}
