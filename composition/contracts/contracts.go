// Package contracts re-exports internal composition contract types for use by contrib sub-modules.
// Contrib packages (which are separate Go modules) cannot import internal/ directly,
// so this package provides stable public aliases.
package contracts

import (
	internal "github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// =============================================================================
// Core Handler Interfaces
// =============================================================================

// UseCaseHandler represents the generic handler interface for route execution.
type UseCaseHandler = internal.UseCaseHandler

// ProtobufParser defines the interface for handlers that can parse protobuf data.
type ProtobufParser = internal.ProtobufParser

// RouteHandler defines the framework-agnostic handler interface.
type RouteHandler = internal.RouteHandler

// =============================================================================
// Route Types
// =============================================================================

// Route represents a framework-agnostic route definition.
type Route = internal.Route

// RouteMetadata contains additional information about a route.
type RouteMetadata = internal.RouteMetadata

// RouteGroup represents a collection of related routes.
type RouteGroup = internal.RouteGroup

// GroupMetadata contains information about a route group.
type GroupMetadata = internal.GroupMetadata

// =============================================================================
// HTTP Request/Response Types
// =============================================================================

// Request represents a framework-agnostic HTTP request.
type Request = internal.Request

// Response represents a framework-agnostic HTTP response.
type Response = internal.Response

// =============================================================================
// Configuration Types
// =============================================================================

// Config holds routing configuration.
type Config = internal.Config

// CORSConfig holds CORS configuration.
type CORSConfig = internal.CORSConfig

// =============================================================================
// Service / Provider Interfaces
// =============================================================================

// Service is the base interface for infrastructure services.
type Service = internal.Service

// Provider is the base interface for infrastructure providers.
type Provider = internal.Provider
