package usecases

import (
	// Domain use case packages
	"github.com/erniealice/espyna-golang/internal/application/usecases/asset"
	"github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	"github.com/erniealice/espyna-golang/internal/application/usecases/common"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/fulfillment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/application/usecases/ledger"
	"github.com/erniealice/espyna-golang/internal/application/usecases/operation"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payroll"
	"github.com/erniealice/espyna-golang/internal/application/usecases/procurement"
	"github.com/erniealice/espyna-golang/internal/application/usecases/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/subscription"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/application/usecases/workflow"
)

// Aggregate is a collection of all domain use cases across the application.
// This type is exported for use by composition layers (e.g., Container, Factory)
// to aggregate and organize use cases according to their composition strategy.
//
// The Aggregate represents the complete set of entities organized across 8 domains:
// - Asset:        2 entities (Asset, AssetCategory)
// - Common:       1 entity (Attribute - cross-domain dependency)
// - Entity:       16 entities (Admin, Client, Delegate, User, Workspace, etc.)
// - Event:        2 entities (Event, EventClient)
// - Expenditure:  4 entities (Expenditure, ExpenditureLineItem, ExpenditureCategory, ExpenditureAttribute)
// - Treasury:     0 entities (legacy Payment/PaymentAttribute/PaymentMethod/PaymentProfile removed -- superseded by Collection and Disbursement)
// - Product:      8 entities (Product, Collection, Resource, PriceProduct, etc.)
// - Revenue:      4 entities (Revenue, RevenueLineItem, RevenueCategory, RevenueAttribute)
// - Subscription: 6 entities (Plan, Subscription, Invoice, Balance, etc.)
// - Workflow:     3 entities (Workflow, StageTemplate, ActivityTemplate)
// - Payroll:      2 entities (PayrollRun, PayrollRemittance)
// - Fulfillment:  1 entity (Fulfillment — placeholder, use cases pending)
// - Auth:         identity-lifecycle use cases (authenticate_session,
//                 issue_session, invalidate_session). Exempt from the
//                 authcheck coverage test — see usecases/auth/usecases.go
//                 package doc for the invariant.
type Aggregate struct {
	Auth         *auth.UseCases
	Common       *common.CommonUseCases
	Entity       *entity.EntityUseCases
	Event        *event.EventUseCases
	Expenditure  *expenditure.ExpenditureUseCases
	Fulfillment  *fulfillment.UseCases
	Inventory    *inventory.InventoryUseCases
	Ledger       *ledger.LedgerUseCases
	Operation    *operation.OperationUseCases
	Payroll      *payroll.PayrollUseCases
	Procurement  *procurement.ProcurementUseCases
	Treasury     *treasury.TreasuryUseCases
	Product      *product.ProductUseCases
	Revenue      *revenue.RevenueUseCases
	Subscription *subscription.SubscriptionUseCases
	Workflow     *workflow.WorkflowUseCases
	Integration  *integration.IntegrationUseCases
	Asset        *asset.AssetUseCases // Phase 1-2: asset typed stack (adapter in Phase 4)
}

// NewAggregate creates a new use case aggregate with all domains initialized.
// This is typically called by composition layers during container initialization.
//
// Note: Each domain's use cases should be initialized with their required
// repositories and services before being passed to this constructor.
func NewAggregate(
	authUC *auth.UseCases,
	commonUC *common.CommonUseCases,
	entityUC *entity.EntityUseCases,
	eventUC *event.EventUseCases,
	expenditureUC *expenditure.ExpenditureUseCases,
	fulfillmentUC *fulfillment.UseCases,
	inventoryUC *inventory.InventoryUseCases,
	ledgerUC *ledger.LedgerUseCases,
	operationUC *operation.OperationUseCases,
	payrollUC *payroll.PayrollUseCases,
	procurementUC *procurement.ProcurementUseCases,
	treasuryUC *treasury.TreasuryUseCases,
	productUC *product.ProductUseCases,
	revenueUC *revenue.RevenueUseCases,
	subscriptionUC *subscription.SubscriptionUseCases,
	workflowUC *workflow.WorkflowUseCases,
	integrationUC *integration.IntegrationUseCases,
	assetUC *asset.AssetUseCases,
) *Aggregate {
	return &Aggregate{
		Auth:         authUC,
		Common:       commonUC,
		Entity:       entityUC,
		Event:        eventUC,
		Expenditure:  expenditureUC,
		Fulfillment:  fulfillmentUC,
		Inventory:    inventoryUC,
		Ledger:       ledgerUC,
		Operation:    operationUC,
		Payroll:      payrollUC,
		Procurement:  procurementUC,
		Treasury:     treasuryUC,
		Product:      productUC,
		Revenue:      revenueUC,
		Subscription: subscriptionUC,
		Workflow:     workflowUC,
		Integration:  integrationUC,
		Asset:        assetUC,
	}
}

// NewEmptyAggregate creates an aggregate with empty (nil) use cases.
// This is useful for testing or gradual initialization scenarios.
func NewEmptyAggregate() *Aggregate {
	return &Aggregate{
		Auth:         &auth.UseCases{},
		Common:       &common.CommonUseCases{},
		Entity:       &entity.EntityUseCases{},
		Event:        &event.EventUseCases{},
		Expenditure:  &expenditure.ExpenditureUseCases{},
		Fulfillment:  &fulfillment.UseCases{},
		Inventory:    &inventory.InventoryUseCases{},
		Ledger:       &ledger.LedgerUseCases{},
		Operation:    &operation.OperationUseCases{},
		Payroll:      &payroll.PayrollUseCases{},
		Procurement:  &procurement.ProcurementUseCases{},
		Treasury:     &treasury.TreasuryUseCases{},
		Product:      &product.ProductUseCases{},
		Revenue:      &revenue.RevenueUseCases{},
		Subscription: &subscription.SubscriptionUseCases{},
		Workflow:     &workflow.WorkflowUseCases{},
		Integration:  &integration.IntegrationUseCases{},
		Asset:        &asset.AssetUseCases{},
	}
}
