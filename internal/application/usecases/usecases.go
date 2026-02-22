package usecases

import (
	// Domain use case packages
	"github.com/erniealice/espyna-golang/internal/application/usecases/common"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/subscription"
	"github.com/erniealice/espyna-golang/internal/application/usecases/workflow"
)

// Aggregate is a collection of all domain use cases across the application.
// This type is exported for use by composition layers (e.g., Container, Factory)
// to aggregate and organize use cases according to their composition strategy.
//
// The Aggregate represents the complete set of entities organized across 7 domains:
// - Common:       1 entity (Attribute - cross-domain dependency)
// - Entity:       16 entities (Admin, Client, Delegate, User, Workspace, etc.)
// - Event:        2 entities (Event, EventClient)
// - Payment:      3 entities (Payment, PaymentMethod, PaymentProfile)
// - Product:      8 entities (Product, Collection, Resource, PriceProduct, etc.)
// - Subscription: 6 entities (Plan, Subscription, Invoice, Balance, etc.)
// - Workflow:     3 entities (Workflow, StageTemplate, ActivityTemplate)
type Aggregate struct {
	Common       *common.CommonUseCases
	Entity       *entity.EntityUseCases
	Event        *event.EventUseCases
	Inventory    *inventory.InventoryUseCases
	Payment      *payment.PaymentUseCases
	Product      *product.ProductUseCases
	Subscription *subscription.SubscriptionUseCases
	Workflow     *workflow.WorkflowUseCases
	Integration  *integration.IntegrationUseCases
}

// NewAggregate creates a new use case aggregate with all domains initialized.
// This is typically called by composition layers during container initialization.
//
// Note: Each domain's use cases should be initialized with their required
// repositories and services before being passed to this constructor.
func NewAggregate(
	commonUC *common.CommonUseCases,
	entityUC *entity.EntityUseCases,
	eventUC *event.EventUseCases,
	inventoryUC *inventory.InventoryUseCases,
	paymentUC *payment.PaymentUseCases,
	productUC *product.ProductUseCases,
	subscriptionUC *subscription.SubscriptionUseCases,
	workflowUC *workflow.WorkflowUseCases,
	integrationUC *integration.IntegrationUseCases,
) *Aggregate {
	return &Aggregate{
		Common:       commonUC,
		Entity:       entityUC,
		Event:        eventUC,
		Inventory:    inventoryUC,
		Payment:      paymentUC,
		Product:      productUC,
		Subscription: subscriptionUC,
		Workflow:     workflowUC,
		Integration:  integrationUC,
	}
}

// NewEmptyAggregate creates an aggregate with empty (nil) use cases.
// This is useful for testing or gradual initialization scenarios.
func NewEmptyAggregate() *Aggregate {
	return &Aggregate{
		Common:       &common.CommonUseCases{},
		Entity:       &entity.EntityUseCases{},
		Event:        &event.EventUseCases{},
		Inventory:    &inventory.InventoryUseCases{},
		Payment:      &payment.PaymentUseCases{},
		Product:      &product.ProductUseCases{},
		Subscription: &subscription.SubscriptionUseCases{},
		Workflow:     &workflow.WorkflowUseCases{},
		Integration:  &integration.IntegrationUseCases{},
	}
}
