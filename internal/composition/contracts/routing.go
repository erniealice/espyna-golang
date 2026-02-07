package contracts

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// ============================================================================
// Core Routing Interfaces
// ============================================================================

// RouteHandler defines the framework-agnostic handler interface - matches use case Execute signature
type RouteHandler interface {
	Execute(ctx context.Context, request proto.Message) (proto.Message, error)
}

// ============================================================================
// HTTP Request/Response Types
// ============================================================================

// Request represents a framework-agnostic HTTP request
type Request struct {
	Method      string
	Path        string
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	Body        []byte
	Context     context.Context
}

// Response represents a framework-agnostic HTTP response
type Response struct {
	Data    interface{}
	Status  int
	Headers map[string]string
}

// ============================================================================
// Route Definition Types
// ============================================================================

// Route represents a framework-agnostic route definition
type Route struct {
	Method     string
	Path       string
	Handler    RouteHandler
	Middleware []RouteHandler
	Metadata   RouteMetadata
}

// LeapforCustomRoute is an exported alias for Route to allow consumers to customize routes
// This type enables consumer applications to create and register custom routes
type LeapforCustomRoute = Route

// RouteMetadata contains additional information about a route
type RouteMetadata struct {
	Name        string   // Unique route identifier (auto-generated)
	Domain      string   // e.g., "entity", "event", "framework"
	Resource    string   // e.g., "admin", "client", "user"
	Operation   string   // e.g., "create", "read", "update", "delete", "list"
	Description string   // Human-readable description
	Tags        []string // e.g., ["admin", "public", "internal"]
	Version     string   // API version
	Deprecated  bool     // Whether the route is deprecated
}

// ============================================================================
// Route Group Types
// ============================================================================

// RouteGroup represents a collection of related routes
type RouteGroup struct {
	Prefix     string
	Routes     []*Route
	SubGroups  []*RouteGroup
	Middleware []RouteHandler
	Metadata   GroupMetadata
}

// GroupMetadata contains information about a route group
type GroupMetadata struct {
	Name        string
	Description string
	Version     string
}

// ============================================================================
// Domain Configuration Types
// ============================================================================

// DomainRouteConfiguration defines all routes for a specific domain
type DomainRouteConfiguration struct {
	Domain  string
	Prefix  string
	Enabled bool
	Routes  []RouteConfiguration
}

// RouteConfiguration defines how a route should be configured
type RouteConfiguration struct {
	Method  string
	Path    string
	Handler UseCaseHandler // Direct use case handler
}

// ============================================================================
// Subscription Route Types
// ============================================================================

// SubscriptionRouteDeclaration represents a single subscription route declaration
type SubscriptionRouteDeclaration struct {
	Method  string
	Path    string
	Handler SubscriptionRouteHandler
}

// SubscriptionRouteHandler defines a function signature for subscription route handlers
type SubscriptionRouteHandler func(ctx context.Context, request *Request) (*Response, error)
