package domain

import (
	"fmt"
	"sync"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// Registry manages domain repository collections
type Registry struct {
	mu sync.RWMutex

	// Repository collections by domain
	entity       *EntityRepositories
	subscription *SubscriptionRepositories
	payment      *PaymentRepositories
	product      *ProductRepositories
	inventory    *InventoryRepositories
	event        *EventRepositories
	workflow     *WorkflowRepositories
	common       *CommonRepositories

	// Database provider and config needed for lazy initialization
	dbProvider    contracts.Provider
	dbTableConfig *registry.DatabaseTableConfig
}

// NewRegistry creates a new domain registry
func NewRegistry(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) *Registry {
	return &Registry{
		dbProvider:    dbProvider,
		dbTableConfig: dbTableConfig,
	}
}

// InitializeAll creates all domain repository collections
func (r *Registry) InitializeAll() error {
	if r.dbProvider == nil {
		return fmt.Errorf("database provider not set")
	}

	var err error

	// Initialize common repositories first (other domains may depend on it)
	r.common, err = NewCommonRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create common repositories: %w", err)
	}

	// Initialize entity repositories
	r.entity, err = NewEntityRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create entity repositories: %w", err)
	}

	// Initialize subscription repositories
	r.subscription, err = NewSubscriptionRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create subscription repositories: %w", err)
	}

	// Initialize payment repositories
	r.payment, err = NewPaymentRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create payment repositories: %w", err)
	}

	// Initialize product repositories
	r.product, err = NewProductRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create product repositories: %w", err)
	}

	// Initialize inventory repositories
	r.inventory, err = NewInventoryRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create inventory repositories: %w", err)
	}

	// Initialize event repositories
	r.event, err = NewEventRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create event repositories: %w", err)
	}

	// Initialize workflow repositories
	r.workflow, err = NewWorkflowRepositories(r.dbProvider, r.dbTableConfig)
	if err != nil {
		return fmt.Errorf("failed to create workflow repositories: %w", err)
	}

	return nil
}

// GetEntity returns entity repositories (lazy init if needed)
func (r *Registry) GetEntity() (*EntityRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.entity == nil {
		var err error
		r.entity, err = NewEntityRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.entity, nil
}

// GetSubscription returns subscription repositories (lazy init if needed)
func (r *Registry) GetSubscription() (*SubscriptionRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.subscription == nil {
		var err error
		r.subscription, err = NewSubscriptionRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.subscription, nil
}

// GetPayment returns payment repositories (lazy init if needed)
func (r *Registry) GetPayment() (*PaymentRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.payment == nil {
		var err error
		r.payment, err = NewPaymentRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.payment, nil
}

// GetProduct returns product repositories (lazy init if needed)
func (r *Registry) GetProduct() (*ProductRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.product == nil {
		var err error
		r.product, err = NewProductRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.product, nil
}

// GetInventory returns inventory repositories (lazy init if needed)
func (r *Registry) GetInventory() (*InventoryRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inventory == nil {
		var err error
		r.inventory, err = NewInventoryRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.inventory, nil
}

// GetEvent returns event repositories (lazy init if needed)
func (r *Registry) GetEvent() (*EventRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.event == nil {
		var err error
		r.event, err = NewEventRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.event, nil
}

// GetWorkflow returns workflow repositories (lazy init if needed)
func (r *Registry) GetWorkflow() (*WorkflowRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.workflow == nil {
		var err error
		r.workflow, err = NewWorkflowRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.workflow, nil
}

// GetCommon returns common repositories (lazy init if needed)
func (r *Registry) GetCommon() (*CommonRepositories, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.common == nil {
		var err error
		r.common, err = NewCommonRepositories(r.dbProvider, r.dbTableConfig)
		if err != nil {
			return nil, err
		}
	}
	return r.common, nil
}
