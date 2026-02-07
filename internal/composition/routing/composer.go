package routing

import (
	"context"
	"fmt"
	"log"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/composition/routing/config"
)

// Note: Composer and ComposerConfig structs have been moved to types.go

// NewMigrationManager creates a new migration manager
func NewMigrationManager(routeManager *RouteManager, config *MigrationConfig) *MigrationManager {
	return &MigrationManager{
		routeManager: routeManager,
		config:       config,
		metrics:      &MigrationMetrics{},
	}
}

// NewComposer creates a new routing composer
func NewComposer(config *ComposerConfig) (*Composer, error) {
	if config == nil {
		return nil, fmt.Errorf("composer config cannot be nil")
	}

	if config.Config == nil {
		config.Config = DefaultConfig()
	}

	// Create route manager
	routeManager := NewRouteManager(config.Config)

	// Create migration manager
	migrationMgr := NewMigrationManager(routeManager, &config.Config.MigrationConfig)

	// Inject use cases from container into handlers (if container supports it)
	var useCases *usecases.Aggregate
	if container, ok := config.Container.(interface{ GetUseCases() *usecases.Aggregate }); ok {
		useCases = container.GetUseCases()
	}

	composer := &Composer{
		config:       config.Config,
		routeManager: routeManager,
		migrationMgr: migrationMgr,
		container:    config.Container,
		useCases:     useCases,
	}

	// Initialize routes
	if err := composer.initializeRoutes(); err != nil {
		return nil, fmt.Errorf("failed to initialize routes: %w", err)
	}

	return composer, nil
}

// Note: HandlerAdapter struct has been moved to types.go

// Execute implements the routing.Handler interface
func (h *HandlerAdapter) Execute(ctx context.Context, request *Request) (*Response, error) {
	return &Response{
		Data:   map[string]interface{}{"error": "handlers not yet implemented"},
		Status: 501,
	}, nil
}

// Note: RoutingRequestAdapter struct has been moved to types.go

func (r *RoutingRequestAdapter) GetPathParams() map[string]string {
	return r.request.PathParams
}

func (r *RoutingRequestAdapter) GetBody() []byte {
	return r.request.Body
}

func (r *RoutingRequestAdapter) GetContext() context.Context {
	return r.request.Context
}

// initializeRoutes initializes all routes in the routing system
func (c *Composer) initializeRoutes() error {
	// Register declarative domain routes from configs (if use cases are available)
	log.Printf("🔍 Initializing routes... use cases available: %v", c.useCases != nil)
	if c.useCases != nil {
		// Try to get engine service from container if available
		var engineService ports.WorkflowEngineService
		if container, ok := c.container.(interface {
			GetWorkflowEngineService() ports.WorkflowEngineService
		}); ok {
			engineService = container.GetWorkflowEngineService()
			log.Printf("✅ Workflow engine service available for routing")
		}

		domainConfigs := config.GetAllDomainConfigurations(c.useCases, engineService)
		log.Printf("📊 Found %d domain configurations", len(domainConfigs))
		for _, domainConfig := range domainConfigs {
			log.Printf("📋 Processing domain '%s' (enabled: %v, routes: %d)",
				domainConfig.Domain, domainConfig.Enabled, len(domainConfig.Routes))
			if domainConfig.Enabled {
				for _, routeConfig := range domainConfig.Routes {
					// Extract metadata from path
					resource := extractResourceFromPath(routeConfig.Path)
					operation := extractOperationFromPath(routeConfig.Path)

					// Auto-generate route name
					name := fmt.Sprintf("%s.%s.%s", domainConfig.Domain, resource, operation)

					route := &Route{
						Method:  routeConfig.Method,
						Path:    routeConfig.Path,
						Handler: routeConfig.Handler,
						Metadata: RouteMetadata{
							Name:      name,
							Domain:    domainConfig.Domain,
							Resource:  resource,
							Operation: operation,
						},
					}
					if err := c.routeManager.RegisterRoute(route); err != nil {
						log.Printf("⚠️  Warning: Failed to register route %s %s: %v", route.Method, route.Path, err)
					} else {
						// log.Printf("✅ Registered route: %s %s (name: %s)", route.Method, route.Path, route.Metadata.Name)
					}
				}
			}
		}
		log.Printf("Routes initialized successfully from domain configs")
	} else {
		log.Printf("⚠️  No use cases available - skipping domain route registration")
	}

	log.Printf("Total routes registered: %d", len(c.routeManager.GetAllRoutes()))
	return nil
}

// GetRouteManager returns the route manager
func (c *Composer) GetRouteManager() *RouteManager {
	return c.routeManager
}

// GetMigrationManager returns the migration manager
func (c *Composer) GetMigrationManager() *MigrationManager {
	return c.migrationMgr
}

// GetMetrics returns migration metrics
func (c *Composer) GetMetrics() *MigrationMetrics {
	// Return empty metrics for now
	return &MigrationMetrics{}
}

// extractResourceFromPath extracts resource name from path
// "/api/workflow/workflow/create" -> "workflow"
func extractResourceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// extractOperationFromPath extracts operation from path
// "/api/workflow/workflow/create" -> "create"
func extractOperationFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}
