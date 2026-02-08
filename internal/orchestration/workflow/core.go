package workflow

import (
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/domain"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/integration"
)

// Registry implements ports.ExecutorRegistry for the workflow engine.
// It provides dynamic lookup of use case executors by their string codes.
//
// Use case codes follow the format: {domain}.{resource}.{operation}
// Examples:
//   - "entity.client.create"
//   - "subscription.plan.list"
//   - "integration.email.send"
type Registry struct {
	useCases  *usecases.Aggregate
	executors map[string]ports.ActivityExecutor
}

// NewRegistry creates a new workflow use case registry.
// The registry registers all executors during initialization.
func NewRegistry(useCases *usecases.Aggregate) ports.ExecutorRegistry {
	r := &Registry{
		useCases:  useCases,
		executors: make(map[string]ports.ActivityExecutor),
	}
	r.registerAll()
	return r
}

// GetExecutor returns an executor for the given use case code.
// Returns an error if the use case is not found in the registry.
func (r *Registry) GetExecutor(useCaseCode string) (ports.ActivityExecutor, error) {
	log.Printf("[WorkflowRegistry] GetExecutor called: %s (registry=%p, total=%d)", useCaseCode, r, len(r.executors))
	if executor, ok := r.executors[useCaseCode]; ok {
		return executor, nil
	}
	// Log available executors for debugging
	available := make([]string, 0, len(r.executors))
	for k := range r.executors {
		available = append(available, k)
	}
	log.Printf("[WorkflowRegistry] NOT FOUND: %s. Available: %v", useCaseCode, available)
	return nil, fmt.Errorf("use case not found in registry: %s", useCaseCode)
}

// registerAll registers all use case executors.
// This method is called once during registry initialization.
func (r *Registry) registerAll() {
	// Register domain use cases
	domain.RegisterEntityUseCases(r.useCases, r.register)
	domain.RegisterSubscriptionUseCases(r.useCases, r.register)
	domain.RegisterPaymentUseCases(r.useCases, r.register)
	domain.RegisterProductUseCases(r.useCases, r.register)
	domain.RegisterEventUseCases(r.useCases, r.register)
	domain.RegisterWorkflowUseCases(r.useCases, r.register)
	domain.RegisterCommonUseCases(r.useCases, r.register)

	// Register integration use cases
	integration.RegisterEmailIntegrationUseCases(r.useCases, r.register)
	integration.RegisterPaymentIntegrationUseCases(r.useCases, r.register)
	integration.RegisterTabularIntegrationUseCases(r.useCases, r.register)
}

// register adds an executor to the registry.
// This is a helper method used by domain-specific registration methods.
func (r *Registry) register(code string, executor ports.ActivityExecutor) {
	// log.Printf("[WorkflowRegistry] Storing executor: %s (registry=%p)", code, r)
	r.executors[code] = executor
}
