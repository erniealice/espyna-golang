package contracts

import (
	"sync"
	"time"
)

// ============================================================================
// Route Management Types
// ============================================================================

// RouteManager manages framework-agnostic routes and provides a unified interface
// for registering, organizing, and retrieving routes
type RouteManager struct {
	Config     *Config
	Routes     map[string]*Route
	Groups     map[string]*RouteGroup
	Middleware map[string]RouteHandler
	mu         sync.RWMutex
}

// MigrationManager handles the gradual migration from legacy routing to the new system
type MigrationManager struct {
	RouteManager *RouteManager
	Config       *MigrationConfig

	// Traffic splitting
	TrafficSplitter *TrafficSplitter

	// Route mapping
	RouteMapper *RouteMapper

	// Metrics
	Metrics *MigrationMetrics

	mu sync.RWMutex
}

// TrafficSplitter handles traffic splitting between old and new systems
type TrafficSplitter struct {
	Config TrafficSplitConfig
	Rules  []SplitRule

	// Runtime state
	SessionStore map[string]SessionInfo
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
	Mappings map[string]string // old_route -> new_route
	Reverse  map[string]string // new_route -> old_route
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
// Composer Config Types
// ============================================================================

// ComposerConfig represents the composer configuration
type ComposerConfig struct {
	Config    *Config
	Container interface{}
}
