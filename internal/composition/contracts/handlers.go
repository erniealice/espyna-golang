package contracts

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ============================================================================
// Core Use Case Interfaces
// ============================================================================

// UseCaseExecutor defines the interface that all use case Execute methods implement
// Request and Response must be protobuf messages
type UseCaseExecutor[Request proto.Message, Response proto.Message] interface {
	Execute(ctx context.Context, req Request) (Response, error)
}

// UseCaseHandler represents the generic handler interface for configuration
// This interface uses proto.Message to ensure type safety with protobuf types
type UseCaseHandler interface {
	Execute(ctx context.Context, req proto.Message) (proto.Message, error)
}

// ProtobufParser defines the interface for handlers that can parse protobuf data
type ProtobufParser interface {
	UseCaseHandler
	ParseRequestFromJSON(jsonData []byte) (proto.Message, error)
}

// ============================================================================
// Generic Handler Implementation
// ============================================================================

// GenericHandler wraps a use case executor to match the Handler interface
// Request and Response are constrained to proto.Message types
type GenericHandler[Request proto.Message, Response proto.Message] struct {
	executor         UseCaseExecutor[Request, Response]
	requestPrototype Request
}

// NewGenericHandler creates a type-safe handler wrapper for any use case
// requestPrototype is used as a template for creating new request instances during JSON parsing
func NewGenericHandler[Request proto.Message, Response proto.Message](
	executor UseCaseExecutor[Request, Response],
	requestPrototype Request,
) *GenericHandler[Request, Response] {
	return &GenericHandler[Request, Response]{
		executor:         executor,
		requestPrototype: requestPrototype,
	}
}

// Execute implements the Handler interface by delegating to the typed use case
func (h *GenericHandler[Request, Response]) Execute(ctx context.Context, req proto.Message) (proto.Message, error) {
	// Type assert the request to the specific protobuf type
	typedReq, ok := req.(Request)
	if !ok {
		return nil, fmt.Errorf("invalid request type for use case: expected %T, got %T", *new(Request), req)
	}

	// Call the typed use case Execute method and capture both response and error
	response, err := h.executor.Execute(ctx, typedReq)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ParseRequestFromJSON parses JSON data into the appropriate Request type using protojson
func (h *GenericHandler[Request, Response]) ParseRequestFromJSON(jsonData []byte) (proto.Message, error) {
	// Clone the prototype to get a fresh instance - this avoids reflection entirely
	req := proto.Clone(h.requestPrototype).(Request)

	// Parse JSON into the protobuf message
	if err := protojson.Unmarshal(jsonData, req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON into protobuf %T: %w", req, err)
	}

	return req, nil
}

// ============================================================================
// Builder Pattern Types
// ============================================================================

// RouteBuilder provides a fluent interface for building routes
type RouteBuilder struct {
	route *Route
}

// GroupBuilder provides a fluent interface for building route groups
type GroupBuilder struct {
	group *RouteGroup
}

// NewRouteBuilder creates a new route builder
func NewRouteBuilder(method, path string) *RouteBuilder {
	return &RouteBuilder{
		route: &Route{
			Method:     method,
			Path:       path,
			Middleware: []RouteHandler{},
			Metadata:   RouteMetadata{},
		},
	}
}

// Handler sets the handler for the route
func (rb *RouteBuilder) Handler(handler RouteHandler) *RouteBuilder {
	rb.route.Handler = handler
	return rb
}

// UseCase sets a use case executor as the handler
func (rb *RouteBuilder) UseCase(executor interface{}) *RouteBuilder {
	// For now, just set the executor directly as the handler
	// TODO: Implement proper generic handler wrapping
	if handler, ok := executor.(RouteHandler); ok {
		rb.route.Handler = handler
	}
	return rb
}

// Middleware adds middleware to the route
func (rb *RouteBuilder) Middleware(middleware ...RouteHandler) *RouteBuilder {
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
			Middleware: []RouteHandler{},
			Metadata:   GroupMetadata{},
		},
	}
}

// Route adds a route to the group
func (gb *GroupBuilder) Route(method, path string, handler RouteHandler) *RouteBuilder {
	route := &Route{
		Method:     method,
		Path:       path,
		Handler:    handler,
		Middleware: []RouteHandler{},
		Metadata:   RouteMetadata{},
	}

	gb.group.Routes = append(gb.group.Routes, route)
	return NewRouteBuilder(method, path).Handler(handler)
}

// UseCase adds a use case route to the group
func (gb *GroupBuilder) UseCase(method, path string, executor interface{}) *RouteBuilder {
	// For now, just set the executor directly as the handler
	// TODO: Implement proper generic handler wrapping
	if handler, ok := executor.(RouteHandler); ok {
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
func (gb *GroupBuilder) Middleware(middleware ...RouteHandler) *GroupBuilder {
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
