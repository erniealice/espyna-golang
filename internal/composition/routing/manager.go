package routing

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	// TODO: Uncomment when handlers package is available
	// "github.com/erniealice/espyna-golang/internal/composition/routing/handlers"
)

// Note: RouteManager struct has been moved to types.go

// NewRouteManager creates a new route manager with the given configuration
func NewRouteManager(config *Config) *RouteManager {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		log.Printf("Warning: Invalid routing config: %v", err)
	}

	rm := &RouteManager{
		config:     config,
		routes:     make(map[string]*Route),
		groups:     make(map[string]*RouteGroup),
		middleware: make(map[string]Handler),
	}

	return rm
}

// GetConfig returns the routing configuration
func (rm *RouteManager) GetConfig() *Config {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.config
}

// RegisterRoute registers a single route
func (rm *RouteManager) RegisterRoute(route *Route) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := rm.routeKey(route.Method, route.Path)
	if _, exists := rm.routes[key]; exists {
		return fmt.Errorf("route already exists: %s %s", route.Method, route.Path)
	}

	rm.routes[key] = route
	// log.Printf("Registered route: %s %s", route.Method, route.Path)
	return nil
}

// RegisterGroup registers a route group
func (rm *RouteManager) RegisterGroup(group *RouteGroup) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if group.Prefix == "" {
		return fmt.Errorf("route group prefix cannot be empty")
	}

	rm.groups[group.Prefix] = group
	// log.Printf("Registered route group: %s", group.Prefix)
	return nil
}

// RegisterMiddleware registers a named middleware
func (rm *RouteManager) RegisterMiddleware(name string, middleware Handler) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.middleware[name] = middleware
	log.Printf("Registered middleware: %s", name)
}

// GetRoute retrieves a route by method and path
func (rm *RouteManager) GetRoute(method, path string) (*Route, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Try exact match first
	key := rm.routeKey(method, path)
	if route, exists := rm.routes[key]; exists {
		return route, true
	}

	// Try to find a matching route in groups
	for _, group := range rm.groups {
		if strings.HasPrefix(path, group.Prefix) {
			for _, route := range group.Routes {
				fullPath := group.Prefix + route.Path
				if rm.pathMatches(fullPath, path) && route.Method == method {
					return route, true
				}
			}

			// Check subgroups
			for _, subGroup := range group.SubGroups {
				fullPrefix := group.Prefix + subGroup.Prefix
				if strings.HasPrefix(path, fullPrefix) {
					for _, route := range subGroup.Routes {
						fullPath := fullPrefix + route.Path
						if rm.pathMatches(fullPath, path) && route.Method == method {
							return route, true
						}
					}
				}
			}
		}
	}

	return nil, false
}

// GetAllRoutes returns all registered routes
func (rm *RouteManager) GetAllRoutes() []*Route {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var allRoutes []*Route

	// Add individual routes
	for _, route := range rm.routes {
		allRoutes = append(allRoutes, route)
	}

	// Add routes from groups
	for _, group := range rm.groups {
		allRoutes = append(allRoutes, rm.getGroupRoutes(group, group.Prefix)...)
	}

	// Sort routes by path and method for consistency
	sort.Slice(allRoutes, func(i, j int) bool {
		if allRoutes[i].Path == allRoutes[j].Path {
			return allRoutes[i].Method < allRoutes[j].Method
		}
		return allRoutes[i].Path < allRoutes[j].Path
	})

	return allRoutes
}

// getGroupRoutes recursively gets all routes from a group and its subgroups
func (rm *RouteManager) getGroupRoutes(group *RouteGroup, prefix string) []*Route {
	var routes []*Route

	for _, route := range group.Routes {
		// Create a copy of the route with the full path
		routeCopy := &Route{
			Method:     route.Method,
			Path:       prefix + route.Path,
			Handler:    route.Handler,
			Middleware: append([]Handler{}, group.Middleware...),
			Metadata:   route.Metadata,
		}
		// Add route-specific middleware after group middleware
		routeCopy.Middleware = append(routeCopy.Middleware, route.Middleware...)
		routes = append(routes, routeCopy)
	}

	for _, subGroup := range group.SubGroups {
		subPrefix := prefix + subGroup.Prefix
		routes = append(routes, rm.getGroupRoutes(subGroup, subPrefix)...)
	}

	return routes
}

