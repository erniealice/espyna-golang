package customization

import (
	"strings"

	"github.com/erniealice/espyna-golang/internal/composition/routing"
)

// NewRouteCustomizer creates a new route customizer
func NewRouteCustomizer() *RouteCustomizer {
	return &RouteCustomizer{
		domainPrefixes: make(map[string]string),
		routePaths:     make(map[string]string),
	}
}

// Builder pattern methods

// WithGlobalPrefix sets a global prefix for all routes
func (rc *RouteCustomizer) WithGlobalPrefix(prefix string) *RouteCustomizer {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.globalPrefix = strings.TrimSuffix(prefix, "/")
	return rc
}

// WithDomainPrefix sets a custom prefix for all routes in a domain
func (rc *RouteCustomizer) WithDomainPrefix(domain, prefix string) *RouteCustomizer {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.domainPrefixes[domain] = strings.TrimSuffix(prefix, "/")
	return rc
}

// WithRoutePath sets a custom path for a specific route by name
func (rc *RouteCustomizer) WithRoutePath(routeName, path string) *RouteCustomizer {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.routePaths[routeName] = path
	return rc
}

// ApplyCustomizations applies all customizations to a slice of routes
// Returns new route instances with modified paths (originals unchanged)
func (rc *RouteCustomizer) ApplyCustomizations(routes []*routing.Route) []*routing.Route {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	customized := make([]*routing.Route, len(routes))

	for i, route := range routes {
		// Create a copy of the route
		customized[i] = &routing.Route{
			Method:     route.Method,
			Path:       rc.customizePath(route),
			Handler:    route.Handler, // Handler unchanged
			Middleware: route.Middleware,
			Metadata:   route.Metadata,
		}
	}

	return customized
}

// customizePath determines the final path for a route based on customizations
func (rc *RouteCustomizer) customizePath(route *routing.Route) string {
	// Priority 1: Specific route path override
	if customPath, ok := rc.routePaths[route.Metadata.Name]; ok {
		return rc.applyGlobalPrefix(customPath)
	}

	// Priority 2: Domain prefix override
	if domainPrefix, ok := rc.domainPrefixes[route.Metadata.Domain]; ok {
		return rc.applyGlobalPrefix(rc.replaceDomainInPath(route.Path, domainPrefix))
	}

	// Priority 3: Global prefix only
	if rc.globalPrefix != "" {
		return rc.applyGlobalPrefix(route.Path)
	}

	// No customization - return original path
	return route.Path
}

// applyGlobalPrefix applies global prefix to a path
func (rc *RouteCustomizer) applyGlobalPrefix(path string) string {
	if rc.globalPrefix == "" {
		return path
	}
	return rc.globalPrefix + "/" + strings.TrimPrefix(path, "/")
}

// replaceDomainInPath replaces the domain portion of the path
func (rc *RouteCustomizer) replaceDomainInPath(originalPath, domainPrefix string) string {
	// Replace the domain portion of the path
	// "/api/workflow/..." -> "{domainPrefix}/..."
	parts := strings.Split(strings.Trim(originalPath, "/"), "/")
	if len(parts) >= 2 {
		// Replace domain part (assuming format: /api/{domain}/...)
		parts[1] = strings.Trim(domainPrefix, "/")
		return "/" + strings.Join(parts, "/")
	}
	return originalPath
}
