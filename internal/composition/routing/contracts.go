package routing

import (
	"context"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ============================================================================
// Type Aliases from contracts (for backward compatibility)
// These types are defined in contracts/ and re-exported here for consumers
// who import routing/ directly.
// ============================================================================

// Handler is an alias for contracts.RouteHandler - the framework-agnostic handler interface
type Handler = contracts.RouteHandler

// Request is an alias for contracts.Request - framework-agnostic HTTP request
type Request = contracts.Request

// Response is an alias for contracts.Response - framework-agnostic HTTP response
type Response = contracts.Response

// Route is an alias for contracts.Route - framework-agnostic route definition
type Route = contracts.Route

// LeapforCustomRoute is an exported alias for Route to allow consumers to customize routes
type LeapforCustomRoute = contracts.Route

// RouteMetadata is an alias for contracts.RouteMetadata
type RouteMetadata = contracts.RouteMetadata

// RouteGroup is an alias for contracts.RouteGroup
type RouteGroup = contracts.RouteGroup

// GroupMetadata is an alias for contracts.GroupMetadata
type GroupMetadata = contracts.GroupMetadata

// Config is an alias for contracts.Config - routing configuration
type Config = contracts.Config

// CORSConfig is an alias for contracts.CORSConfig
type CORSConfig = contracts.CORSConfig

// RateLimitConfig is an alias for contracts.RateLimitConfig
type RateLimitConfig = contracts.RateLimitConfig

// EndpointRateLimit is an alias for contracts.EndpointRateLimit
type EndpointRateLimit = contracts.EndpointRateLimit

// DomainConfig is an alias for contracts.DomainConfig
type DomainConfig = contracts.DomainConfig

// EndpointConfig is an alias for contracts.EndpointConfig
type EndpointConfig = contracts.EndpointConfig

// AuthConfig is an alias for contracts.AuthConfig
type AuthConfig = contracts.AuthConfig

// CacheConfig is an alias for contracts.CacheConfig
type CacheConfig = contracts.CacheConfig

// ValidationConfig is an alias for contracts.ValidationConfig
type ValidationConfig = contracts.ValidationConfig

// MigrationConfig is an alias for contracts.MigrationConfig
type MigrationConfig = contracts.MigrationConfig

// TrafficSplitConfig is an alias for contracts.TrafficSplitConfig
type TrafficSplitConfig = contracts.TrafficSplitConfig

// SplitRule is an alias for contracts.SplitRule
type SplitRule = contracts.SplitRule

// SubscriptionRouteDeclaration is an alias for contracts.SubscriptionRouteDeclaration
type SubscriptionRouteDeclaration = contracts.SubscriptionRouteDeclaration

// SubscriptionRouteHandler is an alias for contracts.SubscriptionRouteHandler
type SubscriptionRouteHandler = contracts.SubscriptionRouteHandler

// ============================================================================
// Builder Types
// ============================================================================

// RouteBuilder provides a fluent interface for building routes
type RouteBuilder struct {
	route *Route
}

// GroupBuilder provides a fluent interface for building route groups
type GroupBuilder struct {
	group *RouteGroup
}

// ============================================================================
// Builder Functions & Methods
// ============================================================================

// NewRouteBuilder creates a new route builder
func NewRouteBuilder(method, path string) *RouteBuilder {
	return &RouteBuilder{
		route: &Route{
			Method:     method,
			Path:       path,
			Middleware: []Handler{},
			Metadata:   RouteMetadata{},
		},
	}
}

// Handler sets the handler for the route
func (rb *RouteBuilder) Handler(handler Handler) *RouteBuilder {
	rb.route.Handler = handler
	return rb
}

// UseCase sets a use case executor as the handler
func (rb *RouteBuilder) UseCase(executor interface{}) *RouteBuilder {
	// For now, just set the executor directly as the handler
	// TODO: Implement proper generic handler wrapping
	if handler, ok := executor.(Handler); ok {
		rb.route.Handler = handler
	}
	return rb
}

// Middleware adds middleware to the route
func (rb *RouteBuilder) Middleware(middleware ...Handler) *RouteBuilder {
	rb.route.Middleware = append(rb.route.Middleware, middleware...)
	return rb
}

// Metadata sets the route metadata
func (rb *RouteBuilder) Metadata(metadata RouteMetadata) *RouteBuilder {
	rb.route.Metadata = metadata
	return rb
}

// Build creates the final route
func (rb *RouteBuilder) Build() *Route {
	return rb.route
}

// NewGroupBuilder creates a new group builder
func NewGroupBuilder(prefix string) *GroupBuilder {
	return &GroupBuilder{
		group: &RouteGroup{
			Prefix:     prefix,
			Routes:     []*Route{},
			SubGroups:  []*RouteGroup{},
			Middleware: []Handler{},
			Metadata:   GroupMetadata{},
		},
	}
}

// Route adds a route to the group
func (gb *GroupBuilder) Route(method, path string, handler Handler) *RouteBuilder {
	route := &Route{
		Method:     method,
		Path:       path,
		Handler:    handler,
		Middleware: []Handler{},
		Metadata:   RouteMetadata{},
	}

	gb.group.Routes = append(gb.group.Routes, route)
	return NewRouteBuilder(method, path).Handler(handler)
}

// UseCase adds a use case route to the group
func (gb *GroupBuilder) UseCase(method, path string, executor interface{}) *RouteBuilder {
	// For now, just set the executor directly as the handler
	// TODO: Implement proper generic handler wrapping
	if handler, ok := executor.(Handler); ok {
		return gb.Route(method, path, handler)
	}
	return gb.Route(method, path, nil)
}

// SubGroup adds a subgroup to the group
func (gb *GroupBuilder) SubGroup(prefix string) *GroupBuilder {
	subGroup := NewGroupBuilder(prefix)
	gb.group.SubGroups = append(gb.group.SubGroups, subGroup.group)
	return subGroup
}

// Middleware adds middleware to the group
func (gb *GroupBuilder) Middleware(middleware ...Handler) *GroupBuilder {
	gb.group.Middleware = append(gb.group.Middleware, middleware...)
	return gb
}

// Metadata sets the group metadata
func (gb *GroupBuilder) Metadata(metadata GroupMetadata) *GroupBuilder {
	gb.group.Metadata = metadata
	return gb
}

// Build creates the final route group
func (gb *GroupBuilder) Build() *RouteGroup {
	return gb.group
}

// ============================================================================
// Management Types (routing-specific, have mutable state)
// ============================================================================

// RouteManager manages framework-agnostic routes and provides a unified interface
// for registering, organizing, and retrieving routes
type RouteManager struct {
	config     *Config
	routes     map[string]*Route
	groups     map[string]*RouteGroup
	middleware map[string]Handler
	mu         sync.RWMutex
}

// ============================================================================
// Migration Types (routing-specific, have mutable state)
// ============================================================================

// MigrationManager handles the gradual migration from legacy routing to the new system
type MigrationManager struct {
	routeManager *RouteManager
	config       *MigrationConfig

	// Traffic splitting
	trafficSplitter *TrafficSplitter

	// Route mapping
	routeMapper *RouteMapper

	// Metrics
	metrics *MigrationMetrics

	mu sync.RWMutex
}

// TrafficSplitter handles traffic splitting between old and new systems
type TrafficSplitter struct {
	config TrafficSplitConfig
	rules  []SplitRule

	// Runtime state
	sessionStore map[string]SessionInfo
	mu           sync.RWMutex
}

// SessionInfo tracks routing decisions for a session
type SessionInfo struct {
	UseNewSystem bool
	ExpiresAt    time.Time
	Reason       string
}

// RouteMapper handles route mapping between old and new systems
type RouteMapper struct {
	mappings map[string]string // old_route -> new_route
	reverse  map[string]string // new_route -> old_route
}

// MigrationMetrics tracks migration metrics
type MigrationMetrics struct {
	RequestsTotal  int64
	RequestsLegacy int64
	RequestsNew    int64
	ErrorsLegacy   int64
	ErrorsNew      int64
	LatencyLegacy  time.Duration
	LatencyNew     time.Duration

	// Per-route metrics
	RouteMetrics map[string]*RouteMetrics

	mu sync.RWMutex
}

// RouteMetrics tracks metrics for a specific route
type RouteMetrics struct {
	RequestsTotal  int64
	RequestsLegacy int64
	RequestsNew    int64
	ErrorsLegacy   int64
	ErrorsNew      int64
	LatencyLegacy  time.Duration
	LatencyNew     time.Duration
}

// ============================================================================
// Composer Types (routing-specific, imports application layer)
// ============================================================================

// Composer orchestrates the entire routing system
type Composer struct {
	config       *Config
	routeManager *RouteManager
	migrationMgr *MigrationManager
	// Container for dependencies
	container interface{}

	// Use cases for direct route configuration
	useCases *usecases.Aggregate
}

// ComposerConfig represents the composer configuration
type ComposerConfig struct {
	Config    *Config
	Container interface{}
}

// HandlerAdapter adapts domain handlers to the routing.Handler interface
type HandlerAdapter struct {
	entityHandler interface{} // handlers.EntityHandler
}

// RoutingRequestAdapter implements the handlers.RoutingRequest interface
type RoutingRequestAdapter struct {
	request *Request
}

// handlerFunc implements Handler interface for functions
type handlerFunc struct {
	fn func(context.Context, *Request) (*Response, error)
}