// GetRoutesByDomain returns all routes for a specific domain
func (rm *RouteManager) GetRoutesByDomain(domain string) []*Route {
	var domainRoutes []*Route

	for _, route := range rm.GetAllRoutes() {
		if route.Metadata.Domain == domain {
			domainRoutes = append(domainRoutes, route)
		}
	}

	return domainRoutes
}

// GetRoutesByResource returns all routes for a specific resource
func (rm *RouteManager) GetRoutesByResource(resource string) []*Route {
	var resourceRoutes []*Route

	for _, route := range rm.GetAllRoutes() {
		if route.Metadata.Resource == resource {
			resourceRoutes = append(resourceRoutes, route)
		}
	}

	return resourceRoutes
}

// GetMiddleware retrieves a named middleware
func (rm *RouteManager) GetMiddleware(name string) (Handler, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	middleware, exists := rm.middleware[name]
	return middleware, exists
}

// CreateDomainGroup creates a route group for a specific domain
func (rm *RouteManager) CreateDomainGroup(domain string) *GroupBuilder {
	if !rm.config.IsEndpointEnabled(domain, "") {
		// Domain is disabled
		return nil
	}

	domainConfig, exists := rm.config.Domains[domain]
	if !exists {
		return nil
	}

	builder := NewGroupBuilder(domainConfig.Prefix).
		Metadata(GroupMetadata{
			Name:        domain,
			Description: fmt.Sprintf("%s domain routes", strings.Title(domain)),
			Version:     rm.config.Version,
		})

	// Add domain-level middleware if configured
	for _, middlewareName := range domainConfig.Middleware {
		if middleware, exists := rm.GetMiddleware(middlewareName); exists {
			builder.Middleware(middleware)
		}
	}

	return builder
}

// CreateResourceRoutes creates standard CRUD routes for a resource
// TODO: Uncomment when handlers package is available
// func (rm *RouteManager) CreateResourceRoutes(domain, resource string, routeHandlers handlers.ResourceHandlers) *RouteBuilder {
func (rm *RouteManager) CreateResourceRoutes(domain, resource string, routeHandlers interface{}) *RouteBuilder {
	// TODO: Implement when handlers package is available
	// This method creates CRUD routes from handler functions
	return nil

	/* TODO: Uncomment when handlers are available
	if !rm.config.IsEndpointEnabled(domain, resource) {
		return nil
	}

	groupBuilder := rm.CreateDomainGroup(domain)
	if groupBuilder == nil {
		return nil
	}

	// Create resource group
	resourceGroup := groupBuilder.SubGroup("/" + resource).
		Metadata(GroupMetadata{
			Name:        resource,
			Description: fmt.Sprintf("%s resource routes", strings.Title(resource)),
		})

	// Add standard CRUD operations
	if routeHandlers.Create != nil {
		resourceGroup.UseCase("POST", "/create", routeHandlers.Create).
			Metadata(RouteMetadata{
				Domain:      domain,
				Resource:    resource,
				Operation:   "create",
				Description: fmt.Sprintf("Create a new %s", resource),
				Tags:        []string{"crud", "create"},
				Version:     rm.config.Version,
			})
	}

	if routeHandlers.Read != nil {
		resourceGroup.UseCase("POST", "/read", routeHandlers.Read).
			Metadata(RouteMetadata{
				Domain:      domain,
				Resource:    resource,
				Operation:   "read",
				Description: fmt.Sprintf("Read a %s", resource),
				Tags:        []string{"crud", "read"},
				Version:     rm.config.Version,
			})
	}

	if routeHandlers.Update != nil {
		resourceGroup.UseCase("POST", "/update", routeHandlers.Update).
			Metadata(RouteMetadata{
				Domain:      domain,
				Resource:    resource,
				Operation:   "update",
				Description: fmt.Sprintf("Update a %s", resource),
				Tags:        []string{"crud", "update"},
				Version:     rm.config.Version,
			})
	}

	if routeHandlers.Delete != nil {
		resourceGroup.UseCase("POST", "/delete", routeHandlers.Delete).
			Metadata(RouteMetadata{
				Domain:      domain,
				Resource:    resource,
				Operation:   "delete",
				Description: fmt.Sprintf("Delete a %s", resource),
				Tags:        []string{"crud", "delete"},
				Version:     rm.config.Version,
			})
	}

	if routeHandlers.List != nil {
		resourceGroup.UseCase("POST", "/list", routeHandlers.List).
			Metadata(RouteMetadata{
				Domain:      domain,
				Resource:    resource,
				Operation:   "list",
				Description: fmt.Sprintf("List %s items", resource),
				Tags:        []string{"crud", "list"},
				Version:     rm.config.Version,
			})
	}

	// Register the group
	group := resourceGroup.Build()
	rm.RegisterGroup(group)

	return NewRouteBuilder("POST", "/"+resource)
	*/
}

