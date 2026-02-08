package consumer

import (
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	infraopts "github.com/erniealice/espyna-golang/internal/composition/options/infrastructure"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
	"github.com/erniealice/espyna-golang/internal/composition/routing/customization"
)

// Container is the exported container type for consumer apps
type Container = core.Container

// UseCases is the exported use cases aggregate type for consumer apps
type UseCases = usecases.Aggregate

// RouteExporter provides consumer apps access to routes with customization support
type RouteExporter struct {
	container  *core.Container
	customizer *customization.RouteCustomizer
}

// NewContainer creates a new espyna container with the specified options
func NewContainer(opts ...infraopts.ContainerOption) (*core.Container, error) {
	return core.NewContainerWithOptions(opts...)
}

// NewContainerFromEnv creates a new espyna container from environment variables
// This reads CONFIG_* environment variables and configures providers accordingly.
//
// Environment variables:
//   - CONFIG_DATABASE_PROVIDER: mock_db, postgres, firestore (default: mock_db)
//   - CONFIG_AUTH_PROVIDER: mock_auth, firebase_auth (default: mock_auth)
//   - CONFIG_ID_PROVIDER: noop, google_uuidv7 (default: noop)
//   - CONFIG_STORAGE_PROVIDER: mock_storage, local (default: mock_storage)
func NewContainerFromEnv() *core.Container {
	return core.NewContainerFromEnv()
}

// NewRouteExporter creates a route exporter from a container
func NewRouteExporter(container *core.Container) *RouteExporter {
	return &RouteExporter{
		container:  container,
		customizer: customization.NewRouteCustomizer(),
	}
}

// GetCustomizer returns the route customizer for configuration
func (re *RouteExporter) GetCustomizer() *customization.RouteCustomizer {
	return re.customizer
}

// GetCustomizedRoutes returns all routes with customizations applied
func (re *RouteExporter) GetCustomizedRoutes() []*routing.Route {
	baseRoutes := re.container.GetRouteManager().GetAllRoutes()
	return re.customizer.ApplyCustomizations(baseRoutes)
}

// GetRoutesByDomain returns customized routes for a specific domain
func (re *RouteExporter) GetRoutesByDomain(domain string) []*routing.Route {
	allRoutes := re.GetCustomizedRoutes()
	filtered := []*routing.Route{}
	for _, route := range allRoutes {
		if route.Metadata.Domain == domain {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// ListRouteNames returns all available route names for reference
func (re *RouteExporter) ListRouteNames() []string {
	routes := re.container.GetRouteManager().GetAllRoutes()
	names := []string{}
	for _, route := range routes {
		if route.Metadata.Name != "" {
			names = append(names, route.Metadata.Name)
		}
	}
	return names
}

// GetRouteByName returns a customized route by its name
func (re *RouteExporter) GetRouteByName(name string) *routing.Route {
	routes := re.GetCustomizedRoutes()
	for _, route := range routes {
		if route.Metadata.Name == name {
			return route
		}
	}
	return nil
}

// Re-export options for convenience
var (
	// Database Options
	WithDatabaseFromEnv   = infraopts.WithDatabaseFromEnv
	WithMockDatabase      = infraopts.WithMockDatabase
	WithPostgresDatabase  = infraopts.WithPostgresDatabase
	WithFirestoreDatabase = infraopts.WithFirestoreDatabase
	// Auth Options
	WithMockAuth = infraopts.WithMockAuth
	// Storage Options
	WithMockStorage = infraopts.WithMockStorage
)

// Type aliases for convenience
type (
	ProtobufParser          = contracts.ProtobufParser
	Route                   = routing.Route
	LeapforCustomRoute      = routing.LeapforCustomRoute
	MockDatabaseConfig      = infraopts.MockDatabaseConfig
	FirestoreDatabaseConfig = infraopts.FirestoreDatabaseConfig
	PostgresDatabaseConfig  = infraopts.PostgresDatabaseConfig
)