// Helper methods

func (rm *RouteManager) routeKey(method, path string) string {
	return fmt.Sprintf("%s:%s", method, path)
}

func (rm *RouteManager) pathMatches(routePath, requestPath string) bool {
	// Simple path matching - can be enhanced for wildcard routes
	return routePath == requestPath
}

func (rm *RouteManager) extractDomainAndResource(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" {
		return parts[1], parts[2]
	}
	return "unknown", "unknown"
}

func (rm *RouteManager) extractOperation(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		switch lastPart {
		case "create", "read", "update", "delete", "list":
			return lastPart
		case "upload", "download":
			return lastPart
		}
	}
	return "unknown"
}

// Close performs cleanup of route manager resources
func (rm *RouteManager) Close() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Clear all routes and groups
	rm.routes = make(map[string]*Route)
	rm.groups = make(map[string]*RouteGroup)
	rm.middleware = make(map[string]Handler)

	return nil
}

// ============================================================================
// Configuration Functions (moved from route_config.go)
// ============================================================================

// DefaultConfig returns a default routing configuration
func DefaultConfig() *Config {
	return &Config{
		Version:     "v1",
		Title:       "Espyna API",
		Description: "Framework-agnostic API routing",
		BasePath:    "/api",
		CORS: CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
			MaxAge:           86400,
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerMinute: 100,
			BurstSize:         20,
			Endpoints:         make(map[string]EndpointRateLimit),
		},
		Timeout:           30 * time.Second,
		EnableMetrics:     true,
		EnableHealthCheck: true,
		EnableAuth:        true,
		EnableAuditLog:    true,
		Domains: map[string]DomainConfig{
			"entity": {
				Enabled: true,
				Prefix:  "/entity",
				Endpoints: map[string]EndpointConfig{
					"admin": {
						Enabled: true,
						Methods: []string{"POST"},
						Auth: AuthConfig{
							Required: true,
							Roles:    []string{"admin"},
						},
					},
				},
			},
			"event": {
				Enabled: true,
				Prefix:  "/event",
			},
			"payment": {
				Enabled: true,
				Prefix:  "/payment",
			},
			"product": {
				Enabled: true,
				Prefix:  "/product",
			},
			"record": {
				Enabled: true,
				Prefix:  "/record",
			},
			"subscription": {
				Enabled: true,
				Prefix:  "/subscription",
			},
		},
		MigrationConfig: MigrationConfig{
			EnableLegacyRoutes: true,
			Strategy:           "parallel",
			RouteMappings:      make(map[string]string),
			TrafficSplitting: TrafficSplitConfig{
				Enabled:          false,
				NewSystemPercent: 0,
				Rules:            []SplitRule{},
			},
		},
	}
}

// NOTE: Config methods (Validate, GetEnabledDomains, IsEndpointEnabled) are
// now defined in contracts/config.go and available via the type alias.
